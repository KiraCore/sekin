package update

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/kiracore/sekin/src/shidai/internal/logger"
	sekaihelper "github.com/kiracore/sekin/src/shidai/internal/sekai_handler/sekai_helper"
	"github.com/kiracore/sekin/src/shidai/internal/types"
	"github.com/kiracore/sekin/src/shidai/internal/types/endpoints/interx"
	githubhelper "github.com/kiracore/sekin/src/shidai/internal/update/github_helper"
	upgradehandler "github.com/kiracore/sekin/src/shidai/internal/update/upgrade_handler"
	versioncontroll "github.com/kiracore/sekin/src/shidai/internal/update/version_controll"
	"go.uber.org/zap"
)

var log = logger.GetLogger() // Initialize the logger instance at the package level

type Github interface {
	GetLatestSekinVersion() (*types.SekinPackagesVersion, error)
}

// Update check runner (run in goroutine)
func UpdateRunner(ctx context.Context) {
	log.Info("Starting upgrade runner")
	normalUpdateInterval := time.Hour * 6
	errorUpdateInterval := time.Hour * 3
	hardforkStagedInterval := time.Minute * 20

	// TODO: FOR TESTING PURPOSES, DELETE AFTER
	// normalUpdateInterval = time.Second * 20
	// errorUpdateInterval = time.Second * 20
	// hardforkStagedInterval = time.Second * 20

	ticker := time.NewTicker(normalUpdateInterval)
	defer ticker.Stop()
	gh := githubhelper.ComposeFileParser{}

	// TODO: should we run update immediately after start or after 24h
	// err := UpdateOrUpgrade(gh)
	// if err != nil {
	// 	log.Warn("Error when executing update:", zap.Error(err))
	// }
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := SekinUpdateOrUpgrade(gh)
			if err != nil {
				log.Warn("Error when executing update:", zap.Error(err))
				ticker.Reset(errorUpdateInterval)
			}
			sekaiStaged, err := SekaiUpdateOrUpgrade()
			if err != nil {
				log.Warn("Error when executing sekai upgrade:", zap.Error(err))
				ticker.Reset(errorUpdateInterval)
			}
			if sekaiStaged != nil && *sekaiStaged {
				ticker.Reset(hardforkStagedInterval)
			} else {
				ticker.Reset(normalUpdateInterval)
			}

			interxStaged, err := InterxUpdateOrUpgrade()
			if err != nil {
				log.Warn("Error when executing interx upgrade:", zap.Error(err))
				ticker.Reset(errorUpdateInterval)
			} else {
				if interxStaged {
					log.Debug("interx update staged")
					ticker.Reset(hardforkStagedInterval)
				} else {
					log.Debug("no interx updates are staged")
					ticker.Reset(normalUpdateInterval)
				}
			}

		}

	}

}

// checks and performs interx updates
func InterxUpdateOrUpgrade() (bool, error) {
	log.Info("Checking for Interx update plans")
	plan, err := upgradehandler.CheckInterxUpgrade(context.Background(), types.INTERX_CONTAINER_ADDRESS)
	log.Debug("Plan after check", zap.Any("plan", plan))
	if err != nil {
		return false, err
	}
	if plan == nil {
		log.Info("No interx updates are staged")
		return false, nil
	}
	planTime := plan.Plan.UpgradeTime
	upgradeTime, err := strconv.Atoi(planTime)
	if err != nil {
		return false, err
	}
	currentTime := time.Now().Unix()
	log.Debug("checking upgrade time", zap.Int64("current", currentTime), zap.Int64("upgrade", int64(upgradeTime)))
	if currentTime >= int64(upgradeTime) {
		err = writeUpgradePlanToFile(plan, types.UPGRADE_PLAN_FILE_PATH)
		if err != nil {
			return false, err
		}
		err = executeUpdaterBin()
		if err != nil {
			return false, err
		}
	} else {
		return true, nil
	}

	return false, nil
}

// checks for updates and executes updates if needed (auto-update only for shidai)
func SekinUpdateOrUpgrade(gh Github) error {
	log.Info("Checking for update")
	latest, err := gh.GetLatestSekinVersion()
	if err != nil {
		return err
	}

	current, err := upgradehandler.GetCurrentVersions()
	if err != nil {
		return err
	}

	results, err := versioncontroll.Compare(current, latest)
	if err != nil {
		return err
	}

	log.Debug("SEKIN VERSIONS:", zap.Any("latest", latest), zap.Any("current", current))
	log.Debug("RESULT:", zap.Any("result", results))

	if results.Shidai == versioncontroll.Lower {
		err = executeUpdaterBin()
		if err != nil {
			return err
		}
	} else {
		log.Info("shidai update not required:", zap.Any("results", results))
	}

	return nil
}

// returns bool to track if plan exist but consensus is still running and error
func SekaiUpdateOrUpgrade() (*bool, error) {
	log.Info("Checking for hard fork")
	plan, err := upgradehandler.CheckHardFork(context.Background(), types.INTERX_CONTAINER_ADDRESS)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		log.Info("No hard fork upgrade staged")
		return nil, nil
	}

	current, err := upgradehandler.GetCurrentVersions()
	if err != nil {
		return nil, err
	}

	var status string

	//check if resources are not nil
	if len(plan.Plan.Resources) > 0 && plan.Plan.Resources[0] != (interx.UpgradePlanResource{}) {
		log.Debug("Comparing versions", zap.Strings("current and plan version", []string{current.Sekai, plan.Plan.Resources[0].Version}))
		status, err = versioncontroll.CompareVersions(current.Sekai, plan.Plan.Resources[0].Version)
		if err != nil {
			return nil, err
		}
	} else {
		log.Debug("resources in upgrade plan empty")
		return nil, types.ErrResourcePlanIsEmpty
	}

	log.Debug("versions status", zap.String("status", status))

	if status != versioncontroll.Lower {
		log.Debug("status != Lower")
		return nil, nil
	}

	consensus, err := sekaihelper.CheckConsensus(context.Background(), types.SEKAI_CONTAINER_ADDRESS, strconv.Itoa(types.DEFAULT_RPC_PORT), time.Second*30)
	if err != nil {
		return nil, err
	}
	if !consensus {
		err = writeUpgradePlanToFile(plan, types.UPGRADE_PLAN_FILE_PATH)
		if err != nil {
			return nil, err
		}
		err = executeUpdaterBin()
		if err != nil {
			return nil, err
		}
	} else {
		return &consensus, nil
	}

	return nil, nil
}

func writeUpgradePlanToFile(plan *interx.PlanData, path string) error {
	log.Debug("creating upgrade plan", zap.String("path", path), zap.Any("play", plan))
	jsonData, err := json.Marshal(plan)
	if err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer file.Close()

	_, err = file.Write(jsonData)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func executeUpdaterBin() error {
	log.Debug("Executing update binary", zap.String("bin path", types.UPDATER_BIN_PATH))

	cmd := exec.Command(types.UPDATER_BIN_PATH)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to execute binary: %w, output: %s", err, output)
	}
	return nil
}
