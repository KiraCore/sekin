package upgradehandler

import (
	"context"
	"strconv"

	interxhelper "github.com/kiracore/sekin/src/shidai/internal/interx_handler/interx_helper"
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

	// if both exist if both are valid execute current
	// check current, check next, check chainID, if newer chainID exist - plan = newer chainID

	iStatus, err := interxhelper.GetInterxStatus(ctx, interxAddress, strconv.Itoa(types.DEFAULT_INTERX_PORT))
	if err != nil {
		return nil, err
	}

	log.Debug("check for current")
	currentCheck := checkIfPlanIsHardFork(currentPlan, iStatus)
	log.Debug("check for next")
	nextCheck := checkIfPlanIsHardFork(nextPlan, iStatus)
	log.Debug("current and next check", zap.Bools("current, next", []bool{currentCheck, nextCheck}))
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
	log.Debug("hard fork check out", zap.Any("plan", plan))
	return plan, nil

}

// upgrade plan resources should be valid:
//
// id of hardfork upgrade should be "sekai" with valid version "v0.4.1" and chain name suppose to be not the same as current running chain
func checkIfPlanIsHardFork(plan *interx.PlanData, status *interx.Status) bool {
	if plan == nil {
		log.Debug("plan is nil")
		return false
	}
	if !plan.Plan.InstateUpgrade && !plan.Plan.SkipHandler && plan.Plan.RebootRequired {
		if plan.Plan.Resources[0].ID != SEKAI_ID_RESOURCE {
			log.Debug("cant recognize upgrade plan resource ID", zap.String("plan resource id", plan.Plan.Resources[0].ID), zap.String("expected", SEKAI_ID_RESOURCE))
			return false
		}
	}
	_, _, _, err := utils.ParseVersion(plan.Plan.Resources[0].Version)
	if err != nil {
		log.Debug("unable to parse version", zap.String("input version", plan.Plan.Resources[0].Version), zap.Error(err))
		return false
	}

	if status.NodeInfo.Network == plan.Plan.NewChainID {
		log.Debug("Plan has the same chainID as a current network", zap.String("upgrade plan chain ID", plan.Plan.NewChainID), zap.String("current chain ID", status.NodeInfo.Network))
		return false
	}

	return true
}

func ParseUpgradePlan(plan *interx.PlanData) error {

	return nil
}
