package docker

import (
	"bytes"
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

const (
	SIGKILL string = "SIGKILL" // 9 - interx
	SIGTERM string = "SIGTERM" // 15 - sekai
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

func KillContainerWithSigkill(ctx context.Context, cli *client.Client, containerID, signal string) error {

	err := cli.ContainerKill(ctx, containerID, signal)
	if err != nil {
		return err
	}
	return nil
}

func ExecInContainer(ctx context.Context, cli *client.Client, containerID string, command []string) ([]byte, error) {
	execConfig := container.ExecOptions{
		Cmd:          command,
		AttachStdout: true,
		AttachStderr: true,
		Detach:       false,
	}
	execCreateResponse, err := cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create container exec instance for container %s: %w", containerID, err)
	}

	execAttachConfig := container.ExecStartOptions{}
	resp, err := cli.ContainerExecAttach(ctx, execCreateResponse.ID, execAttachConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to attach to container exec instance %s: %w", execCreateResponse.ID, err)
	}
	defer resp.Close()

	var outBuf, errBuf bytes.Buffer
	_, err = stdcopy.StdCopy(&outBuf, &errBuf, resp.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to copy output from container exec: %w", err)
	}

	if len(errBuf.Bytes()) > 0 {
		return nil, fmt.Errorf(errBuf.String())
	}

	output := outBuf.Bytes()
	return output, nil
}
