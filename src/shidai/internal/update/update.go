package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kiracore/sekin/src/shidai/internal/types"
	githubhelper "github.com/kiracore/sekin/src/shidai/internal/update/github_helper"
)

// var log = logger.GetLogger() // Initialize the logger instance at the package level

type Github interface {
	GetLatestSekinVersion() (*types.SekinPackagesVersion, error)
}

func UpdateRunner(ctx context.Context) {
	updateInterval := time.Hour * 24
	ticker := time.NewTicker(updateInterval)
	defer ticker.Stop()
	gh := githubhelper.GithubTestHelper{}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			UpdateOrUpgrade(gh)
		}

	}

}

// checks for updates and executes updates if needed (auto-update only for shidai)
func UpdateOrUpgrade(gh Github) error {
	// log.Debug("Checking for update")
	latest, err := gh.GetLatestSekinVersion()
	if err != nil {
		return err
	}

	current, err := getCurrentVersions()
	if err != nil {
		return err
	}
	fmt.Println("Current", current)
	fmt.Println("Latest", latest)

	//compare latest and current
	// if shidai have newer version //Create update plan and execute updater bin

	// log.Debug("SEKIN LATEST PACKAGES", zap.Any("sekin", sekin))
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

func executeUpdaterBin() {

}
