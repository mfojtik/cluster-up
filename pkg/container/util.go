package container

import (
	"regexp"

	"github.com/mfojtik/cluster-up/pkg/api"
)

const (
	cmdTestNsenterMount = "nsenter --mount=/rootfs/proc/1/ns/mnt findmnt"

	ensureVolumeShareCmd = `#/bin/bash
set -x
nsenter --mount=/rootfs/proc/1/ns/mnt mkdir -p %[1]s
grep -F %[1]s /rootfs/proc/1/mountinfo || nsenter --mount=/rootfs/proc/1/ns/mnt mount -o bind %[1]s %[1]s
grep -F %[1]s /rootfs/proc/1/mountinfo | grep shared || nsenter --mount=/rootfs/proc/1/ns/mnt mount --make-shared %[1]s
`
)

var (
	fedoraPackage = regexp.MustCompile("\\.fc[0-9_]*\\.")
	rhelPackage   = regexp.MustCompile("\\.el[0-9_]*\\.")
)

func CanUseNSenterMounter(c Client) (bool, error) {
	cmd := NewRunner(c).
		RemoveWhenExit().
		Privileged().
		Bind("/:/rootfs:ro").
		Entrypoint("/bin/bash").
		Command("-c", cmdTestNsenterMount).
		RunImageWithName(api.DefaultImagePrefix+"origin:latest", "test-nsenter")
	if cmd.Error() != nil {
		return false, cmd.Error()
	}
	return true, nil
}

func IsRedhatDocker(c Client) (bool, error) {
	info, err := c.Info()
	if err != nil {
		return false, err
	}
	kernelVersion := info.KernelVersion
	if len(kernelVersion) == 0 {
		return false, nil
	}
	return fedoraPackage.MatchString(kernelVersion) || rhelPackage.MatchString(kernelVersion), nil
}

func UserNamespaceEnabled(c Client) (bool, error) {
	info, err := c.Info()
	if err != nil {
		return false, err
	}
	for _, val := range info.SecurityOptions {
		if val == "name=userns" {
			return true, nil
		}
	}
	return false, nil
}
