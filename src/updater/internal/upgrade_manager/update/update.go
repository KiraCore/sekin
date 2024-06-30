package update

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/kiracore/sekin/src/updater/internal/types"
	"github.com/kiracore/sekin/src/updater/internal/utils"
)

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

type GithubTestHelper struct{}

func (GithubTestHelper) GetLatestSekinVersion() (*types.SekinPackagesVersion, error) {
	return &types.SekinPackagesVersion{Sekai: "v0.3.45", Interx: "v0.4.49", Shidai: "v1.9.0"}, nil
}

func CheckUpgradePlan(path string) (*types.UpgradePlan, error) {
	exist := utils.FileExists(path)
	if !exist {
		return nil, fmt.Errorf("file <%v>  does not exist", path)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var plan types.UpgradePlan
	err = json.Unmarshal(b, &plan)
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

func CheckShidaiUpdate() (latestShidaiVersion *string, err error) {
	log.Println("Checking for update")
	gh := GithubTestHelper{}
	latest, err := gh.GetLatestSekinVersion()
	if err != nil {
		return nil, err
	}

	current, err := getCurrentVersions()
	if err != nil {
		return nil, err
	}

	results, err := compare(current, latest)
	if err != nil {
		return nil, err
	}

	log.Printf("Version status:\nCurrent: %+v\nLatest: %+v\nResults: %+v", current, latest, results)

	if results.Shidai == Lower {
		log.Printf("shidai has newer version: <%v>, current: <%v>\n", latest.Shidai, current.Shidai)
		return &latest.Shidai, nil
	} else {
		log.Printf("shidai is up to date: <%v>\n", latest.Shidai)
		return nil, nil
	}
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

// if v1 > v2 = higher, if v1 < v2 = lower else equal
func compare(current, latest *types.SekinPackagesVersion) (ComparisonResult, error) {
	var result ComparisonResult
	var err error

	result.Sekai, err = compareVersions(current.Sekai, latest.Sekai)
	if err != nil {
		return result, err
	}

	result.Interx, err = compareVersions(current.Interx, latest.Interx)
	if err != nil {
		return result, err
	}

	result.Shidai, err = compareVersions(current.Shidai, latest.Shidai)
	if err != nil {
		return result, err
	}

	return result, nil
}

// version has to be in format "v0.4.49"
// CompareVersions compares two version strings and returns 1 if v1 > v2, -1 if v1 < v2, and 0 if they are equal.
//
//	if v1 > v2 = higher, if v1 < v2 = lower else equal
func compareVersions(v1, v2 string) (string, error) {
	major1, minor1, patch1, err := parseVersion(v1)
	if err != nil {
		return "", err
	}

	major2, minor2, patch2, err := parseVersion(v2)
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

func parseVersion(version string) (major, minor, patch int, err error) {
	parts := strings.TrimPrefix(version, "v")
	components := strings.Split(parts, ".")
	if len(components) != 3 {
		return 0, 0, 0, fmt.Errorf("invalid version format: %s", version)
	}

	major, err = strconv.Atoi(components[0])
	if err != nil {
		return 0, 0, 0, err
	}

	minor, err = strconv.Atoi(components[1])
	if err != nil {
		return 0, 0, 0, err
	}

	patch, err = strconv.Atoi(components[2])
	if err != nil {
		return 0, 0, 0, err
	}

	return major, minor, patch, nil
}
