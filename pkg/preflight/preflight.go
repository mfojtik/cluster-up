package preflight

import (
	"github.com/mfojtik/cluster-up/pkg/container"
)

// Validator performs pre-flight validation checks
type Validator interface {
	Message() string
	Validate() error
}

func NewValidator(client container.Client, portForward, skipRegistryCheck bool) Validator {
	ctx := validatorContext{
		containerClient: client,
	}
	chain := &validator{}
	// Define Docker validation checks
	chain.Add(&DockerVersion{ctx})

	if !skipRegistryCheck {
		chain.Add(&DockerRegistry{ctx})
	}

	// OpenShift pre-flight checks
	chain.Add(&OpenShiftRunning{ctx})

	if portForward {
		chain.Add(&Socat{})
	}
	return chain
}
