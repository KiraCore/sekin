package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/kiracore/sekin/src/shidai/internal/logger"
	"github.com/kiracore/sekin/src/shidai/internal/types"
	"github.com/kiracore/sekin/src/shidai/internal/types/endpoints/interx"
	githubhelper "github.com/kiracore/sekin/src/shidai/internal/update/github_helper"
	upgradehandler "github.com/kiracore/sekin/src/shidai/internal/update/upgrade_handler"
	"github.com/kiracore/sekin/src/shidai/internal/utils"
	"go.uber.org/zap"
)

var log = logger.GetLogger() // Initialize the logger instance at the package level

const (
	Lower  = "LOWER"
	Higher = "HIGHER"
	Same   = "SAME"
)

type ComparisonResult struct {
	Sekai  string
	Interx string
	Shidai string
}

type Github interface {
	GetLatestSekinVersion() (*types.SekinPackagesVersion, error)
}

// Update check runner (run in goroutine)
func UpdateRunner(ctx context.Context) {
	normalUpdateInterval := time.Hour * 24
	errorUpdateInterval := time.Hour * 3

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
			} else {
				err := SekaiUpdateOrUpgrade()
				if err != nil {
					log.Warn("Error when executing sekai upgrade:", zap.Error(err))
					ticker.Reset(errorUpdateInterval)

				} else {
					ticker.Reset(normalUpdateInterval)
				}

			}
		}

	}

}

// checks for updates and executes updates if needed (auto-update only for shidai)
func SekinUpdateOrUpgrade(gh Github) error {
	log.Info("Checking for update")
	latest, err := gh.GetLatestSekinVersion()
	if err != nil {
		return err
	}

	current, err := getCurrentVersions()
	if err != nil {
		return err
	}

	results, err := Compare(current, latest)
	if err != nil {
		return err
	}

	log.Debug("SEKIN VERSIONS:", zap.Any("latest", latest), zap.Any("current", current))
	log.Debug("RESULT:", zap.Any("result", results))

	if results.Shidai == Lower {
		err = executeUpdaterBin()
		if err != nil {
			return err
		}
	} else {
		log.Info("shidai update not required:", zap.Any("results", results))
	}

	return nil
}

func SekaiUpdateOrUpgrade() error {
	log.Info("Checking for hard fork")
	plan, err := upgradehandler.CheckHardFork(context.Background(), types.INTERX_CONTAINER_ADDRESS)
	if err != nil {
		return err
	}
	if plan != nil {
		err = writeUpgradePlanToFile(plan, types.UPGRADE_PLAN_FILE_PATH)
		if err != nil {
			return err
		}
		err = executeUpdaterBin()
		if err != nil {
			return err
		}
	} else {
		log.Info("No hard fork upgrade staged")
	}
	return nil
}

func writeUpgradePlanToFile(plan *interx.PlanData, path string) error {
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

func getCurrentVersions() (*types.SekinPackagesVersion, error) {
	out, err := http.Get("http://localhost:8282/status")
	if err != nil {
		return nil, err
	}
	defer out.Body.Close()

	b, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, err
	}
	var status types.StatusResponse

	err = json.Unmarshal(b, &status)
	if err != nil {
		// fmt.Println(string(b))
		return nil, err
	}

	pkgVersions := types.SekinPackagesVersion{
		Sekai:  strings.ReplaceAll(status.Sekai.Version, "\n", ""),
		Interx: strings.ReplaceAll(status.Interx.Version, "\n", ""),
		Shidai: strings.ReplaceAll(status.Shidai.Version, "\n", ""),
	}

	return &pkgVersions, nil
}

func executeUpdaterBin() error {
	cmd := exec.Command(types.UPDATER_BIN_PATH)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to execute binary: %w, output: %s", err, output)
	}
	return nil
}

// version has to be in format "v0.4.49"
// CompareVersions compares two version strings and returns 1 if v1 > v2, -1 if v1 < v2, and 0 if they are equal.
//
//	if v1 > v2 = higher, if v1 < v2 = lower else equal
func CompareVersions(v1, v2 string) (string, error) {
	major1, minor1, patch1, err := utils.ParseVersion(v1)
	if err != nil {
		return "", err
	}

	major2, minor2, patch2, err := utils.ParseVersion(v2)
	if err != nil {
		return "", err
	}

	if major1 > major2 {
		return Higher, nil
	} else if major1 < major2 {
		return Lower, nil
	}

	if minor1 > minor2 {
		return Higher, nil
	} else if minor1 < minor2 {
		return Lower, nil
	}

	if patch1 > patch2 {
		return Higher, nil
	} else if patch1 < patch2 {
		return Lower, nil
	}

	return Same, nil
}

// version has to be in format "v0.4.49"
// CompareVersions compares two version strings and returns 1 if v1 > v2, -1 if v1 < v2, and 0 if they are equal.
//
//	if v1 > v2 = higher, if v1 < v2 = lower else equal
//
// Compare compares two SekinPackagesVersion instances and returns the differences, including version comparison.
func Compare(current, latest *types.SekinPackagesVersion) (ComparisonResult, error) {
	var result ComparisonResult
	var err error

	result.Sekai, err = CompareVersions(current.Sekai, latest.Sekai)
	if err != nil {
		return result, err
	}

	result.Interx, err = CompareVersions(current.Interx, latest.Interx)
	if err != nil {
		return result, err
	}

	result.Shidai, err = CompareVersions(current.Shidai, latest.Shidai)
	if err != nil {
		return result, err
	}

	return result, nil
}
