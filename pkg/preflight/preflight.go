package preflight

import (
	"github.com/mfojtik/cluster-up/pkg/container"
)

// Validator performs pre-flight validation checks
type Validator interface {
	Message() string
	Validate() error
}

func NewValidator(client container.Client) Validator {
	ctx := validatorContext{
		containerClient: client,
	}
	chain := &containerValidator{}
	// Define Docker validation checks
	chain.Add(&DockerVersion{ctx})
	chain.Add(&DockerRegistry{ctx})

	// OpenShift pre-flight checks
	chain.Add(&OpenShiftRunning{ctx})
	return chain
}
