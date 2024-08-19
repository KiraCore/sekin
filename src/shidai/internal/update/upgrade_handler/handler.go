package upgradehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	httpexecutor "github.com/kiracore/sekin/src/shidai/internal/http_executor"
	"github.com/kiracore/sekin/src/shidai/internal/types"

	"github.com/kiracore/sekin/src/shidai/internal/logger"
	"github.com/kiracore/sekin/src/shidai/internal/types/endpoints/interx"
	"go.uber.org/zap"
)

var (
	log = logger.GetLogger()
)

func CheckNextUpgradePlan(ctx context.Context, ipAddress, interxPort string) (*interx.PlanData, error) {
	url := fmt.Sprintf("http://%s:%s/%s", ipAddress, interxPort, interx.ENDPOINT_NEXT_PLAN)
	client := &http.Client{}
	log.Debug("Querying sekai status by url:", zap.String("url", url))

	body, err := httpexecutor.DoHttpQuery(ctx, client, url, "GET")
	if err != nil {
		return nil, err
	}
	var response *interx.PlanData
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}
	if !checkIfPlanIsNull(response) {
		return response, nil
	} else {
		return nil, nil
	}
}
func CheckCurrentUpgradePlan(ctx context.Context, ipAddress, interxPort string) (*interx.PlanData, error) {
	url := fmt.Sprintf("http://%s:%s/%s", ipAddress, interxPort, interx.ENDPOINT_CURRENT_PLAN)
	client := &http.Client{}
	log.Debug("Querying sekai status by url:", zap.String("url", url))

	body, err := httpexecutor.DoHttpQuery(ctx, client, url, "GET")
	if err != nil {
		return nil, err
	}
	var response *interx.PlanData
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}
	if !checkIfPlanIsNull(response) {
		return response, nil
	} else {
		return nil, nil
	}
}

func checkIfPlanIsNull(plan *interx.PlanData) bool {
	if plan.Plan.ProposalID == "" {
		return true
	} else {
		return false
	}
}

func GetCurrentVersions() (*types.SekinPackagesVersion, error) {
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
