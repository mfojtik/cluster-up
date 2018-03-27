package container

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/mfojtik/cluster-up/pkg/log"
)

type HookFn func(containerID string) error

type Runner interface {
	// Discard will cause the container to be removed after it exit
	Discard() Runner

	// Privileged will make the container privileged
	Privileged() Runner

	// HostPID will enable the container to see the host PID
	HostPID() Runner

	// HostNetwork will enable the container to bind on host network interfaces
	HostNetwork() Runner

	// Binds define the container bind mounts from the host
	Bind(binds ...string) Runner

	// MountRootFS will bind mount root / into /rootfs inside container
	MountRootFS() Runner

	// OnBackground disable waiting for the container to finish
	// If this is set the OnExit hook will panic if used.
	OnBackground() Runner

	// OnStart allows to execute something after the container was started.
	OnStart(fn HookFn) Runner

	// OnExit allows to execute something after the container finished.
	OnExit(fn HookFn) Runner

	// Entrypoint is the container ENTRYPOINT
	Entrypoint(cmd ...string) Runner

	// Command is the container CMD
	Command(cmd ...string) Runner

	// Run will run the container based on the provided image and the
	// container name.
	Run(image, name string) Runner

	// Error return errors when they occur.
	Error() error

	// Output and ErrorOutput will return the container stdout and stderr.
	Output() []byte
	ErrorOutput() []byte
}

type runner struct {
	client Client

	hostConfig *container.HostConfig
	config     *container.Config

	onStartHooks []HookFn
	onExitHooks  []HookFn
	background   bool
	err          error
	containerID  string
	output       []byte
	outputErr    []byte
	baseDir      string
}

func NewRunner(c Client, baseDir string) Runner {
	return &runner{
		client:     c,
		baseDir:    baseDir,
		hostConfig: &container.HostConfig{},
		config:     &container.Config{},
	}
}

func (r *runner) OnBackground() Runner {
	if len(r.onExitHooks) > 0 {
		panic("cannot run on background with exit hooks defined")
	}
	r.background = true
	return r
}
func (r *runner) OnStart(fn HookFn) Runner {
	r.onStartHooks = append(r.onStartHooks, fn)
	return r
}

func (r *runner) OnExit(fn HookFn) Runner {
	if r.background {
		panic("cannot use OnExit when running container in background")
	}
	r.onExitHooks = append(r.onExitHooks, fn)
	return r
}

func (r *runner) Discard() Runner {
	r.hostConfig.AutoRemove = true
	// TODO: This won't be needed in newer Docker version,
	// the AutoRemove should automatically remove...
	r.onExitHooks = append(r.onExitHooks, func(containerID string) error {
		log.Debugf("Removing container %q", r.containerID)
		return r.client.ContainerRemove(r.containerID, types.ContainerRemoveOptions{Force: true})
	})
	return r
}

func (r *runner) Privileged() Runner {
	r.hostConfig.Privileged = true
	hasUserNs, err := UserNamespaceEnabled(r.client)
	if err != nil {
		r.err = log.Error("unable to check user namespace support", err)
		return r
	}
	if hasUserNs {
		r.hostConfig.UsernsMode = "host"
	}
	return r
}

func (r *runner) HostPID() Runner {
	r.hostConfig.PidMode = "host"
	return r
}

func (r *runner) HostNetwork() Runner {
	r.hostConfig.NetworkMode = "host"
	return r
}

func (r *runner) Bind(binds ...string) Runner {
	r.hostConfig.Binds = append(r.hostConfig.Binds, binds...)
	return r
}

func (r *runner) Entrypoint(cmd ...string) Runner {
	r.config.Entrypoint = cmd
	return r
}

func (r *runner) Command(cmd ...string) Runner {
	r.config.Cmd = cmd
	return r
}

func (r *runner) MountRootFS() Runner {
	r.hostConfig.Binds = append(r.hostConfig.Binds, "/:/rootfs:ro")
	return r
}

func (r *runner) Run(image, name string) Runner {
	if len(r.containerID) != 0 {
		return r
	}
	r.config.Image = image
	response, err := r.client.ContainerCreate(r.config, r.hostConfig, nil, name)
	if err != nil {
		r.err = log.Error(fmt.Sprintf("container %q (%q) failed to run", name, image), err)
		return r
	}
	for _, w := range response.Warnings {
		log.Debugf("ContainerCreate() %q produced warning: %s", name, w)
	}
	r.containerID = response.ID
	defer r.runHooks(r.onExitHooks, r.containerID)

	// If we running container on background,
	// do not capture the stdout/err as the container will keep running when this
	// command finish. That will just give us a portion of the logs.
	if !r.background {
		attachOpts := types.ContainerAttachOptions{
			Stream: true,
			Stdout: true,
			Stderr: true,
		}
		attachResponse, err := r.client.ContainerAttach(r.containerID, attachOpts)
		if err != nil {
			r.err = log.Error("container attach", err)
			return r
		}
		defer attachResponse.Close()
		go r.captureContainerOutput(attachResponse.Reader)()
	}

	log.Debugf("Starting container %q (%s) id: %q, remove: %t, entrypoint: %q, "+
		"command: %q...",
		name, image, r.containerID, r.hostConfig.AutoRemove,
		strings.Join(r.config.Entrypoint, " "),
		strings.Join(r.config.Cmd, " "))

	startTime := time.Now()
	if err := r.client.ContainerStart(r.containerID, types.ContainerStartOptions{}); err != nil {
		r.err = log.Error(fmt.Sprintf("failed to start container %q", name), err)
		return r
	}
	if !r.background {
		r.OnExit(r.storeContainerLog)
	}
	if r.runHooks(r.onStartHooks, r.containerID); r.Error() != nil {
		return r
	}
	if !r.background {
		containerWaitChan := make(chan struct{})
		go func() {
			defer close(containerWaitChan)
			code, err := r.client.ContainerWait(r.containerID)
			if err != nil || code != 0 {
				if code != 0 {
					err = fmt.Errorf("non-zero exit code (%d)", code)
				}
				r.err = log.Error(fmt.Sprintf("container %q (%q) failed to finish (code %d)", name, image, code), err)
			}
		}()

		select {
		case <-containerWaitChan:
			log.Debugf("Container %q (%s) finished, took %s", name, image, time.Since(startTime))
		case <-time.After(1 * time.Minute):
			r.err = log.Error("container timeout", fmt.Errorf("container %q timeouted", name))
		}
		return r
	}
	log.Debugf("Container %q (%s) will run on background", name, image)
	return r
}

func (r *runner) Error() error {
	return r.err
}

func (r *runner) Output() []byte {
	if r.background {
		log.Debugf("Output() called for container that run in background")
		return nil
	}
	return bytes.TrimSpace(r.output)
}

func (r *runner) ErrorOutput() []byte {
	if r.background {
		log.Debugf("ErrorOutput() called for container that run in background")
		return nil
	}
	return bytes.TrimSpace(r.outputErr)
}

func (r *runner) storeContainerLog(name string) error {
	if len(r.baseDir) == 0 {
		return nil
	}
	if len(r.output) == 0 {
		return nil
	}
	logDir := path.Join(r.baseDir, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}
	filename := path.Join(logDir, fmt.Sprintf("%s.stdout.log", name))
	log.Debugf("Storing container %q stdout at %q", name, filename)
	if err := ioutil.WriteFile(filename, r.output, 0755); err != nil {
		return err
	}
	filename = path.Join(logDir, fmt.Sprintf("%s.stderr.log", name))
	log.Debugf("Storing container %q stderr at %q", name, filename)
	if err := ioutil.WriteFile(filename, r.outputErr, 0755); err != nil {
		return err
	}
	return nil
}

func (r *runner) runHooks(hooks []HookFn, containerID string) {
	log.Debugf("Running hooks for container %s", containerID)
	for _, hook := range hooks {
		if err := hook(containerID); err != nil {
			r.err = log.Error("hook failed", err)
			break
		}
	}
}

func (r *runner) captureContainerOutput(reader io.Reader) func() {
	actualStdout := new(bytes.Buffer)
	actualStderr := new(bytes.Buffer)
	return func() {
		_, err := stdcopy.StdCopy(actualStdout, actualStderr, reader)
		if err != nil {
			log.Error("reading container output failed: %v", err)
		}
		defer func() {
			for {
				line, err := actualStdout.ReadBytes('\n')
				if err == io.EOF {
					break
				}
				if err != nil {
					log.Error("failed to read stdout", err)
					break
				}
				r.output = append(r.output, line...)
			}
			for {
				line, err := actualStderr.ReadBytes('\n')
				if err == io.EOF {
					break
				}
				if err != nil {
					log.Error("failed to read stdout", err)
					break
				}
				r.outputErr = append(r.outputErr, line...)
			}
		}()
	}
}
