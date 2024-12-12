package configmanager

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/BurntSushi/toml"
	"github.com/kiracore/sekin/src/shidai/internal/logger"
	"github.com/kiracore/sekin/src/shidai/internal/types"
	"github.com/kiracore/sekin/src/shidai/internal/utils"
	"go.uber.org/zap"
)

type CombinedConfig struct {
	ConfigToml types.Config    `toml:"config_toml"`
	AppToml    types.AppConfig `toml:"app_toml"`
}

var (
	log = logger.GetLogger()
)

func GetCombinedConfig(sekaiHome string) (*CombinedConfig, error) {
	log.Debug("Getting Combined config ", zap.String("sekaiHome", sekaiHome))
	cfgToml, err := GetConfigToml(sekaiHome)
	if err != nil {
		return nil, err
	}
	appToml, err := GetAppToml(sekaiHome)
	if err != nil {
		return nil, err
	}
	return &CombinedConfig{ConfigToml: *cfgToml, AppToml: *appToml}, nil
}

func GetAppToml(sekaiHome string) (*types.AppConfig, error) {
	appTomlPath := filepath.Join(sekaiHome, "config", "app.toml")
	log.Debug("Getting app.toml from", zap.String("path", appTomlPath))

	content, err := os.ReadFile(appTomlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file <%s>: %w", appTomlPath, err)
	}
	var appToml types.AppConfig
	_, err = toml.Decode(appTomlPath, &appToml)
	if err == nil {
		return &appToml, nil
	}

	log.Warn("Direct decoding failed, attempting fallback", zap.Error(err))
	var rawData map[string]interface{}

	if _, decodeErr := toml.Decode(string(content), &rawData); decodeErr != nil {
		return nil, fmt.Errorf("error decoding TOML during fallback: %w", decodeErr)
	}

	if err := appTomlConvertor(rawData); err != nil {
		return nil, fmt.Errorf("error transforming config data: %w", err)
	}

	var buffer bytes.Buffer
	if err := toml.NewEncoder(&buffer).Encode(rawData); err != nil {
		return nil, fmt.Errorf("error re-encoding transformed data: %w", err)
	}

	if _, finalDecodeErr := toml.Decode(buffer.String(), &appToml); finalDecodeErr != nil {
		return nil, fmt.Errorf("error decoding transformed data into struct: %w", finalDecodeErr)
	}
	return &appToml, nil
}

func GetConfigToml(sekaiHome string) (*types.Config, error) {
	configTomlPath := filepath.Join(sekaiHome, "config", "config.toml")
	log.Debug("Getting config.toml from", zap.String("path", configTomlPath))
	content, err := os.ReadFile(configTomlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file <%s>: %w", configTomlPath, err)
	}

	var cfgToml types.Config
	_, err = toml.Decode(string(content), &cfgToml)
	if err == nil {
		return &cfgToml, nil
	}
	log.Warn("Direct decoding failed, attempting fallback", zap.Error(err))

	var rawData map[string]interface{}

	if _, decodeErr := toml.Decode(string(content), &rawData); decodeErr != nil {
		return nil, fmt.Errorf("error decoding TOML during fallback: %w", decodeErr)
	}

	if err := configTomlConvertor(rawData); err != nil {
		return nil, fmt.Errorf("error transforming config data: %w", err)
	}

	var buffer bytes.Buffer
	if err := toml.NewEncoder(&buffer).Encode(rawData); err != nil {
		return nil, fmt.Errorf("error re-encoding transformed data: %w", err)
	}

	if _, finalDecodeErr := toml.Decode(buffer.String(), &cfgToml); finalDecodeErr != nil {
		return nil, fmt.Errorf("error decoding transformed data into struct: %w", finalDecodeErr)
	}
	return &cfgToml, nil
}

func SetAppToml(cfg types.AppConfig, sekaiHome string) error {
	appTomlPath := filepath.Join(sekaiHome, "config", "app.toml")

	err := utils.SaveAppConfig(appTomlPath, cfg)
	if err != nil {
		return err
	}

	return nil
}

func SetConfigToml(cfg types.Config, sekaiHome string) error {
	configTomlPath := filepath.Join(sekaiHome, "config", "config.toml")

	err := utils.SaveConfig(configTomlPath, cfg)
	if err != nil {
		return err
	}
	return nil
}

func configTomlConvertor(data map[string]interface{}) error {
	if section, ok := data["statesync"].(map[string]interface{}); ok {
		convertFieldFromNumToString(section, "chunk_fetchers")
	}
	return nil
}
func appTomlConvertor(data map[string]interface{}) error {
	convertFieldFromNumToString(data, "pruning-keep-recent")
	convertFieldFromNumToString(data, "pruning-keep-every")
	convertFieldFromNumToString(data, "pruning-interval")
	return nil
}
func convertFieldFromNumToString(data map[string]interface{}, field string) {
	if value, exists := data[field]; exists {
		switch v := value.(type) {
		case int:
			data[field] = strconv.Itoa(v) // Convert int to string
		case int64:
			data[field] = strconv.FormatInt(v, 10) // Convert int64 to string
		case float64:
			data[field] = strconv.Itoa(int(v)) // Convert float64 to string
		case string:
			// Already a string, no conversion needed
		}
	}
}
