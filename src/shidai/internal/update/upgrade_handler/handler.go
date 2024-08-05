package upgradehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	httpexecutor "github.com/kiracore/sekin/src/shidai/internal/http_executor"

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
	return response, nil
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
	return response, nil
}
