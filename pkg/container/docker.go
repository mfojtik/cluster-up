package container

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/mfojtik/cluster-up/pkg/context"

	"github.com/mfojtik/cluster-up/pkg/log"
)

// Client interface has methods we need to call in Docker.
// The context is set in each function.
type Client interface {
	Info() (types.Info, error)
	ServerVersion() (types.Version, error)
	ContainerInspect(containerID string) (types.ContainerJSON, error)
	ContainerRemove(containerID string, options types.ContainerRemoveOptions) error
}

// This is nuts, the docker/docker cannot be used as client because they vendor
// their own context package...
type internalDocker struct {
	client *client.Client
}

func NewDockerClient() (Client, error) {
	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, log.Error("getting docker client", err)
	}
	context.Run(func(c context.Context) error { dockerClient.NegotiateAPIVersion(c); return nil })
	return &internalDocker{client: dockerClient}, nil
}

func (d *internalDocker) ContainerRemove(containerID string, options types.ContainerRemoveOptions) error {
	return context.Run(func(c context.Context) error {
		return d.client.ContainerRemove(c, containerID, options)
	})
}

func (d *internalDocker) ContainerInspect(containerID string) (types.ContainerJSON, error) {
	var result types.ContainerJSON
	err := context.Run(func(c context.Context) error {
		var resultErr error
		result, resultErr = d.client.ContainerInspect(c, containerID)
		return resultErr
	})
	return result, err
}

func (d *internalDocker) ServerVersion() (types.Version, error) {
	var version types.Version
	err := context.Run(func(c context.Context) error {
		var versionErr error
		version, versionErr = d.client.ServerVersion(c)
		return versionErr
	})
	return version, err
}

func (d *internalDocker) Info() (types.Info, error) {
	var info types.Info
	err := context.Run(func(c context.Context) error {
		var infoErr error
		info, infoErr = d.client.Info(c)
		return infoErr
	})
	return info, err
}
