package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/kiracore/sekin/src/shidai/internal/commands"
	interxhandler "github.com/kiracore/sekin/src/shidai/internal/interx_handler"
	"github.com/kiracore/sekin/src/shidai/internal/logger"
	"github.com/kiracore/sekin/src/shidai/internal/types"
	"github.com/kiracore/sekin/src/shidai/internal/update"
	"go.uber.org/zap"
)

var (
	log *zap.Logger = logger.GetLogger()
)

func Serve() {

	router := gin.New()
	router.Use(gin.Recovery())

	router.POST("/api/execute", commands.ExecuteCommandHandler)
	router.GET("/logs/shidai", streamLogs(types.ShidaiLogPath))
	router.GET("/logs/sekai", streamLogs(types.SekaiLogPath))
	router.GET("/logs/interx", streamLogs(types.InterxLogPath))
	router.GET("/status", infraStatus())
	router.GET("/dashboard", getDashboardHandler())
	router.POST("/config", getCurrentConfigs())
	router.PUT("/config", setConfig())

	updateContext := context.Background()

	go backgroundUpdate()
	go update.UpdateRunner(updateContext)
	go interxhandler.AddrbookManager(context.Background())
	if err := router.Run(":8282"); err != nil {
		log.Error("Failed to start the server", zap.Error(err))
	}
}
