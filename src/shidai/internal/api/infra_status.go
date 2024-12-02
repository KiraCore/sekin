package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kiracore/sekin/src/shidai/internal/docker"
	"github.com/kiracore/sekin/src/shidai/internal/types"
	"github.com/kiracore/sekin/src/shidai/internal/utils"
)

func infraStatus() gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			SekaiVersion, InterxVersion, SyslogVersion, ShidaiVersion []byte
			err                                                       error
		)
		cm, err := docker.NewContainerManager()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}
		ctx := c.Request.Context()

		if SekaiVersion, err = cm.ExecInContainer(ctx, "sekin-sekai-1", []string{"/sekaid", "version"}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get Sekai version"})
			return
		}

		if InterxVersion, err = cm.ExecInContainer(ctx, "sekin-interx-1", []string{"/interxd", "version"}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get Interx version"})
			return
		}
		if SyslogVersion, err = cm.ExecInContainer(ctx, "sekin-syslog-ng-1", []string{"syslog-ng", "--version"}); err != nil {
			// c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get Syslog version"})
			// return
			// syslog version is not used anywhere in code, but sometimes returns error instead of version which causes problems with ui not logic
			SyslogVersion = []byte(err.Error())
		}

		if ShidaiVersion, err = cm.ExecInContainer(ctx, "sekin-shidai-1", []string{"/shidai", "version"}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get Shidai version"})
			return
		}

		response := types.StatusResponse{
			Sekai:  types.AppInfo{Version: string(SekaiVersion), Infra: utils.CheckInfra(types.SekaiFiles)},
			Interx: types.AppInfo{Version: string(InterxVersion), Infra: utils.CheckInfra(types.InterxFiles)},
			Syslog: types.AppInfo{Version: string(SyslogVersion), Infra: utils.CheckInfra(types.SyslogFiles)},
			Shidai: types.AppInfo{Version: string(ShidaiVersion), Infra: true},
		}
		c.JSON(http.StatusOK, response)
	}
}
