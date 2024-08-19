package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
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
	log.Println("Output docker exec:", string(output))
	return output, nil
}

func ExecInContainerV2(ctx context.Context, cli *client.Client, containerID string, command []string) ([]byte, error) {

	fmt.Println("executing in container:", strings.Join(command, " "))

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

	// Inspect the exec instance to get the exit code
	execInspectResp, err := cli.ContainerExecInspect(ctx, execCreateResponse.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect exec instance %s: %w", execCreateResponse.ID, err)
	}

	if execInspectResp.ExitCode != 0 {
		return nil, fmt.Errorf("command failed with exit code %d: %s", execInspectResp.ExitCode, errBuf.String())
	}

	output := outBuf.Bytes()
	log.Println("Output docker exec:", string(output))
	return output, nil
}

// DeleteFileOrFolder deletes a file or folder from a container
func DeleteFileOrFolder(cli *client.Client, containerID, targetPath string) error {
	log.Printf("Trying to delete %v", targetPath)

	// Check if targetPath is a directory
	isDir := targetPath[len(targetPath)-1] == '/'

	// Create an empty tar stream to overwrite the target path
	tarReader := createEmptyTarForTarget(targetPath, isDir)

	// Apply the empty tar archive to the container filesystem
	err := cli.CopyToContainer(context.Background(), containerID, "/", tarReader, types.CopyToContainerOptions{
		AllowOverwriteDirWithFile: true,
	})
	if err != nil {
		return err
	}

	return nil
}

// createEmptyTarForTarget creates a tar archive containing an empty directory or file
func createEmptyTarForTarget(targetPath string, isDir bool) *bytes.Reader {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	// Create the tar header for the empty directory or file
	var header *tar.Header
	if isDir {
		header = &tar.Header{
			Name:     targetPath,
			Mode:     0755,
			Typeflag: tar.TypeDir,
		}
	} else {
		header = &tar.Header{
			Name: targetPath,
			Mode: 0644,
			Size: 0, // Empty file
		}
	}

	// Write the header to the tar archive
	if err := tw.WriteHeader(header); err != nil {
		log.Fatalf("Failed to write header to tar: %v", err)
	}

	// Close the tar writer
	if err := tw.Close(); err != nil {
		log.Fatalf("Failed to close tar writer: %v", err)
	}

	return bytes.NewReader(buf.Bytes())
}

func CreateFileInContainer(cli *client.Client, containerID, filePath string, data []byte) error {
	// Create a tar archive that contains the file with the data and directory structure
	log.Printf("Creating %v in %v, data:\n%v...", filePath, containerID, string(data[0:50]))
	tarReader := createTarWithDirsAndFile(filePath, data)

	// Copy the tar archive to the container's filesystem
	err := cli.CopyToContainer(context.Background(), containerID, "/", tarReader, container.CopyToContainerOptions{
		AllowOverwriteDirWithFile: true,
	})
	if err != nil {
		return err
	}

	return nil
}

// createTarWithDirsAndFile creates a tar archive containing the necessary directory structure
// and the file with the provided data.
func createTarWithDirsAndFile(filePath string, data []byte) *bytes.Reader {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	// Ensure all necessary directories are included in the tar
	dir := filepath.Dir(filePath)
	if dir != "/" {
		dirs := strings.Split(dir, "/")
		for i := range dirs {
			currentDir := filepath.Join(dirs[:i+1]...)
			if currentDir == "" || currentDir == "." {
				continue
			}

			header := &tar.Header{
				Name:     currentDir + "/",
				Mode:     0755,
				Typeflag: tar.TypeDir,
			}

			if err := tw.WriteHeader(header); err != nil {
				log.Fatalf("Failed to write directory header to tar: %v", err)
			}
		}
	}

	// Create a tar header for the file
	header := &tar.Header{
		Name: filePath,
		Mode: 0644,
		Size: int64(len(data)),
	}

	// Write the header
	if err := tw.WriteHeader(header); err != nil {
		log.Fatalf("Failed to write file header to tar: %v", err)
	}

	// Write the file data
	if _, err := tw.Write(data); err != nil {
		log.Fatalf("Failed to write file data to tar: %v", err)
	}

	// Close the tar writer
	if err := tw.Close(); err != nil {
		log.Fatalf("Failed to close tar writer: %v", err)
	}

	return bytes.NewReader(buf.Bytes())
}

func DeleteFileOrFolderWithBusybox(cli *client.Client, containerID, targetPath, mountedFolder string) error {
	log.Printf("Trying to delete %v in %v container, in mounted volume %v ", targetPath, containerID, mountedFolder)
	// Ensure that the busybox image is available
	if err := ensureImageExists(cli, "busybox"); err != nil {
		return fmt.Errorf("failed to ensure busybox image exists: %w", err)
	}
	const targetFolder = "/mnt"
	// Create a new busybox container
	resp, err := cli.ContainerCreate(context.Background(), &container.Config{
		Image: "busybox",
		Cmd:   []string{"sh", "-c", fmt.Sprintf("rm -rf %s/%s", targetFolder, targetPath)},
	}, &container.HostConfig{
		Mounts: []mount.Mount{
			{
				// Type:   mount.TypeVolume,
				Type:   mount.TypeVolume,
				Source: mountedFolder,
				Target: targetFolder,
			},
		},
	}, nil, nil, "")
	if err != nil {
		return err
	}
	// Start the busybox container
	if err := cli.ContainerStart(context.Background(), resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start busybox container: %w", err)
	}

	// Wait for the container to finish execution
	statusCh, errCh := cli.ContainerWait(context.Background(), resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("error while waiting for container: %w", err)
		}
	case <-statusCh:
	}

	// Capture the output (optional)
	out, err := cli.ContainerLogs(context.Background(), resp.ID, container.LogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return fmt.Errorf("failed to capture logs: %w", err)
	}
	stdcopy.StdCopy(log.Writer(), log.Writer(), out)

	// Clean up the busybox container
	if err := cli.ContainerRemove(context.Background(), resp.ID, container.RemoveOptions{Force: true}); err != nil {
		return fmt.Errorf("failed to remove busybox container: %w", err)
	}

	return nil
}

func ensureImageExists(cli *client.Client, imageName string) error {
	_, _, err := cli.ImageInspectWithRaw(context.Background(), imageName)
	if err != nil {
		log.Printf("Image %s not found locally, pulling...", imageName)

		out, err := cli.ImagePull(context.Background(), imageName, image.PullOptions{})
		if err != nil {
			return fmt.Errorf("failed to pull image %s: %w", imageName, err)
		}
		defer out.Close()

		// Print the output of the pull
		_, err = io.Copy(log.Writer(), out)
		if err != nil {
			return fmt.Errorf("failed to read pull output: %w", err)
		}
	}

	return nil
}
