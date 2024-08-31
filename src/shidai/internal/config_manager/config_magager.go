package configmanager

import (
	"fmt"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/kiracore/sekin/src/shidai/internal/logger"
	"github.com/kiracore/sekin/src/shidai/internal/types"
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

	var appToml types.AppConfig
	_, err := toml.DecodeFile(appTomlPath, &appToml)
	if err != nil {
		return nil, err
	}

	return &appToml, nil
}

func GetConfigToml(sekaiHome string) (*types.Config, error) {
	configTomlPath := filepath.Join(sekaiHome, "config", "config.toml")
	log.Debug("Getting config.toml from", zap.String("path", configTomlPath))

	var cfgToml types.Config
	_, err := toml.DecodeFile(configTomlPath, &cfgToml)
	if err != nil {
		return nil, fmt.Errorf("error when unmarshaling <%s>: %w", configTomlPath, err)
	}

	return &cfgToml, nil
}
