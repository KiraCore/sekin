package upgradeplanhandler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/docker/docker/client"
	"github.com/kiracore/sekin/src/updater/internal/types"
	"github.com/kiracore/sekin/src/updater/internal/upgrade_manager/docker"
	dockercompose "github.com/kiracore/sekin/src/updater/internal/upgrade_manager/docker_compose"
	"github.com/kiracore/sekin/src/updater/internal/utils"

	"gopkg.in/yaml.v2"
)

func CheckIfPlanIsInterxUpgrade(plan *types.PlanData) bool {
	if plan.Plan.InstateUpgrade && plan.Plan.SkipHandler {
		if plan.Plan.Resources[0].ID == "interx" {
			return true
		}
	}
	return false
}

func ExecuteInterxSoftForkUpgrade(plan *types.PlanData, cli *client.Client) error {
	log.Println("executing interx soft fork upgrade")

	composeFilePath := filepath.Join(types.SEKIN_HOME, "compose.yml")
	backupComposeFilePath := filepath.Join(types.SEKIN_HOME, "compose.yml.bak")

	err := utils.CopyFile(composeFilePath, backupComposeFilePath)
	if err != nil {
		log.Printf("ERROR: %v", err.Error())
		return err
	}

	container, err := cli.ContainerInspect(context.Background(), types.INTERX_CONTAINER_ID)
	if err != nil {
		return err
	}

	if !container.State.Running {
		log.Printf("Killing interx container with %v code", docker.SIGKILL)
		err = docker.KillContainerWithSigkill(context.Background(), cli, types.INTERX_CONTAINER_ID, docker.SIGKILL)
		if err != nil {
			log.Printf("ERROR: %v", err.Error())
			return err
		}
	}

	log.Println("reading", composeFilePath)
	var currentCompose map[string]interface{}
	composeData, err := os.ReadFile(composeFilePath)
	if err != nil {
		return nil
	}
	err = yaml.Unmarshal(composeData, &currentCompose)
	if err != nil {
		fmt.Println("Error unmarshalling YAML:", err)
		return err
	}

	log.Println("updating INTERX image field")

	dockercompose.UpdateComposeYMLField(currentCompose, "interx", "image", fmt.Sprintf("%v:%v", "ghcr.io/kiracore/sekin/interx", plan.Plan.Resources[0].Version))

	updatedData, err := yaml.Marshal(&currentCompose)
	if err != nil {
		log.Printf("ERROR: %v", err.Error())

		return err
	}
	var originalPerm os.FileMode = 0644 // Default permission if the file doesn't exist
	if fileInfo, err := os.Stat(composeFilePath); err == nil {
		originalPerm = fileInfo.Mode()
	}

	log.Printf("writing %v with %+v", composeFilePath, string(updatedData))
	err = os.WriteFile(composeFilePath, updatedData, originalPerm)
	if err != nil {
		log.Printf("ERROR: %v", err.Error())
		return err
	}

	diff, err := dockercompose.CompareYAMLFiles(backupComposeFilePath, composeFilePath)
	if err != nil {
		log.Printf("ERROR: %v", err.Error())
		return err
	}
	log.Println("DIFF", diff)

	err = dockercompose.DockerComposeUpService(types.SEKIN_HOME, composeFilePath)
	if err != nil {
		log.Printf("ERROR when trying to run <%v> compose file: %v ", composeFilePath, err)
		return err
	}

	check := false
	attempts := 3
	for i := range attempts {
		status, err := docker.CheckContainerState(cli, types.INTERX_CONTAINER_ID)
		if err != nil {
			fmt.Println("Error checking container status:", err)
			return err
		}
		log.Println("checking container state, status:", status)
		if status == "running" {
			log.Println("interx container updated successfully")
			check = true
			break
		} else {
			//if not "running"
			log.Printf("WARNING: shidai status is %v, trying again. Attempt %v out of %v", status, i+1, attempts)
			time.Sleep(time.Second)
		}
	}

	if check {
		err = utils.DeleteFile(backupComposeFilePath)
		if err != nil {
			log.Printf("ERROR: %v", err.Error())
			return fmt.Errorf("error deleting backup file: %w", err)
		}
	} else {
		log.Println("ERROR: interx container cannot start")
		return fmt.Errorf("ERROR: interx container cannot start")
	}

	log.Println("deleting upgrade plan")
	err = utils.DeleteFile(types.UPDATE_PLAN)
	if err != nil {
		return err
	}

	err = startInterx()
	if err != nil {
		return err
	}
	return nil
}

func startInterx() error {
	log.Println("starting interx")
	cmd := CommandRequest{
		Command: "start",
		Args: map[string]interface{}{
			"home": types.CONTAINERIZED_INTERX_HOME,
		},
	}
	_, err := executeCallerCommand("localhost", "8081", "POST", cmd)
	if err != nil {
		if errors.Is(err, io.EOF) {
		} else {
			return fmt.Errorf("unable execute <%v> request, error: %w", cmd, err)
		}
	}

	return nil
}
