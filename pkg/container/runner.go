package container

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/mfojtik/cluster-up/pkg/log"
)

type Runner interface {
	RemoveWhenExit() Runner
	Privileged() Runner
	HostPID() Runner
	HostNetwork() Runner
	Bind(binds ...string) Runner
	Entrypoint(cmd ...string) Runner
	Command(cmd ...string) Runner

	RunImageWithName(image, name string) Runner
	Error() error
	CombinedOutput() []byte
}

type runner struct {
	client Client

	hostConfig *container.HostConfig
	config     *container.Config

	err         error
	containerID string
	output      []byte
}

func NewRunner(c Client) Runner {
	return &runner{
		client:     c,
		hostConfig: &container.HostConfig{},
		config:     &container.Config{},
	}
}

func (r *runner) RemoveWhenExit() Runner {
	r.hostConfig.AutoRemove = true
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

func (r *runner) RunImageWithName(image, name string) Runner {
	r.config.Image = image
	response, err := r.client.ContainerCreate(r.config, r.hostConfig, nil, name)
	if err != nil {
		r.err = log.Error(fmt.Sprintf("container %q (%q) failed to run", name, image), err)
	}
	for _, w := range response.Warnings {
		log.Infof("Container %q produced warning: %s", name, w)
	}
	r.containerID = response.ID

	attachOpts := types.ContainerAttachOptions{
		Stream: true,
		Stdin:  false,
		Stdout: true,
		Stderr: true,
	}
	attachResponse, err := r.client.ContainerAttach(r.containerID, attachOpts)
	defer attachResponse.Close()
	attachReader := bufio.NewReader(attachResponse.Reader)
	go func() {
		for {
			out, err := attachReader.ReadBytes('\n')
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Error("failed to read container logs", err)
				break
			}
			r.output = append(r.output, out...)
		}
	}()

	log.Debugf("Starting container %q (%s) entrypoint: %q, command: %q...",
		name, image, strings.Join(r.config.Entrypoint, " "), strings.Join(r.config.Cmd, " "))

	startTime := time.Now()
	if err := r.client.ContainerStart(r.containerID, types.ContainerStartOptions{}); err != nil {
		r.err = log.Error(fmt.Sprintf("failed to start container %q", name), err)
		return r
	}
	waitC, errC := r.client.ContainerWait(response.ID, container.WaitConditionRemoved)
	select {
	case err := <-errC:
		r.err = log.Error(fmt.Sprintf("container %q (%q) failed to finish", name, image), err)
	case status := <-waitC:
		if status.StatusCode != 0 {
			r.err = fmt.Errorf("%s\ncontainer %q exited with %d", string(r.CombinedOutput()), name, status.StatusCode)
		}
		log.Debugf("Container %q (%s) finished, took %s, returned %d", name, image, time.Since(startTime), status.StatusCode)
		// TODO: Add timeout
	}
	return r
}

func (r *runner) Error() error {
	return r.err
}

func (r *runner) CombinedOutput() []byte {
	return r.output
}
