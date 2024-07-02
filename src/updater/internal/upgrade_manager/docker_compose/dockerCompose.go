package dockercompose

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// runs docker-compose -f <composeFilePath> up  -d --no-deps <serviceName>
func DockerComposeUpService(home, composeFilePath string, serviceName ...string) error {
	absComposeFilePath, err := filepath.Abs(composeFilePath)
	if err != nil {
		return fmt.Errorf("could not get absolute path of compose file: %v", err)
	}

	cmdArgs := []string{"-f", absComposeFilePath, "up", "-d", "--no-deps", "--remove-orphans"}
	cmdArgs = append(cmdArgs, serviceName...)

	log.Printf("Trying to run <%v>", strings.Join(cmdArgs, " "))
	cmd := exec.Command("docker-compose", cmdArgs...)

	if _, err := os.Stat(home); os.IsNotExist(err) {
		return fmt.Errorf("home directory does not exist: %v", err)
	}
	cmd.Dir = home

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("error running docker-compose command: %v", err)
	}

	return nil
}
