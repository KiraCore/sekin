package upgrademanager

import (
	"log"

	"github.com/docker/docker/client"
	"github.com/kiracore/sekin/src/updater/internal/types"
	"github.com/kiracore/sekin/src/updater/internal/upgrade_manager/update"
	"github.com/kiracore/sekin/src/updater/internal/upgrade_manager/upgrade"
	"github.com/kiracore/sekin/src/updater/internal/utils"
)

func GetUpgrade() error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Printf("Error creating Docker client: %v", err)
		return err
	}

	exist := utils.FileExists(types.UPDATE_PLAN)
	if exist {
		plan, err := update.CheckUpgradePlan(types.UPDATE_PLAN)
		if err != nil {
			return err
		}
		
		err = upgrade.ExecuteUpgradePlan(plan, cli)
		if err != nil {
			return err
		}
	} else {
		newVersion, err := update.CheckShidaiUpdate()
		if err != nil {
			return err
		}
		if newVersion != nil {
			err := upgrade.UpgradeShidai(cli, types.SEKIN_HOME, *newVersion)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
