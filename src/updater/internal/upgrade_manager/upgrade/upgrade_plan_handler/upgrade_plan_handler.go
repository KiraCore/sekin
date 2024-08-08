package upgradeplanhandler

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/docker/docker/client"
	"github.com/google/shlex"
	"github.com/kiracore/sekin/src/updater/internal/types"
	"github.com/kiracore/sekin/src/updater/internal/upgrade_manager/docker"
	dockercompose "github.com/kiracore/sekin/src/updater/internal/upgrade_manager/docker_compose"
	"github.com/kiracore/sekin/src/updater/internal/utils"
	"gopkg.in/yaml.v2"
)

func CheckIfPlanIsHardFork(plan *types.PlanData) bool {
	if !plan.Plan.InstateUpgrade && !plan.Plan.SkipHandler && plan.Plan.RebootRequired {
		if plan.Plan.Resources[0].ID == "sekai" {
			return true
		}
	}
	return false
}

func ExecuteSekaiHardForkUpgrade(plan *types.PlanData, cli *client.Client) error {
	ctx := context.Background()

	composeFilePath := filepath.Join(types.SEKIN_HOME, "compose.yml")
	backupComposeFilePath := filepath.Join(types.SEKIN_HOME, "compose.yml.bak")

	// exist := utils.FileExists(backupComposeFilePath)
	// if !exist {
	err := utils.CopyFile(composeFilePath, backupComposeFilePath)
	if err != nil {
		return err
	}

	err = docker.KillContainerWithSigkill(context.Background(), cli, types.SEKAI_CONTAINER_ID, docker.SIGTERM)
	if err != nil {
		return err
	}

	exportedGenesisPath := path.Join(types.CONTAINERIZED_SEKAI_CONFIG, "exported.json")
	cmd, err := shlex.Split(fmt.Sprintf("sekaid export --home=%v --output-document=%v", types.CONTAINERIZED_SEKAI_HOME, exportedGenesisPath))
	if err != nil {
		return err
	}
	_, err = docker.ExecInContainer(ctx, cli, types.SEKAI_CONTAINER_ID, cmd)
	if err != nil {
		return err
	}

	composeData, err := os.ReadFile(composeFilePath)
	if err != nil {
		return nil
	}

	var currentCompose map[string]interface{}
	err = yaml.Unmarshal(composeData, &currentCompose)
	dockercompose.GetDockerComposeProject(composeData)

	if err != nil {
		fmt.Println("Error unmarshalling YAML:", err)
		return err
	}

	dockercompose.UpdateComposeYMLField(currentCompose, "sekai", "image", fmt.Sprintf("%v:%v", "ghcr.io/kiracore/sekin/sekai", plan.Plan.Resources[0].Version))

	updatedData, err := yaml.Marshal(&currentCompose)
	if err != nil {
		return err
	}
	var originalPerm os.FileMode = 0644 // Default permission if the file doesn't exist
	if fileInfo, err := os.Stat(composeFilePath); err == nil {
		originalPerm = fileInfo.Mode()
	}

	err = os.WriteFile(composeFilePath, updatedData, originalPerm)
	if err != nil {
		return err
	}

	diff, err := dockercompose.CompareYAMLFiles(backupComposeFilePath, composeFilePath)
	if err != nil {
		return err
	}
	log.Println("DIFF", diff)

	err = dockercompose.DockerComposeUpService(types.SEKIN_HOME, composeFilePath)
	if err != nil {
		log.Printf("ERROR when trying to run <%v> compose file: %v ", composeFilePath, err)
		return err
	}

	var check bool = false

	attempts := 3
	for i := range attempts {

		status, err := docker.CheckContainerState(cli, types.SEKAI_CONTAINER_ID)
		if err != nil {
			fmt.Println("Error checking container status:", err)
			return err
		}
		if status == "running" {
			log.Println("Container updated successfully")
			check = true
			break
		} else {
			//if not "running"
			log.Printf("WARNING: shidai status is %v, trying again. Attempt %v out of %v", status, i+1, attempts)
			check = false
			time.Sleep(time.Second)
		}
	}

	if check {
		err = utils.DeleteFile(backupComposeFilePath)
		if err != nil {
			return fmt.Errorf("error deleting backup file: %w", err)
		}
	} else {
		log.Println("ERROR: sekai container cannot start")
	}

	genesisPath := path.Join(types.CONTAINERIZED_SEKAI_CONFIG, "genesis.json")

	cmd, err = shlex.Split(fmt.Sprintf("sekaid new-genesis-from-exported %v %v --%v --json-minimize=false", exportedGenesisPath, genesisPath, types.CONTAINERIZED_SEKAI_HOME))
	if err != nil {
		return err
	}
	_, err = docker.ExecInContainer(ctx, cli, types.SEKAI_CONTAINER_ID, cmd)
	if err != nil {
		return err
	}

	err = utils.RemoveFolderContent(path.Join(types.SEKAI_HOME, "data"))
	if err != nil {
		return err
	}
	err = setEmptyValidatorState(types.SEKAI_HOME)
	if err != nil {
		return err
	}

	cmd, err = shlex.Split(fmt.Sprintf("sekaid start --home=%v", types.CONTAINERIZED_SEKAI_HOME))
	if err != nil {
		return err
	}
	_, err = docker.ExecInContainer(ctx, cli, types.SEKAI_CONTAINER_ID, cmd)
	if err != nil {
		return err
	}
	return nil
}

func setEmptyValidatorState(sekaidHome string) error {
	emptyState := `
	{
		"height": "0",
		"round": 0,
		"step": 0
	}`
	sekaidDataFolder := sekaidHome + "/data"
	err := os.Mkdir(sekaidDataFolder, 0755)
	if err != nil {
		if !os.IsExist(err) {
			return fmt.Errorf("unable to create <%s> folder, err: %w", sekaidDataFolder, err)
		}
	}
	// utils.CreateFileWithData(sekaidDataFolder+"/priv_validator_state.json", []byte(emptyState))
	file, err := os.Create(sekaidDataFolder + "/priv_validator_state.json")
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = file.Write([]byte(emptyState))
	if err != nil {
		return fmt.Errorf("failed to write data to file: %w", err)
	}
	fmt.Println(emptyState, sekaidDataFolder)
	return nil
}
