package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	configmanager "github.com/kiracore/sekin/src/shidai/internal/config_manager"
	"github.com/kiracore/sekin/src/shidai/internal/types"
)

type ConfigRequest struct{}

func getCurrentConfigs() gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg, err := configmanager.GetCombinedConfig(types.SEKAI_HOME)
		if err != nil {
			c.Error(err)
		}
		c.TOML(http.StatusOK, cfg)
	}
}
