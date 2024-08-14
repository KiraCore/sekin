package main

import (
	"log"

	upgrademanager "github.com/kiracore/sekin/src/updater/internal/upgrade_manager"
)

func main() {
	err := upgrademanager.GetUpgrade()
	if err != nil {
		log.Printf("ERROR: %v", err.Error())
		panic(err)
	}
}
