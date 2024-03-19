package sekaihandler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"shidai/utils/cosmosHelper"

	httpexecutor "shidai/utils/httpExecutor"
	joinermanager "shidai/utils/joinerHandler"
	"shidai/utils/mnemonicController"
	utilsTypes "shidai/utils/types"

	mnemonicsgenerator "github.com/PeepoFrog/validator-key-gen/MnemonicsGenerator"
)

func StartSekai(cfg *utilsTypes.ShidaiConfig) error {
	cmd := httpexecutor.CommandRequest{
		Command: "start",
		Args: map[string]interface{}{
			"home": cfg.SekaidHome,
		},
	}
	_, err := httpexecutor.ExecuteCallerCommand(cfg.SekaiContainerAddress, "8080", "POST", cmd)
	if err != nil {
		if errors.Is(err, io.EOF) {
			log.Println("DEBUG: sekai started")
		} else {
			return fmt.Errorf("unable execute <%v> request, error: %w", cmd, err)
		}
	}

	return nil
}

func InitSekaiJoiner(ctx context.Context, tc *joinermanager.TargetSeedKiraConfig, cfg *utilsTypes.ShidaiConfig, masterMnemonicSet *mnemonicsgenerator.MasterMnemonicSet) error {
	cmd := httpexecutor.CommandRequest{
		Command: "init",
		Args: map[string]interface{}{
			"home":     cfg.SekaidHome,
			"chain-id": "initnet-1",
			"moniker":  "validator node",
		},
	}
	_, err := httpexecutor.ExecuteCallerCommand(cfg.SekaiContainerAddress, "8080", "POST", cmd)
	if err != nil {
		return fmt.Errorf("unable execute <%v> request, error: %w", cmd, err)
	}
	err = setSekaidKeys(cfg, masterMnemonicSet)
	if err != nil {
		return fmt.Errorf("unable to set sekai keys: %w", err)
	}
	genesis, err := joinermanager.GetVerifiedGenesisFile(ctx, tc.IpAddress, tc.SekaidRPCPort, tc.InterxPort)
	if err != nil {
		return fmt.Errorf("unable to receive genesis file: %w", err)
	}
	err = os.WriteFile(fmt.Sprintf("%v/config/genesis.json", cfg.SekaidHome), genesis, 0644)
	if err != nil {
		return fmt.Errorf("cant write genesis.json file: %w", err)
	}

	err = joinermanager.ApplyJoinerTomlSettings(cfg.SekaidHome, tc, cfg)
	if err != nil {
		return fmt.Errorf("unable retrieve join information from <%s>, error: %w", tc.IpAddress, err)
	}

	return nil
}

func setSekaidKeys(cfg *utilsTypes.ShidaiConfig, masterMnemonicSet *mnemonicsgenerator.MasterMnemonicSet) error {

	err := mnemonicController.SetSekaidPrivKeys(masterMnemonicSet, cfg.SekaidHome)
	if err != nil {
		return fmt.Errorf("unable to set sekaid keys: %w", err)
	}
	err = mnemonicController.SetEmptyValidatorState(cfg.SekaidHome)
	if err != nil {
		return fmt.Errorf("unable to set empty validator state : %w", err)
	}

	_, err = cosmosHelper.AddKeyToKeyring("validator", string(masterMnemonicSet.ValidatorAddrMnemonic), cfg.SekaidHome, "test")
	if err != nil {
		return fmt.Errorf("unable to add validator key to keyring: %w", err)
	}
	return nil
}
