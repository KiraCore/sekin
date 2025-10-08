package scaller

import (
	"fmt"

	httpexecutor "github.com/kiracore/sekin/src/shidai/internal/http_executor"
	"github.com/kiracore/sekin/src/shidai/internal/logger"
	"github.com/kiracore/sekin/src/shidai/internal/types"
	"go.uber.org/zap"
)

var log = logger.GetLogger()

func InitCmd(networkName, moniker string) error {
	cmd := httpexecutor.CommandRequest{
		Command: "init",
		Args: map[string]interface{}{
			"home":     types.SEKAI_HOME,
			"chain-id": networkName,
			"moniker":  moniker,
		},
	}
	_, err := httpexecutor.ExecuteCallerCommand(types.SEKAI_CONTAINER_ADDRESS, "8080", "POST", cmd)
	if err != nil {
		log.Error("Failed to execute caller command", zap.Any("command", cmd), zap.Error(err))
		return fmt.Errorf("unable execute <%v> request, error: %w", cmd, err)
	}
	return nil
}

func AddGenesisAccount(networkName, moniker, accountName string, coins []string) error {
	//  "address": "genesis",
	// "coins": ["300000000000000ukex"],
	// "keyring-backend": "test",
	// "home": "/sekai",
	// "log_format": "",
	// "log_level": "",
	// "trace": false
	cmd := httpexecutor.CommandRequest{
		Command: "add-genesis-account",
		Args: map[string]interface{}{
			"home":            types.SEKAI_HOME,
			"chain-id":        networkName,
			"moniker":         moniker,
			"address":         accountName,
			"keyring-backend": types.DefaultKeyring,
			"coins":           coins,
		},
	}
	_, err := httpexecutor.ExecuteCallerCommand(types.SEKAI_CONTAINER_ADDRESS, "8080", "POST", cmd)
	if err != nil {
		log.Error("Failed to execute caller command", zap.Any("command", cmd), zap.Error(err))
		return fmt.Errorf("unable execute <%v> request, error: %w", cmd, err)
	}
	return nil
}

func GentxClaimCmd(networkName, moniker, accountName string) error {
	cmd := httpexecutor.CommandRequest{
		Command: "gentx-claim",
		Args: map[string]interface{}{
			"home":            types.SEKAI_HOME,
			"chain-id":        networkName,
			"moniker":         moniker,
			"address":         accountName,
			"keyring-backend": types.DefaultKeyring,
		},
	}
	_, err := httpexecutor.ExecuteCallerCommand(types.SEKAI_CONTAINER_ADDRESS, "8080", "POST", cmd)
	if err != nil {
		log.Error("Failed to execute caller command", zap.Any("command", cmd), zap.Error(err))
		return fmt.Errorf("unable execute <%v> request, error: %w", cmd, err)
	}
	return nil
}
