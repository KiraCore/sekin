package upgrademanager

import (
	"github.com/kiracore/sekin/src/updater/internal/upgrade_manager/update"
	"github.com/kiracore/sekin/src/updater/internal/upgrade_manager/upgrade"
	"github.com/kiracore/sekin/src/updater/internal/utils"
)

const update_plan string = "./upgradePlan.json"

// const sekin_home string = "/home/km/sekin"
const sekin_home string = "/home/peepo/Projects/Go/KIRA/sekin/"

func GetUpgrade() error {
	exist := utils.FileExists(update_plan)
	if exist {
		plan, err := update.CheckUpgradePlan(update_plan)
		if err != nil {
			return err
		}
		defer utils.DeleteFile(update_plan)
		upgrade.ExecuteUpgradePlan(plan)
		if err != nil {
			return err
		}
	} else {
		newVersion, err := update.CheckShidaiUpdate()
		if err != nil {
			return err
		}
		if newVersion != nil {
			err := upgrade.UpgradeShidai(sekin_home, *newVersion)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
