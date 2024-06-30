package upgrade

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"

	"github.com/kiracore/sekin/src/updater/internal/types"
	"github.com/kiracore/sekin/src/updater/internal/utils"
	"gopkg.in/yaml.v2"
)

const SekinComposeFileURL_main_branch string = "https://raw.githubusercontent.com/KiraCore/sekin/main/compose.yml"

func ExecuteUpgradePlan(plan *types.UpgradePlan) error {
	log.Printf("Executing upgrade plan: %+v", plan)
	return nil
}

func UpgradeShidai(sekinHome, version string) error {
	log.Printf("Trying to upgrade shidai, path: <%v>", sekinHome)

	composeFilePath := filepath.Join(sekinHome, "compose.yml")
	backupComposeFilePath := filepath.Join(sekinHome, "compose.yml.bak")

	exist := utils.FileExists(backupComposeFilePath)
	if !exist {
		err := utils.CopyFile(composeFilePath, backupComposeFilePath)
		if err != nil {
			return err
		}
	} else {
		log.Printf("WARNING: backup file already exist, configuring from old backup file")
	}

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

	currentSekaiImage, err := ReadComposeYMLField(bakCompose, "sekai", "image")
	if err != nil {
		fmt.Println("Error reading field:", err)
		return err
	}
	log.Printf("sekai image: %s\n", currentSekaiImage)

	currentInterxImage, err := ReadComposeYMLField(bakCompose, "interx", "image")
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

	UpdateComposeYMLField(latestCompose, "sekai", "image", currentSekaiImage)
	UpdateComposeYMLField(latestCompose, "interx", "image", currentInterxImage)

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
	diff, err := CompareYAMLFiles(backupComposeFilePath, composeFilePath)
	if err != nil {
		panic(err)
	}
	log.Println("DIFF", diff)
	//deleting backup file after seccusesull upgrade
	// err = utils.DeleteFile(backupComposeFilePath)
	// if err != nil {
	// 	fmt.Println("Error deleting backup file:", err)
	// 	return err
	// }
	return nil
}

func ReadComposeYMLField(compose map[string]interface{}, serviceName, fieldName string) (string, error) {
	if services, ok := compose["services"].(map[interface{}]interface{}); ok {
		if service, ok := services[serviceName].(map[interface{}]interface{}); ok {
			if value, ok := service[fieldName].(string); ok {
				return value, nil
			}
			return "", fmt.Errorf("field %s not found in service %s", fieldName, serviceName)
		}
		return "", fmt.Errorf("service %s not found", serviceName)
	}
	return "", fmt.Errorf("services section not found in compose file")
}

func UpdateComposeYMLField(compose map[string]interface{}, serviceName, fieldName, newValue string) {
	if services, ok := compose["services"].(map[interface{}]interface{}); ok {
		if service, ok := services[serviceName].(map[interface{}]interface{}); ok {
			service[fieldName] = newValue
		}
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

// TODO: this is only for testing purpose, delete after
func CompareYAMLFiles(file1Path, file2Path string) ([]string, error) {
	file1Data, err := os.ReadFile(file1Path)
	if err != nil {
		return nil, fmt.Errorf("error reading file1: %v", err)
	}
	file2Data, err := os.ReadFile(file2Path)
	if err != nil {
		return nil, fmt.Errorf("error reading file2: %v", err)
	}
	var file1Map map[interface{}]interface{}
	err = yaml.Unmarshal(file1Data, &file1Map)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling file1: %v", err)
	}
	var file2Map map[interface{}]interface{}
	err = yaml.Unmarshal(file2Data, &file2Map)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling file2: %v", err)
	}
	differences := compareMaps(file1Map, file2Map, "")
	return differences, nil
}
func compareMaps(map1, map2 map[interface{}]interface{}, prefix string) []string {
	var differences []string
	for key, value1 := range map1 {
		keyStr := fmt.Sprintf("%s.%v", prefix, key)
		value2, ok := map2[key]
		if !ok {
			differences = append(differences, fmt.Sprintf("Missing key in file2: %s", keyStr))
			continue
		}

		switch v1 := value1.(type) {
		case map[interface{}]interface{}:
			if v2, ok := value2.(map[interface{}]interface{}); ok {
				differences = append(differences, compareMaps(v1, v2, keyStr)...)
			} else {
				differences = append(differences, fmt.Sprintf("Type mismatch at key: %s", keyStr))
			}
		default:
			if !reflect.DeepEqual(v1, value2) {
				differences = append(differences, fmt.Sprintf("Different value at key: %s (file1: %v, file2: %v)", keyStr, v1, value2))
			}
		}
	}

	for key := range map2 {
		if _, ok := map1[key]; !ok {
			keyStr := fmt.Sprintf("%s.%v", prefix, key)
			differences = append(differences, fmt.Sprintf("Missing key in file1: %s", keyStr))
		}
	}

	return differences
}
