package container

import (
	"regexp"
)

var (
	fedoraPackage = regexp.MustCompile("\\.fc[0-9_]*\\.")
	rhelPackage   = regexp.MustCompile("\\.el[0-9_]*\\.")
)

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
