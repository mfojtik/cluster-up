package api

var (
	// MinSupportedDockerVersion is the minimum Docker version we will support to run cluster up
	MinSupportedDockerVersion = "1.22"

	// InsecureRegistryAddress is in-secured registry CIDR that host Docker must be configured with
	InsecureRegistryAddress = "172.30.0.0/16"

	// ContainerNameOrigin is the name of the origin container. This is used to check if origin is
	// already running.
	ContainerNameOrigin = "origin"

	// DefaultImagePrefix sets the default prefix for images (like: 'registry.foo.bar/openshift/')
	DefaultImagePrefix = "openshift/"
)
