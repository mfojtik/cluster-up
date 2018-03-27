package up

import (
	"fmt"
	"io"

	"github.com/mfojtik/cluster-up/pkg/api"
	"github.com/mfojtik/cluster-up/pkg/container"
	"github.com/mfojtik/cluster-up/pkg/container/network"
	"github.com/mfojtik/cluster-up/pkg/container/volumes"
	"github.com/mfojtik/cluster-up/pkg/log"
	"github.com/mfojtik/cluster-up/pkg/preflight"
	"github.com/mfojtik/cluster-up/pkg/util/template"
	"github.com/spf13/cobra"
)

const RecommendedClusterUpName = "up"

var upLong = template.LongDesc(`
	Starts an OpenShift cluster using Docker containers, provisioning a registry, router,
	initial templates, and a default project.

	This command will attempt to use an existing connection to a Docker daemon. Before running
	the command, ensure that you can execute docker commands successfully (i.e. 'docker ps').

	By default, the OpenShift cluster will be setup to use a routing suffix that ends in nip.io.
	This is to allow dynamic host names to be created for routes. An alternate routing suffix
	can be specified using the --routing-suffix flag.`)

var upExample = template.Examples(`
	  # Start OpenShift using a specific public host name
	  %[1]s --public-hostname=my.address.example.com

	  # Use a different set of images
	  %[1]s --image="registry.example.com/origin"`)

type ClusterUpOptions struct {
	Output    io.Writer
	ErrOutput io.Writer

	PublicHostname string
	RoutingSuffix  string
	PortForwarding bool

	SkipRegistryCheck bool

	BaseDir           string
	SpecifiedBaseDir  bool
	UseExistingConfig bool
	WriteConfig       bool

	ServerLogLevel int

	HTTPProxy, HTTPSProxy string
	NoProxy               []string

	dockerClient container.Client

	volumeConfig  *volumes.VolumesConfig
	networkConfig *network.NetworkConfig
	proxyConfig   *network.ProxyConfig
}

func NewClusterUpCommand(recommendedName, parentName string, out, errOut io.Writer) *cobra.Command {
	c := &ClusterUpOptions{}
	c.Output = out
	c.ErrOutput = errOut

	client, err := container.NewDockerClient()
	if err != nil {
		log.Fatal(err)
	}
	c.dockerClient = client

	cmd := &cobra.Command{
		Use:     recommendedName,
		Short:   "Brings up a minimal OpenShift cluster",
		Long:    fmt.Sprintf(upLong, parentName, recommendedName),
		Example: fmt.Sprintf(upExample, parentName+" "+recommendedName),
		Run: func(cmd *cobra.Command, args []string) {
			if err := c.Validate(); err != nil {
				log.Fatal(err)
			}
			if err := c.Complete(); err != nil {
				log.Fatal(err)
			}
			if err := c.Run(); err != nil {
				log.Fatal(err)
			}
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&api.ImageTag, "tag", api.DetermineImageTag(), "Specify the tag for OpenShift images")
	flags.StringVar(&api.DefaultImagePrefix, "image", api.DefaultImagePrefix, "Specify the images to use for OpenShift")
	flags.BoolVar(&c.SkipRegistryCheck, "skip-registry-check", false, "Skip Docker daemon registry check")
	flags.StringVar(&c.PublicHostname, "public-hostname", "", "Public hostname for OpenShift cluster")
	flags.StringVar(&c.RoutingSuffix, "routing-suffix", "", "Default suffix for server routes")
	flags.StringVar(&c.BaseDir, "base-dir", c.BaseDir, "Directory on Docker host for cluster up configuration")
	flags.BoolVar(&c.UseExistingConfig, "use-existing-config", false, "Use existing configuration if present")
	flags.BoolVar(&c.WriteConfig, "write-config", false, "Write the configuration files into host config dir")
	flags.BoolVar(&c.PortForwarding, "forward-ports", c.PortForwarding, "Use Docker port-forwarding to communicate with origin container. Requires 'socat' locally.")
	flags.IntVar(&c.ServerLogLevel, "server-loglevel", 3, "Log level for OpenShift server")

	// TODO: Figure out how to externalize these
	/*
		flags.BoolVar(&c.ShouldInstallMetrics, "metrics", false, "Install metrics (experimental)")
		flags.BoolVar(&c.ShouldInstallLogging, "logging", false, "Install logging (experimental)")
		flags.BoolVar(&c.ShouldInstallServiceCatalog, "service-catalog", false, "Install service catalog (experimental).")
	*/

	// TODO: Make these post-install steps
	/*
		flags.StringVar(&c.ImageStreams, "image-streams", defaultImageStreams, "Specify which image streams to use, centos7|rhel7")
	*/

	// Proxy flags
	flags.StringVar(&c.HTTPProxy, "http-proxy", "", "HTTP proxy to use for master and builds")
	flags.StringVar(&c.HTTPSProxy, "https-proxy", "", "HTTPS proxy to use for master and builds")
	flags.StringArrayVar(&c.NoProxy, "no-proxy", c.NoProxy, "List of hosts or subnets for which a proxy should not be used")

	// We don't want to expose this to normal users
	flags.MarkHidden("tag")

	return cmd
}

func (c *ClusterUpOptions) Validate() error {
	if err := preflight.NewValidator(c.dockerClient, c.PortForwarding, c.SkipRegistryCheck).Validate(); err != nil {
		return err
	}
	return nil
}

func (c *ClusterUpOptions) Complete() error {
	c.SpecifiedBaseDir = len(c.BaseDir) != 0
	var err error

	c.volumeConfig, err = volumes.BuildHostVolumesConfig(c.dockerClient, c.BaseDir)
	if err != nil {
		return err
	}

	if len(c.HTTPSProxy) > 0 || len(c.HTTPProxy) > 0 {
		c.proxyConfig = &network.ProxyConfig{
			HTTPProxy:  c.HTTPProxy,
			HTTPSProxy: c.HTTPSProxy,
			NoProxy:    c.NoProxy,
		}
	}
	c.networkConfig, err = network.BuildNetworkConfig(c.dockerClient, c.PublicHostname, c.PortForwarding, c.proxyConfig)
	if err != nil {
		return err
	}
	log.Infof("--> Networking configuration: %s", c.networkConfig)

	// TODO: Pull images
	return nil
}

func (c *ClusterUpOptions) Run() error {
	return nil
}
