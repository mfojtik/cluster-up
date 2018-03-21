package preflight

import (
	"fmt"
	"strings"

	"github.com/mfojtik/cluster-up/pkg/container"
	"github.com/mfojtik/cluster-up/pkg/log"
)

type validatorContext struct {
	containerClient container.Client
}

func (c *validatorContext) ContainerClient() container.Client {
	return c.containerClient
}

type validator struct {
	validators []Validator
}

func (c *validator) Add(v Validator) *validator {
	c.validators = append(c.validators, v)
	return c
}

func (c *validator) Message() string {
	return "Performing pre-flight checks"
}

func (c *validator) Validate() error {
	var (
		errCount    int
		errMessages []string
	)
	for _, v := range c.validators {
		log.Infof("--> %s", v.Message())
		if err := v.Validate(); err != nil {
			errCount++
			errMessages = append(errMessages, err.Error())
		}
	}
	if errCount == 0 {
		return nil
	}
	return fmt.Errorf("validation failed with %d errors:\n%s", errCount, strings.Join(errMessages, "\n"))
}
