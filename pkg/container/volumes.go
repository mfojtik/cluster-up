package container

import (
	"fmt"
	"os"
	"path"

	"github.com/mfojtik/cluster-up/pkg/api"
	"github.com/mfojtik/cluster-up/pkg/util/dir"
)

const (
	nonLinuxBaseDir = "/var/lib/origin/volumes"

	cmdTestNsenterMount  = "nsenter --mount=/rootfs/proc/1/ns/mnt findmnt"
	ensureVolumeShareCmd = `#/bin/bash
set -x
nsenter --mount=/rootfs/proc/1/ns/mnt mkdir -p %[1]s
grep -F %[1]s /rootfs/proc/1/mountinfo || nsenter --mount=/rootfs/proc/1/ns/mnt mount -o bind %[1]s %[1]s
grep -F %[1]s /rootfs/proc/1/mountinfo | grep shared || nsenter --mount=/rootfs/proc/1/ns/mnt mount --make-shared %[1]s
`
)

type VolumesConfig struct {
	baseDir         string
	dockerClient    Client
	useNSEnterMount bool
}

func BuildHostVolumesConfig(dockerClient Client, baseDir string) (*VolumesConfig, error) {
	c := &VolumesConfig{
		dockerClient: dockerClient,
	}
	if len(baseDir) == 0 {
		baseDir = dir.InOpenShiftLocal("cluster-up")
	}
	c.baseDir = baseDir
	if !path.IsAbs(baseDir) {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		absHostDir, err := dir.MakeAbs(baseDir, cwd)
		if err != nil {
			return nil, err
		}
		c.baseDir = absHostDir
	}
	var err error
	c.useNSEnterMount, err = c.hasNSEnterSupport()
	if err != nil {
		return nil, err
	}
	return c, c.makeDirectories()
}

func (c *VolumesConfig) BaseDir() string {
	return c.baseDir
}

func (c *VolumesConfig) HostEtcdDir() string {
	return path.Join(c.BaseDir(), dir.InOpenShiftLocal("etcd"))
}

func (c *VolumesConfig) HostPersistentVolumesDir() string {
	return path.Join(c.BaseDir(), dir.InOpenShiftLocal("pv"))
}

func (c *VolumesConfig) HostVolumesDir() string {
	d := path.Join(c.BaseDir(), dir.InOpenShiftLocal("volumes"))
	if c.useNSEnterMount {
		return d
	}
	return path.Join(nonLinuxBaseDir, d)
}

func (c *VolumesConfig) makeDirectories() error {
	if c.useNSEnterMount {
		if err := os.MkdirAll(c.HostVolumesDir(), 0755); err != nil {
			return err
		}
	} else {
		if err := c.ensureSharedHostVolumes(); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(c.HostEtcdDir(), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(c.HostPersistentVolumesDir(), 0755); err != nil {
		return err
	}
	return nil
}

func (c *VolumesConfig) ensureSharedHostVolumes() error {
	return Docker(c.dockerClient, c.BaseDir()).
		Discard().
		Privileged().
		MountRootFS().
		Entrypoint("/bin/bash").
		Command("-c", fmt.Sprintf(ensureVolumeShareCmd, c.HostVolumesDir())).
		Name("create-shared-volumes").
		Run(api.OriginImage()).Error()
}

func (c *VolumesConfig) hasNSEnterSupport() (bool, error) {
	ok, err := c.isRedhatDocker()
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	cmd := Docker(c.dockerClient, c.BaseDir()).
		Discard().
		Privileged().
		MountRootFS().
		Entrypoint("/bin/bash").
		Command("-c", cmdTestNsenterMount).
		Name("test-nsenter-support").
		Run(api.OriginImage())
	if cmd.Error() != nil {
		return false, cmd.Error()
	}
	return true, nil
}

func (c *VolumesConfig) isRedhatDocker() (bool, error) {
	info, err := c.dockerClient.Info()
	if err != nil {
		return false, err
	}
	kernelVersion := info.KernelVersion
	if len(kernelVersion) == 0 {
		return false, nil
	}
	return fedoraPackage.MatchString(kernelVersion) || rhelPackage.MatchString(kernelVersion), nil
}
