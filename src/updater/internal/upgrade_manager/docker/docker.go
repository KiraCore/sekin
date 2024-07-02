package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/client"
)

func CheckContainerState(cli *client.Client, containerID string) (string, error) {
	ctx := context.Background()

	// Get the container's JSON representation
	containerJSON, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return "", fmt.Errorf("error inspecting container: %v", err)
	}

	// Check the container state
	state := containerJSON.State
	if state == nil {
		return "", fmt.Errorf("container state is nil")
	}

	// Return the container state status
	return state.Status, nil
}
