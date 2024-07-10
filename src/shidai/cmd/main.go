package main

import (
	"github.com/kiracore/sekin/src/shidai/internal/cli"
	"github.com/kiracore/sekin/src/shidai/internal/logger"
	"go.uber.org/zap"
)

var Version = "v0.0.1"

func main() {

	log := logger.GetLogger()
	log.Info("initializing cli ...")

	rootCmd := cli.NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		log.Warn("failed to initialize cli ...", zap.Error(err))
	}

	//api.Serve()
}
