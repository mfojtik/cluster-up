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

func (o *OpenShiftRunning) Message() string {
	return fmt.Sprintf("Checking if OpenShift %q container is running", api.ContainerNameOrigin)
}

func (o *OpenShiftRunning) Validate() error {
	c, err := o.ContainerClient().ContainerInspect(api.ContainerNameOrigin)
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
			log.Error(fmt.Sprintf("removing %q container failer", api.ContainerNameOrigin), err)
		}
		return nil
	}
	return fmt.Errorf("found existing running container %q", api.ContainerNameOrigin)
}
