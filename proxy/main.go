package main

import (
	"github.com/saiset-co/sai-interax-proxy/internal"
	"github.com/saiset-co/sai-interax-proxy/logger"

	"github.com/saiset-co/sai-service/service"
)

func main() {
	svc := service.NewService("saiInterxProxy")
	is := internal.InternalService{Context: svc.Context}

	svc.RegisterConfig("config.yml")

	logger.Logger = svc.Logger

	is.Init()

	svc.RegisterTasks([]func(){
		is.Process,
	})

	svc.Start()
}
