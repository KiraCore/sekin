package upgradeplanhandler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
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
	log.Println("executing hard fork upgrade")
	ctx := context.Background()

	composeFilePath := filepath.Join(types.SEKIN_HOME, "compose.yml")
	backupComposeFilePath := filepath.Join(types.SEKIN_HOME, "compose.yml.bak")

	err := utils.CopyFile(composeFilePath, backupComposeFilePath)
	if err != nil {
		log.Printf("ERROR: %v", err.Error())

		return err
	}

	log.Println("Killing sekaid container with 15 code")
	err = docker.KillContainerWithSigkill(context.Background(), cli, types.SEKAI_CONTAINER_ID, docker.SIGTERM)
	if err != nil {
		log.Printf("ERROR: %v", err.Error())

		return err
	}

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
			log.Printf("ERROR: %v", err.Error())

			return err
		}
		log.Println("checking container state, status:", status)
		if status == "running" {
			log.Println("Container updated successfully")
			check = true
			break
		} else {
			//if not "running"
			log.Printf("WARNING: shidai status is %v, trying again. Attempt %v out of %v", status, i+1, attempts)
			time.Sleep(time.Second)
		}
	}
	exportedGenesisPath := path.Join(types.CONTAINERIZED_SEKAI_CONFIG, "exported.json")
	log.Println("executing sekaid export")

	cmd, err := shlex.Split(fmt.Sprintf("/sekaid export --home=%v --output-document=%v", types.CONTAINERIZED_SEKAI_HOME, exportedGenesisPath))
	if err != nil {
		log.Printf("ERROR: %v", err.Error())

		return err
	}
	_, err = docker.ExecInContainerV2(ctx, cli, types.SEKAI_CONTAINER_ID, cmd)
	if err != nil {
		log.Printf("ERROR: %v", err.Error())

		return err
	}
	log.Println("exported json to", exportedGenesisPath)

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
	//err =  dockercompose.GetDockerComposeProject(composeData)

	log.Println("updating sekai image field")

	dockercompose.UpdateComposeYMLField(currentCompose, "sekai", "image", fmt.Sprintf("%v:%v", "ghcr.io/kiracore/sekin/sekai", plan.Plan.Resources[0].Version))

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

	for i := range attempts {
		status, err := docker.CheckContainerState(cli, types.SEKAI_CONTAINER_ID)
		if err != nil {
			fmt.Println("Error checking container status:", err)
			return err
		}
		log.Println("checking container state, status:", status)
		if status == "running" {
			log.Println("Container updated successfully")
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
		log.Println("ERROR: sekai container cannot start")
		return fmt.Errorf("ERROR: sekai container cannot start")
	}

	genesisPath := path.Join(types.CONTAINERIZED_SEKAI_CONFIG, "genesis.json")

	cmd, err = shlex.Split(fmt.Sprintf("/sekaid new-genesis-from-exported %v %v --home=%v --json-minimize=false", exportedGenesisPath, genesisPath, types.CONTAINERIZED_SEKAI_HOME))
	if err != nil {
		log.Printf("ERROR: %v", err.Error())

		return err
	}
	_, err = docker.ExecInContainerV2(ctx, cli, types.SEKAI_CONTAINER_ID, cmd)
	if err != nil {
		log.Printf("ERROR: %v", err.Error())

		return err
	}
	log.Println("deleting data folder content")
	// err = utils.RemoveFolderContent(path.Join(types.SEKAI_HOME, "data"))
	// if err != nil {
	// 	return err
	// }
	sekaiDataFolder := path.Join(types.SEKAI_HOME, "data")
	// err = docker.DeleteFileOrFolder(cli, types.SEKAI_CONTAINER_ID, sekaiDataFolder)
	// if err != nil {
	// 	return err
	// }
	err = os.RemoveAll(sekaiDataFolder)
	if err != nil {
		return err
	}
	// err = setEmptyValidatorState(types.SEKAI_HOME)
	// if err != nil {
	// 	return err
	// }

	err = os.Mkdir(sekaiDataFolder, originalPerm)
	if err != nil {
		return err
	}

	emptyState := `
	{
		"height": "0",
		"round": 0,
		"step": 0
	}`
	err = docker.CreateFileInContainer(cli, types.SEKAI_CONTAINER_ID, filepath.Join(types.CONTAINERIZED_SEKAI_HOME, "data", "priv_validator_state.json"), []byte(emptyState))
	if err != nil {
		return err
	}
	log.Println("starting sekai")
	err = startSekai()
	if err != nil {
		return err
	}
	log.Println("deleting upgrade plan")
	err = utils.DeleteFile(types.UPDATE_PLAN)
	if err != nil {
		return err
	}
	return nil
}

// func setEmptyValidatorState(sekaidHome string) error {
// 	emptyState := `
// 	{
// 		"height": "0",
// 		"round": 0,
// 		"step": 0
// 	}`
// 	sekaidDataFolder := sekaidHome + "/data"
// 	err := os.Mkdir(sekaidDataFolder, 0755)
// 	if err != nil {
// 		if !os.IsExist(err) {
// 			return fmt.Errorf("unable to create <%s> folder, err: %w", sekaidDataFolder, err)
// 		}
// 	}
// 	// utils.CreateFileWithData(sekaidDataFolder+"/priv_validator_state.json", []byte(emptyState))
// 	file, err := os.Create(sekaidDataFolder + "/priv_validator_state.json")
// 	if err != nil {
// 		return fmt.Errorf("failed to create file: %w", err)
// 	}
// 	defer file.Close()

// 	_, err = file.Write([]byte(emptyState))
// 	if err != nil {
// 		return fmt.Errorf("failed to write data to file: %w", err)
// 	}

// 	log.Println("empty validator state created in:", sekaidDataFolder)
// 	return nil
// }

type CommandRequest struct {
	Command string      `json:"command"`
	Args    interface{} `json:"args"`
}

func startSekai() error {

	cmd := CommandRequest{
		Command: "start",
		Args: map[string]interface{}{
			"home": types.CONTAINERIZED_SEKAI_HOME,
		},
	}
	_, err := executeCallerCommand("localhost", "8080", "POST", cmd)
	if err != nil {
		if errors.Is(err, io.EOF) {
		} else {
			return fmt.Errorf("unable execute <%v> request, error: %w", cmd, err)
		}
	}

	return nil
}

func executeCallerCommand(address, port, method string, commandRequest CommandRequest) ([]byte, error) {

	p, err := strconv.Atoi(port)
	if err != nil {
		return nil, fmt.Errorf("port conversion error for value: <%v>", port)
	}
	if !utils.ValidatePort(p) {
		return nil, fmt.Errorf("port validation failed for value: <%v>", port)
	}

	jsonData, err := json.Marshal(commandRequest)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, fmt.Sprintf("http://%v:%v/api/execute", address, port), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}
