package main

import (
	"github.com/saiset-co/sai-interx-manager/internal"
	"github.com/saiset-co/sai-interx-manager/logger"

	"github.com/saiset-co/sai-service/service"
)

func main() {
	svc := service.NewService("saiInterxManager")
	is := internal.InternalService{Context: svc.Context}

	svc.RegisterConfig("config.yml")

	logger.Logger = svc.Logger

	is.Init()

	svc.RegisterTasks([]func(){
		is.Process,
	})

	svc.RegisterHandlers(
		is.NewHandler(),
	)

	svc.Start()
}
