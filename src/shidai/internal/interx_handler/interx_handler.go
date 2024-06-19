package interxhandler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	mnemonicsgenerator "github.com/KiraCore/tools/validator-key-gen/MnemonicsGenerator"
	"go.uber.org/zap"

	httpexecutor "github.com/kiracore/sekin/src/shidai/internal/http_executor"
	"github.com/kiracore/sekin/src/shidai/internal/logger"
	"github.com/kiracore/sekin/src/shidai/internal/types"
	"github.com/kiracore/sekin/src/shidai/internal/utils"
)

var log = logger.GetLogger()

func InitInterx(ctx context.Context, masterMnemonicSet *mnemonicsgenerator.MasterMnemonicSet) error {
	signerMnemonic := string(masterMnemonicSet.SignerAddrMnemonic)
	nodeType := "validator"

	grpcSekaidAddress := fmt.Sprintf("dns:///%v:%v", types.SEKAI_CONTAINER_ADDRESS, types.DEFAULT_GRPC_PORT)
	rpcSekaidAddress := fmt.Sprintf("http://%v:%v", types.SEKAI_CONTAINER_ADDRESS, types.DEFAULT_RPC_PORT)
	cmd := httpexecutor.CommandRequest{
		Command: "init",
		Args: map[string]interface{}{
			"home":              types.INTERX_HOME,
			"grpc":              grpcSekaidAddress,
			"rpc":               rpcSekaidAddress,
			"node_type":         nodeType,
			"faucet_mnemonic":   signerMnemonic,
			"signing_mnemonic":  signerMnemonic,
			"port":              fmt.Sprintf("%v", types.DEFAULT_INTERX_PORT),
			"validator_node_id": string(masterMnemonicSet.ValidatorNodeId),
		},
	}
	out, err := httpexecutor.ExecuteCallerCommand(types.INTERX_CONTAINER_ADDRESS, "8081", "POST", cmd)
	if err != nil {
		return fmt.Errorf("unable execute <%v> request, error: %w", cmd, err)
	}
	log.Info(string(out))

	return nil
}

func StartInterx(ctx context.Context) error {
	cmd := httpexecutor.CommandRequest{
		Command: "start",
		Args: map[string]interface{}{
			"home": types.INTERX_HOME,
		},
	}
	_, err := httpexecutor.ExecuteCallerCommand(types.INTERX_CONTAINER_ADDRESS, "8081", "POST", cmd)
	if err != nil {
		if errors.Is(err, io.EOF) {
			log.Debug("interx started")
		} else {
			return fmt.Errorf("unable execute <%v> request, error: %w", cmd, err)
		}
	}
	go addrbookManager(ctx)
	return nil
}

// run this in goroutine
func addrbookManager(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()
	errorCooldown := time.Second * 1

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Stopping the addrbookManager loop")
			return
		case t := <-ticker.C:
			for {
				err := addrbookCopy()
				if err != nil {
					log.Debug("Error when replacing interx addrbook with sekai addrbook, sleeping", zap.Duration("errorCooldown", errorCooldown))
					time.Sleep(errorCooldown)
					continue
				}
				log.Debug("Address book copying was executed", zap.Time("ticker", t))
				break
			}
		}
	}
}

func addrbookCopy() error {
	var equal, interxAddrbookExist bool
	var err error
	interxAddrbookExist = utils.FileExists(types.INTERX_ADDRBOOK_PATH)
	if interxAddrbookExist {
		equal, err = utils.FilesAreEqual(types.SEKAI_ADDRBOOK_PATH, types.INTERX_ADDRBOOK_PATH)
		if err != nil {
			return fmt.Errorf("error when comparing sekai and interx address books: %w", err)
		}
	}
	if !equal || !interxAddrbookExist {
		err := utils.SafeCopy(types.SEKAI_ADDRBOOK_PATH, types.INTERX_ADDRBOOK_PATH)
		if err != nil {
			return fmt.Errorf("error when replacing interx addrbook with sekai addrbook: %w", err)
		}
	}

	return nil
}
