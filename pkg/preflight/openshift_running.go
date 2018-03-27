package preflight

import (
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/mfojtik/cluster-up/pkg/api"
	"github.com/mfojtik/cluster-up/pkg/log"
)

type OpenShiftRunning struct {
	validatorContext
}

// List known containers we want to check the existence and remove prior to cluster up
// operations.
var containerToCheck = []string{
	api.ContainerNameOrigin,
}

func (o *OpenShiftRunning) Message() string {
	return "Checking for existing OpenShift containers"
}

func (o *OpenShiftRunning) Validate() error {
	for _, name := range containerToCheck {
		err := o.validateContainerByName(name)
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *OpenShiftRunning) validateContainerByName(containerName string) error {
	c, err := o.ContainerClient().ContainerInspect(containerName)
	if err != nil {
		if client.IsErrNotFound(err) {
			return nil
		} else {
			return log.Error("container inspect result", err)
		}
	}
	if c.State != nil && !c.State.Running {
		log.Debugf("Found %q container in %q state, attempting to remove", c.ID, c.State.Status)
		err := o.ContainerClient().ContainerRemove(c.ID, types.ContainerRemoveOptions{
			Force: true,
		})
		if err != nil {
			log.Error(fmt.Sprintf("removing %q container failed", containerName), err)
		}
		return nil
	}
	return fmt.Errorf("found existing running container %q", containerName)
}
