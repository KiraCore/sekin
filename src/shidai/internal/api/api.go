package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/kiracore/sekin/src/shidai/internal/commands"
	"github.com/kiracore/sekin/src/shidai/internal/logger"
	prometheusexporter "github.com/kiracore/sekin/src/shidai/internal/prometheus_exporter"
	"github.com/kiracore/sekin/src/shidai/internal/types"
	"github.com/kiracore/sekin/src/shidai/internal/update"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

var (
	log *zap.Logger = logger.GetLogger()
)

func Serve() {

	router := gin.New()
	router.Use(gin.Recovery())
	prometheusCustomRegistry := prometheusexporter.RegisterMetrics()

	router.POST("/api/execute", commands.ExecuteCommandHandler)
	router.GET("/logs/shidai", streamLogs(types.ShidaiLogPath))
	router.GET("/logs/sekai", streamLogs(types.SekaiLogPath))
	router.GET("/logs/interx", streamLogs(types.InterxLogPath))
	router.GET("/status", infraStatus())
	router.GET("/dashboard", getDashboardHandler())
	router.POST("/config", getCurrentConfigs())
	router.PUT("/config", setConfig())
	router.GET("/metrics", gin.WrapH(promhttp.HandlerFor(prometheusCustomRegistry, promhttp.HandlerOpts{}))) // Custom metrics only

	updateContext := context.Background()

	go backgroundUpdate()
	go update.UpdateRunner(updateContext)
	go prometheusexporter.RunPrometheusExporterService(updateContext)

	if err := router.Run(":8282"); err != nil {
		log.Error("Failed to start the server", zap.Error(err))
	}
}
