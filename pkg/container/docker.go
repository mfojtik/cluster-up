package container

import (
	"context"
	"time"

	dockerapi "github.com/docker/docker/api"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/versions"
	"github.com/docker/docker/client"

	"github.com/mfojtik/cluster-up/pkg/log"
)

var defaultTimeout = 10 * time.Second

// Client interface has methods we need to call in Docker.
// The context is set in each function.
type Client interface {
	Info() (types.Info, error)
	ServerVersion() (types.Version, error)
	ContainerInspect(containerID string) (types.ContainerJSON, error)
	ContainerRemove(containerID string, options types.ContainerRemoveOptions) error
	ContainerCreate(config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, name string) (container.ContainerCreateCreatedBody, error)
	ContainerStart(containerID string, options types.ContainerStartOptions) error
	ContainerWait(containerID string) (int64, error)
	ContainerAttach(container string, options types.ContainerAttachOptions) (types.HijackedResponse, error)
	ContainerKill(containerID, signal string) error
}

func NewDockerClient() (Client, error) {
	dockerClient, err := client.NewEnvClient()
	if err != nil {
		return nil, log.Error("getting docker client", err)
	}
	internalClient := &internalDocker{client: dockerClient}
	internalClient.negotiateAPIVersion()
	return internalClient, nil
}

// This is nuts, the docker/docker cannot be used as client because they vendor
// their own context package...
type internalDocker struct {
	client *client.Client
}

// negotiateAPIVersion is copied from the latest docker code and it allows to use the
// newer docker client against older docker daemon.
// TODO: When the docker client is updated, this can be replaced by client.NegotiateAPIVersion()
func (d *internalDocker) negotiateAPIVersion() {
	ctx, cancelFn := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancelFn()
	p, _ := d.client.Ping(ctx)
	if p.APIVersion == "" {
		p.APIVersion = "1.24"
	}
	clientVersion := d.client.ClientVersion()
	if len(clientVersion) == 0 {
		clientVersion = dockerapi.DefaultVersion
	}
	// if server version is lower than the client version, downgrade
	if versions.LessThan(p.APIVersion, clientVersion) {
		d.client.UpdateClientVersion(p.APIVersion)
	}
}

func (d *internalDocker) ContainerKill(containerID, signal string) error {
	ctx, cancelFn := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancelFn()
	return d.client.ContainerKill(ctx, containerID, signal)
}

func (d *internalDocker) ContainerAttach(container string, options types.ContainerAttachOptions) (types.HijackedResponse, error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancelFn()
	return d.client.ContainerAttach(ctx, container, options)
}

func (d *internalDocker) ContainerStart(containerID string, options types.ContainerStartOptions) error {
	ctx, cancelFn := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancelFn()
	return d.client.ContainerStart(ctx, containerID, options)
}

func (d *internalDocker) ContainerWait(containerID string) (int64, error) {
	return d.client.ContainerWait(context.Background(), containerID)
}

func (d *internalDocker) ContainerCreate(config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, name string) (container.ContainerCreateCreatedBody, error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancelFn()
	return d.client.ContainerCreate(ctx, config, hostConfig, networkingConfig, name)
}

func (d *internalDocker) ContainerRemove(containerID string, options types.ContainerRemoveOptions) error {
	ctx, cancelFn := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancelFn()
	return d.client.ContainerRemove(ctx, containerID, options)
}

func (d *internalDocker) ContainerInspect(containerID string) (types.ContainerJSON, error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancelFn()
	return d.client.ContainerInspect(ctx, containerID)
}

func (d *internalDocker) ServerVersion() (types.Version, error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancelFn()
	return d.client.ServerVersion(ctx)
}

func (d *internalDocker) Info() (types.Info, error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancelFn()
	return d.client.Info(ctx)
}
