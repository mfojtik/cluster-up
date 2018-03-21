package preflight

import (
	"fmt"
	"os/exec"

	"github.com/mfojtik/cluster-up/pkg/log"
)

type Socat struct{}

func (s *Socat) Message() string {
	return "Checking if 'socat' binary is available"
}

func (s *Socat) Validate() error {
	socatPath, err := exec.LookPath("socat")
	if err != nil {
		return log.Error("socat path lookup", err)
	}
	out, err := exec.Command(socatPath, "-V").CombinedOutput()
	if err != nil {
		return fmt.Errorf("error executing 'socat' binary: %s (%v)", string(out), err)
	}
	return nil
}
