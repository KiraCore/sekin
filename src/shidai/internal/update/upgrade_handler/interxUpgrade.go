package upgradehandler

import (
	"context"
	"strconv"

	"github.com/kiracore/sekin/src/shidai/internal/types"
	"github.com/kiracore/sekin/src/shidai/internal/types/endpoints/interx"
	versioncontroll "github.com/kiracore/sekin/src/shidai/internal/update/version_controll"
	"go.uber.org/zap"
)

const INTERX_ID_RESOURCE = "interx"

func CheckInterxUpgrade(ctx context.Context, interxAddress string) (*interx.PlanData, error) {
	currentPlan, err := CheckCurrentUpgradePlan(ctx, interxAddress, strconv.Itoa(types.DEFAULT_INTERX_PORT))
	if err != nil {
		return nil, err
	}
	nextPlan, err := CheckNextUpgradePlan(ctx, interxAddress, strconv.Itoa(types.DEFAULT_INTERX_PORT))
	if err != nil {
		return nil, err
	}
	var plan *interx.PlanData
	log.Debug("Soft fork check", zap.Any("Current plan", currentPlan), zap.Any("Next plan", nextPlan))

	currentVersions, err := GetCurrentVersions()
	if err != nil {
		return nil, err
	}
	currentCheck, err := interxPlanIsValid(currentVersions, currentPlan)
	if err != nil {
		return nil, err
	}
	nextCheck, err := interxPlanIsValid(currentVersions, nextPlan)
	if err != nil {
		return nil, err
	}

	switch {
	case currentCheck:
		plan = currentPlan
	case nextCheck:
		plan = nextPlan
	case currentCheck && nextCheck:
		plan = currentPlan
	default:
		plan = nil
	}

	return plan, nil
}

// the current version of interx has to be lower then the version mentioned in upgrade plan
func interxPlanIsValid(currentVersions *types.SekinPackagesVersion, plan *interx.PlanData) (bool, error) {
	if len(plan.Plan.Resources) == 0 || plan.Plan.Resources[0] == (interx.UpgradePlanResource{}) {
		log.Debug("resources in upgrade plan empty")
		return false, types.ErrPlanIsEmptyOrNil
	}
	status, err := versioncontroll.CompareVersions(currentVersions.Interx, plan.Plan.Resources[0].Version)
	if err != nil {
		return false, err
	}

	if plan.Plan.Resources[0].ID != INTERX_ID_RESOURCE {
		log.Debug("cant recognize upgrade plan resource ID", zap.String("plan resource id", plan.Plan.Resources[0].ID), zap.String("expected", INTERX_ID_RESOURCE))
		return false, nil
	}
	log.Debug("versions status", zap.String("status", status))

	if status != versioncontroll.Lower {
		log.Debug("status != Lower")
		return false, nil
	}
	return true, nil
}
