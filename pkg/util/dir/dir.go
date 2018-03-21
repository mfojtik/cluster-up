package dir

import (
	"os"
	"path/filepath"
)

var openshiftLocalDirectoryPrefix = "openshift.local"

func InOpenShiftLocal(name string) string {
	return openshiftLocalDirectoryPrefix + "." + name
}
func MakeAbs(path, base string) (string, error) {
	if filepath.IsAbs(path) {
		return path, nil
	}
	if len(base) == 0 {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		base = cwd
	}
	return filepath.Join(base, path), nil
}
