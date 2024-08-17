package versioncontroll

import (
	"github.com/kiracore/sekin/src/shidai/internal/types"
	"github.com/kiracore/sekin/src/shidai/internal/utils"
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
