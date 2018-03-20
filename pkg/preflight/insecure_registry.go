package preflight

import (
	"fmt"
	"strings"

	"github.com/mfojtik/cluster-up/pkg/api"
	"github.com/mfojtik/cluster-up/pkg/log"
)

type DockerRegistry struct {
	validatorContext
}

func (d *DockerRegistry) Message() string {
	return "Checking insecure registry configuration has " + api.InsecureRegistryAddress
}

func (d *DockerRegistry) Validate() error {
	info, err := d.ContainerClient().Info()
	if err != nil {
		return log.Error("docker info", err)
	}
	var (
		found bool
		ips   []string
	)
	for _, r := range info.RegistryConfig.InsecureRegistryCIDRs {
		if strings.Contains(r.String(), api.InsecureRegistryAddress) {
			found = true
		}
		ips = append(ips, strings.TrimSuffix(strings.TrimPrefix(r.String(), "["), "]"))
	}
	if found {
		return nil
	}
	return log.Error(
		"insecured registry",
		fmt.Errorf("insecure registry %q must be configured in Docker (found: %q)", api.InsecureRegistryAddress, strings.Join(ips, ",")),
	)
}
