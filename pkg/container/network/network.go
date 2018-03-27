package network

import (
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/mfojtik/cluster-up/pkg/api"
	"github.com/mfojtik/cluster-up/pkg/container"
	"github.com/mfojtik/cluster-up/pkg/log"
	"github.com/mfojtik/cluster-up/pkg/util/sets"
)

type NetworkConfig struct {
	dockerClient container.Client

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

func BuildNetworkConfig(dockerClient container.Client, publicHostname string, portForward bool, proxy *ProxyConfig) (*NetworkConfig, error) {
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

func (c *NetworkConfig) String() string {
	return fmt.Sprintf("server: %s, additional: %s", c.ServerIP(),
		strings.Join(c.AdditionalIPs(), ","))
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

// Determine if we can use the 127.0.0.1 as server address
func (c *NetworkConfig) runDummySocatServer(containerName string, testDialFn func(string) error) {
	container.Docker(c.dockerClient, "").
		Discard().
		HostNetwork().
		Privileged().
		OnStart(testDialFn).
		Entrypoint("socat").
		Name(containerName).
		Command("TCP-LISTEN:8443,crlf,reuseaddr,fork", "SYSTEM:\"echo 'hello world'\"").
		Run(api.OriginImage())
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
			testDoneChan := make(chan error, 1)
			serverStopChan := make(chan struct{}, 1)
			testContainerName := "test-localhost-bind"
			go func() {
				defer close(serverStopChan)
				c.runDummySocatServer(testContainerName,
					func(string) error {
						testHost := "127.0.0.1:8443"
						err := WaitForSuccessfulDial(false, "tcp", testHost, 200*time.Millisecond, 1*time.Second, 10)
						if err != nil {
							testDoneChan <- err
							return nil
						}
						defer close(testDoneChan)
						return nil
					})
			}()
			defer func() {
				if err := c.dockerClient.ContainerKill(testContainerName, "TERM"); err != nil {
					log.Error("killing test container", err)
				}
				log.Debugf("Waiting for the test server to finish ...")
				<-serverStopChan
			}()
			select {
			case err := <-testDoneChan:
				if err != nil {
					return log.Error("dial localhost test", err)
				}
				log.Debugf("Using 127.0.0.1 IP as the host IP")
				c.serverIP = "127.0.0.1"
			case <-time.After(10 * time.Second):
				return fmt.Errorf("failed to determine the host IP address")
			}
		}
	}

	cmd := container.Docker(c.dockerClient, "").
		Discard().
		HostNetwork().
		Privileged().
		Entrypoint("hostname").
		Name("test-additional-ips").
		Command("-I").Run(api.OriginImage())
	if cmd.Error() != nil {
		return log.Error("test-additional-ip", cmd.Error())
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

func WaitForSuccessfulDial(https bool, network, address string, timeout, interval time.Duration, retries int) error {
	var (
		conn net.Conn
		err  error
	)
	for i := 0; i <= retries; i++ {
		dialer := net.Dialer{Timeout: timeout}
		if https {
			conn, err = tls.DialWithDialer(&dialer, network, address, &tls.Config{InsecureSkipVerify: true})
		} else {
			conn, err = dialer.Dial(network, address)
		}
		if err != nil {
			glog.V(5).Infof("Got error %#v, trying again: %#v\n", err, address)
			time.Sleep(interval)
			continue
		}
		conn.Close()
		return nil
	}
	return err
}
