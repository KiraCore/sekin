package sekaihandler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	mnemonicsgenerator "github.com/KiraCore/tools/validator-key-gen/MnemonicsGenerator"
	httpexecutor "github.com/kiracore/sekin/src/shidai/internal/http_executor"
	"github.com/kiracore/sekin/src/shidai/internal/logger"
	mnemonicmanager "github.com/kiracore/sekin/src/shidai/internal/mnemonic_manager"
	"go.uber.org/zap"

	configconstructor "github.com/kiracore/sekin/src/shidai/internal/sekai_handler/config_constructor"
	genesishandler "github.com/kiracore/sekin/src/shidai/internal/sekai_handler/genesis_handler"
	"github.com/kiracore/sekin/src/shidai/internal/types"
	"github.com/kiracore/sekin/src/shidai/internal/utils"
)

var log = logger.GetLogger()

func InitSekaiJoiner(ctx context.Context, tc *configconstructor.TargetSeedKiraConfig, masterMnemonicSet *mnemonicsgenerator.MasterMnemonicSet) error {
	log.Debug("Initializing Sekai Joiner", zap.String("home", types.SEKAI_HOME), zap.String("chain-id", "initnet-1"))

	cmd := httpexecutor.CommandRequest{
		Command: "init",
		Args: map[string]interface{}{
			"home":     types.SEKAI_HOME,
			"chain-id": "initnet-1",
			"moniker":  "validator node",
		},
	}
	_, err := httpexecutor.ExecuteCallerCommand("sekai.local", "8080", "POST", cmd)
	if err != nil {
		log.Error("Failed to execute caller command", zap.Any("command", cmd), zap.Error(err))
		return fmt.Errorf("unable execute <%v> request, error: %w", cmd, err)
	}
	log.Debug("Caller command executed successfully")

	err = setSekaidKeys(masterMnemonicSet)
	if err != nil {
		log.Error("Failed to set Sekai keys", zap.Error(err))
		return fmt.Errorf("unable to set sekai keys: %w", err)
	}
	log.Debug("Sekai keys set successfully")

	genesis, err := genesishandler.GetVerifiedGenesisFile(ctx, tc.IpAddress, tc.SekaidRPCPort, tc.InterxPort)
	if err != nil {
		log.Error("Failed to receive verified genesis file", zap.String("IP", tc.IpAddress), zap.Error(err))
		return fmt.Errorf("unable to receive genesis file: %w", err)
	}
	log.Debug("Genesis file received and verified")

	err = os.WriteFile(fmt.Sprintf("%v/config/genesis.json", types.SEKAI_HOME), genesis, 0644)
	if err != nil {
		log.Error("Failed to write genesis.json file", zap.String("file_path", fmt.Sprintf("%v/config/genesis.json", types.SEKAI_HOME)), zap.Error(err))
		return fmt.Errorf("cant write genesis.json file: %w", err)
	}
	log.Debug("Genesis.json file written successfully")

	err = configconstructor.FormSekaiJoinerConfigs(tc)
	if err != nil {
		log.Error("Failed to form Sekai joiner configurations", zap.String("IP", tc.IpAddress), zap.Error(err))
		return fmt.Errorf("unable retrieve join information from <%s>, error: %w", tc.IpAddress, err)
	}
	log.Debug("Sekai joiner configurations formed successfully")

	return nil
}

// TODO:
func InitSekaiNew(ctx context.Context, masterMnemonicSet *mnemonicsgenerator.MasterMnemonicSet) error {
	return nil
}

func setSekaidKeys(masterMnemonicSet *mnemonicsgenerator.MasterMnemonicSet) error {
	log.Debug("Setting Sekaid keys", zap.String("home", types.SEKAI_HOME))

	err := mnemonicmanager.SetSekaidPrivKeys(masterMnemonicSet, types.SEKAI_HOME)
	if err != nil {
		log.Error("Failed to set Sekaid private keys", zap.Error(err))
		return fmt.Errorf("unable to set sekaid keys: %w", err)
	}
	log.Debug("Sekaid private keys set successfully")

	err = mnemonicmanager.SetEmptyValidatorState(types.SEKAI_HOME)
	if err != nil {
		log.Error("Failed to set empty validator state", zap.Error(err))
		return fmt.Errorf("unable to set empty validator state: %w", err)
	}
	log.Debug("Empty validator state set successfully")

	_, err = utils.AddKeyToKeyring("validator", string(masterMnemonicSet.ValidatorAddrMnemonic), types.SEKAI_HOME, "test")
	if err != nil {
		log.Error("Failed to add validator key to keyring", zap.Error(err))
		return fmt.Errorf("unable to add validator key to keyring: %w", err)
	}
	log.Debug("Validator key added to keyring successfully")

	return nil
}

func StartSekai() error {
	log.Debug("Starting Sekai", zap.String("home", types.SEKAI_HOME))

	cmd := httpexecutor.CommandRequest{
		Command: "start",
		Args: map[string]interface{}{
			"home": types.SEKAI_HOME,
		},
	}
	_, err := httpexecutor.ExecuteCallerCommand(types.SEKAI_CONTAINER_ADDRESS, "8080", "POST", cmd)
	if err != nil {
		log.Error("Failed to execute start command", zap.Any("command", cmd), zap.Error(err))
		if errors.Is(err, io.EOF) {
			log.Debug("Sekai started despite EOF error", zap.Error(err))
		} else {
			return fmt.Errorf("unable execute <%v> request, error: %w", cmd, err)
		}
	}
	log.Debug("Sekai start command executed successfully")

	return nil
}
