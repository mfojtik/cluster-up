package api

// ClusterConfig contains variables we need to build the configuration and run the cluster.
type ClusterConfig struct {
	UseNSEnterMount bool

	ServerIP     string
	AdditionalIP []string
}
