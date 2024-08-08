package dockercompose

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/compose-spec/compose-go/v2/loader"
	composeTypes "github.com/compose-spec/compose-go/v2/types"
	"gopkg.in/yaml.v2"
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

func GetDockerComposeProject(dockerComposeFile []byte) (*composeTypes.Project, error) {
	project, err := loader.Load(composeTypes.ConfigDetails{
		ConfigFiles: []composeTypes.ConfigFile{{Content: dockerComposeFile}},
	}, func(o *loader.Options) {
		name, projectNameImperativel := o.GetProjectName() // Set a default project name if none is provided
		if name == "" {
			o.SetProjectName("default_project", projectNameImperativel)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("error loading compose file: %w", err)
	}
	return project, nil
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
