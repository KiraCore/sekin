package main

import upgrademanager "github.com/kiracore/sekin/src/updater/internal/upgrade_manager"

func main() {
	err := upgrademanager.GetUpgrade()
	if err != nil {
		panic(err)
	}
}
