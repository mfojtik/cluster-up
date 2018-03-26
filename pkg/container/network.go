package container

import (
	"net"
	"strings"
	"time"

	"github.com/mfojtik/cluster-up/pkg/api"
	"github.com/mfojtik/cluster-up/pkg/log"
	"github.com/mfojtik/cluster-up/pkg/util/sets"
)

type NetworkConfig struct {
	dockerClient Client

	portForwarding bool
	publicHostname string
	serverIP       string
	additionalIPs  []string
	proxyConfig    *ProxyConfig
}

type ProxyConfig struct {
	HTTPSProxy string
	HTTPProxy  string
	NoProxy    []string
}

func BuildNetworkConfig(dockerClient Client, publicHostname string, portForward bool, proxy *ProxyConfig) (*NetworkConfig, error) {
	c := &NetworkConfig{
		dockerClient:   dockerClient,
		publicHostname: publicHostname,
		portForwarding: portForward,
		proxyConfig:    proxy,
	}
	if err := c.build(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *NetworkConfig) ServerIP() string {
	return c.serverIP
}

func (c *NetworkConfig) AdditionalIPs() []string {
	return c.additionalIPs
}

func (c *NetworkConfig) ProxyConfig() *ProxyConfig {
	values := []string{"127.0.0.1", c.ServerIP(), "localhost", api.RegistryServiceClusterIP}
	// FIXME: This should move away, external componets should not be able to modify the no_proxy settings
	values = append(values, api.RegistryServiceClusterIP)
	noProxySet := sets.NewString(c.proxyConfig.NoProxy...)
	newProxyConfig := *c.proxyConfig
	for _, v := range values {
		if !noProxySet.Has(v) {
			noProxySet.Insert(v)
			newProxyConfig.NoProxy = append(newProxyConfig.NoProxy, v)
		}
	}
	return &newProxyConfig
}

func (c *NetworkConfig) build() error {
	if c.portForwarding {
		log.Debugf("Using 127.0.0.1 IP as the host IP, ports will be forwarded")
		c.serverIP = "127.0.0.1"
	} else {
		if ip := net.ParseIP(c.publicHostname); ip != nil && !ip.IsUnspecified() {
			log.Debugf("Using public hostname %s IP %s as the hostIP", c.publicHostname, ip)
			c.serverIP = ip.String()
		} else {
			testDoneChan := make(chan error)
			testDial := func() error {
				testHost := "127.0.0.1:8443"
				err := WaitForSuccessfulDial(false, "tcp", testHost, 200*time.Millisecond, 1*time.Second, 10)
				if err != nil {
					testDoneChan <- err
					return nil
				}
				close(testDoneChan)
				return nil
			}
			testContainerName := "test-localhost-bind"
			go func() {
				// Determine if we can use the 127.0.0.1 as server address
				cmd := NewRunner(c.dockerClient, "").
					RemoveWhenExit().
					HostNetwork().
					Privileged().
					AfterStartHook(testDial).
					Entrypoint("socat").
					Command("TCP-LISTEN:8443,crlf,reuseaddr,fork", "SYSTEM:\"echo 'hello world'\"").
					RunImageWithName(api.OriginImage(), testContainerName)
				if cmd.Error() != nil {
					log.Error("test localhost bind", cmd.Error())
				}
			}()
			defer func() {
				err := c.dockerClient.ContainerKill(testContainerName, "TERM")
				if err != nil {
					log.Error("killing test container", err)
				}
			}()
			select {
			case err := <-testDoneChan:
				if err != nil {
					return log.Error("dial localhost test", err)
				}
				log.Debugf("Using 127.0.0.1 IP as the host IP")
				c.serverIP = "127.0.0.1"
				break
			case <-time.After(10 * time.Second):
			}
		}
	}
	cmd := NewRunner(c.dockerClient, "").
		RemoveWhenExit().
		HostNetwork().
		Privileged().
		Entrypoint("hostname").
		Command("-I").RunImageWithName(api.OriginImage(), "test-additional-ip")
	if cmd.Error() != nil {
		return cmd.Error()
	}
	candidates := strings.Split(string(cmd.Output()), " ")
	for _, ip := range candidates {
		if len(strings.TrimSpace(ip)) == 0 {
			continue
		}
		if ip != c.serverIP && !strings.Contains(ip, ":") {
			c.additionalIPs = append(c.additionalIPs, ip)
		}
	}
	log.Debugf("Using %q as additional IPs", strings.Join(c.additionalIPs, ","))
	return nil
}
