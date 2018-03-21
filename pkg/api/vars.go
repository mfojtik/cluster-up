package api

import "fmt"

var (
	// MinSupportedDockerVersion is the minimum Docker version we will support to run cluster up
	MinSupportedDockerVersion = "1.22"

	// InsecureRegistryAddress is in-secured registry CIDR that host Docker must be configured with
	InsecureRegistryAddress = "172.30.0.0/16"

	// ContainerNameOrigin is the name of the origin container. This is used to check if origin is
	// already running.
	ContainerNameOrigin = "origin"

	// DefaultImagePrefix sets the default prefix for images (like: 'registry.foo.bar/openshift/')
	DefaultImagePrefix = "openshift"

	// OriginImageName is the name of the openshift/origin image
	OriginImageName = "origin"

	// ImageTag is the image tag to use to run cluster.
	// This is mutated by CLI --tag argument, the default is what the 'oc' executable version is.
	ImageTag = "latest"

	// FIXME: This should come from the registry install component
	RegistryServiceClusterIP = "172.30.1.1"
)

// TODO: tbd
func DetermineImageTag() string {
	return "latest"
}

// OriginImage returns the openshift/origin image pull spec
func OriginImage() string {
	return fmt.Sprintf("%s/%s:%s", DefaultImagePrefix, OriginImageName, ImageTag)
}
