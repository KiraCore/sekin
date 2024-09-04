package api

import (
	"fmt"
	"net/http"

	"github.com/BurntSushi/toml"
	"github.com/gin-gonic/gin"
	configmanager "github.com/kiracore/sekin/src/shidai/internal/config_manager"
	"github.com/kiracore/sekin/src/shidai/internal/types"
	"github.com/kiracore/sekin/src/shidai/internal/utils"
	"go.uber.org/zap"
)

// send toml data as encoded struct to string bytes
type ConfigRequest struct {
	Type string `json:"type"`
	// TomlData []byte `json:"toml_data"`
	TomlData string `json:"toml_data"`
}

const (
	ConfigTomlType = "config_toml"
	AppTomlType    = "app_toml"
)

func getCurrentConfigs() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ConfigRequest

		err := c.BindJSON(&req)
		if err != nil {
			log.Debug("error when binding json", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"details": fmt.Sprintf("error: %+v", err), "error": "invalid request"})
			return
		}

		log.Debug("Getting  current configs", zap.String("req.Type", req.Type))
		switch req.Type {
		case AppTomlType:
			cfg, err := configmanager.GetAppToml(types.SEKAI_HOME)
			if err != nil {
				c.Error(err)
				return
			}
			c.TOML(http.StatusOK, cfg)

		case ConfigTomlType:
			cfg, err := configmanager.GetConfigToml(types.SEKAI_HOME)
			if err != nil {
				c.Error(err)
				return
			}
			c.TOML(http.StatusOK, cfg)
		default:
			log.Error("", zap.Error(types.ErrInvalidRequest))
			c.Error(types.ErrInvalidRequest)
			return
		}
		// cfg, err := configmanager.GetCombinedConfig(types.SEKAI_HOME)
		// if err != nil {
		// 	c.Error(err)
		// }
	}
}

func setConfig() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ConfigRequest

		if err := c.BindJSON(&req); err != nil {
			log.Error("error when binding toml", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"details": fmt.Sprintf("error: %+v", err), "error": "invalid request"})
			return
		}

		switch req.Type {
		case AppTomlType:
			var appToml types.AppConfig
			_, err := toml.Decode(string(req.TomlData), &appToml)
			if err != nil {
				log.Error("error when decoding app toml", zap.Error(err))
				c.JSON(http.StatusBadRequest, gin.H{"details": fmt.Sprintf("error: %+v", err), "error": "when decoding app toml"})
				return
			}
			log.Debug("request to edit app.toml file", zap.Any("configToml", appToml))

			err = utils.ValidateToml([]byte(req.TomlData), &appToml)
			if err != nil {
				log.Error("validation failed", zap.Error(err))
				c.JSON(http.StatusBadRequest, gin.H{"details": fmt.Sprintf("error: %+v", err), "error": "validation failed"})
				return
			}

			err = configmanager.SetAppToml(appToml, types.SEKAI_HOME)
			if err != nil {
				log.Error("error when setting app toml", zap.Error(err))
				c.JSON(http.StatusBadRequest, gin.H{"details": fmt.Sprintf("error: %+v", err), "error": "error when setting app toml"})
				return
			}
		case ConfigTomlType:
			var configToml types.Config
			_, err := toml.Decode(string(req.TomlData), &configToml)
			if err != nil {
				log.Error("error when decoding config toml", zap.Error(err))
				c.JSON(http.StatusBadRequest, gin.H{"details": fmt.Sprintf("error: %+v", err), "error": "error when decoding config toml"})
				return
			}
			log.Debug("request to edit config.toml file", zap.Any("configToml", configToml))

			err = utils.ValidateToml([]byte(req.TomlData), &configToml)
			if err != nil {
				log.Error("validation failed", zap.Error(err))
				c.JSON(http.StatusBadRequest, gin.H{"details": fmt.Sprintf("error: %+v", err), "error": "validation failed"})
				return
			}
			err = configmanager.SetConfigToml(configToml, types.SEKAI_HOME)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"details": fmt.Sprintf("error: %+v", err), "error": "error when setting config toml"})
				return
			}
		default:
			log.Error("", zap.Error(types.ErrInvalidRequest))
			c.JSON(http.StatusBadRequest, gin.H{"details": fmt.Sprintf("error: %+v", types.ErrInvalidRequest), "error": "invalid request"})
			return
		}
		// ctx.Params
	}
}
