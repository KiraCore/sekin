package upgradehandler

import (
	"context"
	"strconv"

	"github.com/kiracore/sekin/src/shidai/internal/types"
	"github.com/kiracore/sekin/src/shidai/internal/types/endpoints/interx"
	"github.com/kiracore/sekin/src/shidai/internal/utils"
	"go.uber.org/zap"
)

const SEKAI_ID_RESOURCE = "sekai"

// returns error and upgrade plan if plan is hard fork
func CheckHardFork(ctx context.Context, interxAddress string) (*interx.PlanData, error) {
	currentPlan, err := CheckCurrentUpgradePlan(ctx, interxAddress, strconv.Itoa(types.DEFAULT_INTERX_PORT))
	if err != nil {
		return nil, err
	}
	nextPlan, err := CheckNextUpgradePlan(ctx, interxAddress, strconv.Itoa(types.DEFAULT_INTERX_PORT))
	if err != nil {
		return nil, err
	}
	var plan *interx.PlanData
	log.Debug("Hard fork check", zap.Any("Current plan", currentPlan), zap.Any("Next plan", nextPlan))
	//TODO: what if upgrade already happen and it stuck in current plan?
	if currentPlan != nil {
		plan = currentPlan
	} else {
		if nextPlan != nil {
			plan = nextPlan
		} else {
			return nil, nil
		}
	}

	hardFork := checkIfPlanIsHardFork(plan)

	if hardFork {
		return plan, nil
	} else {
		return nil, nil
	}
}

// upgrade plan resources should be valid:
//
// id of hardfork upgrade should be "sekai" with valid version "v0.4.1"
func checkIfPlanIsHardFork(plan *interx.PlanData) bool {
	if !plan.Plan.InstateUpgrade && !plan.Plan.SkipHandler && plan.Plan.RebootRequired {
		if plan.Plan.Resources[0].ID == SEKAI_ID_RESOURCE {
			_, _, _, err := utils.ParseVersion(plan.Plan.Resources[0].Version)
			if err == nil {
				return true
			}
		}
	}
	return false
}

func ParseUpgradePlan(plan *interx.PlanData) error {

	return nil
}
