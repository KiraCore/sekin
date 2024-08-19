package upgrade

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/docker/docker/client"
	"github.com/kiracore/sekin/src/updater/internal/types"
	"github.com/kiracore/sekin/src/updater/internal/upgrade_manager/docker"
	dockercompose "github.com/kiracore/sekin/src/updater/internal/upgrade_manager/docker_compose"
	upgradeplanhandler "github.com/kiracore/sekin/src/updater/internal/upgrade_manager/upgrade/upgrade_plan_handler"
	"github.com/kiracore/sekin/src/updater/internal/utils"
	"gopkg.in/yaml.v2"
)

const SekinComposeFileURL_main_branch string = "https://raw.githubusercontent.com/KiraCore/sekin/main/compose.yml"

const ShidaiServiceName string = "shidai"
const ShidaiContainerName string = "sekin-" + ShidaiServiceName + "-1"

func ExecuteUpgradePlan(plan *types.PlanData, cli *client.Client) error {
	log.Printf("Executing upgrade plan: %+v", plan)
	hardfork := upgradeplanhandler.CheckIfPlanIsHardFork(plan)
	softfork := upgradeplanhandler.CheckIfPlanIsInterxUpgrade(plan)
	switch {
	case hardfork:
		err := upgradeplanhandler.ExecuteSekaiHardForkUpgrade(plan, cli)
		if err != nil {
			return err
		}
	case softfork:
		err := upgradeplanhandler.ExecuteInterxSoftForkUpgrade(plan, cli)
		if err != nil {
			return err
		}
	}
	return nil
}

func UpgradeShidai(cli *client.Client, sekinHome, version string) error {
	log.Printf("Trying to upgrade shidai, path: <%v>", sekinHome)

	composeFilePath := filepath.Join(sekinHome, "compose.yml")
	backupComposeFilePath := filepath.Join(sekinHome, "compose.yml.bak")

	// exist := utils.FileExists(backupComposeFilePath)
	// if !exist {
	err := utils.CopyFile(composeFilePath, backupComposeFilePath)
	if err != nil {
		return err
	}
	// } else {
	// 	log.Printf("WARNING: backup file already exist, configuring from old backup file")
	// }

	bakData, err := os.ReadFile(backupComposeFilePath)
	if err != nil {
		return nil
	}

	var bakCompose map[string]interface{}
	err = yaml.Unmarshal(bakData, &bakCompose)
	if err != nil {
		fmt.Println("Error unmarshalling YAML:", err)
		return err
	}

	currentSekaiImage, err := dockercompose.ReadComposeYMLField(bakCompose, "sekai", "image")
	if err != nil {
		fmt.Println("Error reading field:", err)
		return err
	}
	log.Printf("sekai image: %s\n", currentSekaiImage)

	currentInterxImage, err := dockercompose.ReadComposeYMLField(bakCompose, "interx", "image")
	if err != nil {
		fmt.Println("Error reading field:", err)
		return err
	}
	log.Printf("sekai image: %s\n", currentInterxImage)

	err = downloadLatestSekinComposeFile(SekinComposeFileURL_main_branch, composeFilePath)
	if err != nil {
		return err
	}

	latestData, err := os.ReadFile(composeFilePath)
	if err != nil {
		return nil
	}

	var latestCompose map[string]interface{}
	err = yaml.Unmarshal(latestData, &latestCompose)
	if err != nil {
		fmt.Println("Error unmarshalling YAML:", err)
		return err
	}

	dockercompose.UpdateComposeYMLField(latestCompose, "sekai", "image", currentSekaiImage)
	dockercompose.UpdateComposeYMLField(latestCompose, "interx", "image", currentInterxImage)

	updatedData, err := yaml.Marshal(&latestCompose)
	if err != nil {
		log.Println("Error marshalling YAML:", err)
		return err
	}

	var originalPerm os.FileMode = 0644 // Default permission if the file doesn't exist
	if fileInfo, err := os.Stat(composeFilePath); err == nil {
		originalPerm = fileInfo.Mode()
	}

	err = os.WriteFile(composeFilePath, updatedData, originalPerm)
	if err != nil {
		log.Println("Error writing file:", err)
		return err
	}
	diff, err := dockercompose.CompareYAMLFiles(backupComposeFilePath, composeFilePath)
	if err != nil {
		return err
	}
	log.Println("DIFF", diff)

	err = dockercompose.DockerComposeUpService(sekinHome, composeFilePath)
	if err != nil {
		log.Printf("ERROR when trying to run <%v> compose file: %v ", composeFilePath, err)
		return err
	}

	var check bool = false

	attempts := 3
	for i := range attempts {

		status, err := docker.CheckContainerState(cli, ShidaiContainerName)
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
		// deleting backup file after successful upgrade
		log.Println("SHIDAI updated successfully")
		err = utils.DeleteFile(backupComposeFilePath)
		if err != nil {
			fmt.Println("Error deleting backup file:", err)
			return err
		}
		return nil
	} else {
		log.Printf("WARNING: unable to update shidai trying to run backup compose file.")
		err = dockercompose.DockerComposeUpService(sekinHome, composeFilePath)
		if err != nil {
			log.Printf("ERROR: when trying to run <%v> backup compose file: %v ", backupComposeFilePath, err)
			return fmt.Errorf("ERROR when trying to run <%v> backup compose file: %w", backupComposeFilePath, err)
		}
		log.Println("Deleting new compose file")
		err := utils.DeleteFile(composeFilePath)
		if err != nil {
			log.Printf("ERROR: when trying to delete old compose file: <%v>. Error: %v ", composeFilePath, err)
			return fmt.Errorf("ERROR when trying to delete old compose file: <%v>. Error: %v ", composeFilePath, err)
		}
		err = utils.RenameFile(backupComposeFilePath, composeFilePath)
		if err != nil {
			return fmt.Errorf("ERROR: when renaming backup file to old name: %w", err)
		}
		log.Println("WARNING: unable to run new version of shidai, update rollback to previous version")
		return nil
	}
}

func downloadLatestSekinComposeFile(composeFileURL, filepath string) error {
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(composeFileURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}
