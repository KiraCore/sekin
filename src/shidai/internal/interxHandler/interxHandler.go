package interxhandler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	httpexecutor "shidai/utils/httpExecutor"
	joinermanager "shidai/utils/joinerHandler"
	utilsTypes "shidai/utils/types"

	mnemonicsgenerator "github.com/PeepoFrog/validator-key-gen/MnemonicsGenerator"
)

func InitInterx(ctx context.Context, cfg *utilsTypes.ShidaiConfig, masterMnemonicSet *mnemonicsgenerator.MasterMnemonicSet) error {
	nodeId, err := joinermanager.GetLocalSekaidNodeID(cfg.SekaiContainerAddress, cfg.RpcPort)
	if err != nil {
		return err
	}
	signerMnemonic := string(masterMnemonicSet.SignerAddrMnemonic)
	nodeType := "validator"

	grpcSekaidAddress := fmt.Sprintf("dns:///%v:%v", cfg.SekaiContainerAddress, cfg.GrpcPort)
	rpcSekaidAddress := fmt.Sprintf("http://%v:%v", cfg.SekaiContainerAddress, cfg.RpcPort)
	cmd := httpexecutor.CommandRequest{
		Command: "init",
		Args: map[string]interface{}{
			"home":              cfg.InterxHome,
			"grpc":              grpcSekaidAddress,
			"rpc":               rpcSekaidAddress,
			"node_type":         nodeType,
			"faucet_mnemonic":   signerMnemonic,
			"signing_mnemonic":  signerMnemonic,
			"port":              cfg.InterxPort,
			"validator_node_id": nodeId,
		},
	}

	out, err := httpexecutor.ExecuteCallerCommand(cfg.InterxContainerAddress, "8081", "POST", cmd)
	if err != nil {
		return fmt.Errorf("unable execute <%v> request, error: %w", cmd, err)
	}
	log.Println(string(out))

	return nil
}

func StartInterx(cfg *utilsTypes.ShidaiConfig) error {
	cmd := httpexecutor.CommandRequest{
		Command: "start",
		Args: map[string]interface{}{
			"home": cfg.InterxHome,
		},
	}
	_, err := httpexecutor.ExecuteCallerCommand(cfg.InterxContainerAddress, "8081", "POST", cmd)
	if err != nil {
		if errors.Is(err, io.EOF) {
			log.Println("DEBUG: interx started")
		} else {
			return fmt.Errorf("unable execute <%v> request, error: %w", cmd, err)
		}
	}

	return nil
}
