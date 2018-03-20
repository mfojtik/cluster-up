package preflight

import (
	"fmt"

	"github.com/docker/docker/api/types/versions"
	"github.com/mfojtik/cluster-up/pkg/api"
	"github.com/mfojtik/cluster-up/pkg/log"
)

type DockerVersion struct {
	validatorContext
}

func (d *DockerVersion) Message() string {
	return "Checking if Docker version is >= " + api.MinSupportedDockerVersion
}

func (d *DockerVersion) Validate() error {
	version, err := d.ContainerClient().ServerVersion()
	if err != nil {
		return log.Error("server version", err)
	}
	if versions.LessThan(version.APIVersion, api.MinSupportedDockerVersion) {
		return fmt.Errorf("insufficient Docker version, required >=%s, have %s", api.MinSupportedDockerVersion, version.APIVersion)
	}
	return nil
}
