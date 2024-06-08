package commands

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"

	interxhandler "github.com/kiracore/sekin/src/shidai/internal/interx_handler"
	interxhelper "github.com/kiracore/sekin/src/shidai/internal/interx_handler/interx_helper"
	"github.com/kiracore/sekin/src/shidai/internal/logger"
	mnemonicmanager "github.com/kiracore/sekin/src/shidai/internal/mnemonic_manager"
	sekaihandler "github.com/kiracore/sekin/src/shidai/internal/sekai_handler"
	configconstructor "github.com/kiracore/sekin/src/shidai/internal/sekai_handler/config_constructor"
	sekaihelper "github.com/kiracore/sekin/src/shidai/internal/sekai_handler/sekai_helper"
	"github.com/kiracore/sekin/src/shidai/internal/types"
	"github.com/kiracore/sekin/src/shidai/internal/utils"
	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
)

// CommandRequest defines the structure for incoming command requests
type CommandRequest struct {
	Command string                 `json:"command"`
	Args    map[string]interface{} `json:"args"`
}

// CommandResponse represents the response structure
type CommandResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// HandlerFunc is a function type for command handlers
type HandlerFunc func(map[string]interface{}) (string, error)

// CommandHandlers maps command strings to handler functions
var (
	log             *zap.Logger = logger.GetLogger()
	CommandHandlers             = map[string]HandlerFunc{
		"join":   handleJoinCommand,
		"status": handleStatusCommand,
		"start":  handleStartComamnd,
		"tx":     handleTxCommand,
	}
)

// ExecuteCommandHandler handles incoming commands and directs them to the correct function
func ExecuteCommandHandler(c *gin.Context) {
	var req CommandRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, CommandResponse{Status: "error", Message: "Invalid request"})
		return
	}

	if handler, ok := CommandHandlers[req.Command]; ok {
		response, err := handler(req.Args)
		if err != nil {
			c.JSON(http.StatusInternalServerError, CommandResponse{Status: "error", Message: err.Error()})
			return
		}
		c.JSON(http.StatusOK, CommandResponse{Status: "success", Message: response})
		return
	}

	c.JSON(http.StatusBadRequest, CommandResponse{Status: "error", Message: fmt.Sprintf("Unknown command: %s", req.Command)})
}

// [COMMANDS] //

func handleTxCommand(args map[string]interface{}) (string, error) {
	cmd, ok := args["tx"].(string)
	if !ok {
		return "", types, ErrInvalidOrMissingTx
	}
	return "", nil
}

// handleJoinCommand processes the "join" command
func handleJoinCommand(args map[string]interface{}) (string, error) {
	// Unmarshal arguments to a specific struct if needed or handle them as a map
	ip, ok := args["ip"].(string)
	if !utils.ValidateIP(ip) || !ok {
		return "", types.ErrInvalidOrMissingIP
	}

	m, ok := args["mnemonic"].(string)
	if !utils.ValidateMnemonic(m) || !ok {
		return "", types.ErrInvalidOrMissingMnemonic
	}
	pathsToDel := []string{"/sekai/", "/interx/"}
	for _, path := range pathsToDel {
		err := os.RemoveAll(path)
		if err != nil {
			log.Error("Failed to delele ", zap.String("path", path), zap.Error(err))
		}
	}

	masterMnemonic, err := mnemonicmanager.GenerateMnemonicsFromMaster(m)
	if err != nil {
		return "", err
	}

	ctx := context.Background()

	p2p, ok := args["p2p_port"].(float64)
	if !utils.ValidatePort(int(p2p)) || !ok {
		return "", types.ErrInvalidOrMissingP2PPort
	}
	rpc, ok := args["rpc_port"].(float64)
	if !utils.ValidatePort(int(rpc)) || !ok {
		return "", types.ErrInvalidOrMissingRPCPort
	}
	interx, ok := args["interx_port"].(float64)
	if !utils.ValidatePort(int(interx)) || !ok {
		return "", types.ErrInvalidOrMissingInterxPort
	}

	tc := configconstructor.TargetSeedKiraConfig{IpAddress: ip, InterxPort: strconv.Itoa(int(interx)), SekaidRPCPort: strconv.Itoa(int(rpc)), SekaidP2PPort: strconv.Itoa(int(p2p))}
	err = sekaihandler.InitSekaiJoiner(ctx, &tc, masterMnemonic)
	if err != nil {
		return "", err
	}
	err = sekaihandler.StartSekai()
	if err != nil {
		return "", fmt.Errorf("unable to start sekai: %w", err)
	}
	err = sekaihelper.CheckSekaiStart(ctx)
	if err != nil {
		return "", err
	}
	err = interxhandler.InitInterx(ctx, masterMnemonic)
	if err != nil {
		return "", fmt.Errorf("unable to init interx: %w", err)
	}
	err = interxhandler.StartInterx()
	if err != nil {
		return "", fmt.Errorf("unable to start interx: %w", err)
	}
	err = interxhelper.CheckInterxStart(ctx)
	if err != nil {
		return "", err
	}
	// Example of using the IP, and similar for other fields
	// This function would contain the logic specific to handling a join command
	return fmt.Sprintf("Join command processed for IP: %s", ip), nil
}

func handleStatusCommand(args map[string]interface{}) (string, error) {
	// TODO:
	// 1. Return publicIP
	// 2. Return validatorAddress
	// 3. Return validatorStatus
	// 4. Return missChance
	// 5.

	return "", nil
}

func handleStartComamnd(args map[string]interface{}) (string, error) {
	err := sekaihandler.StartSekai()
	if err != nil {
		return "", fmt.Errorf("unable to start sekai: %w", err)
	}
	ctx := context.Background()

	err = sekaihelper.CheckSekaiStart(ctx)
	if err != nil {
		return "", err
	}

	err = interxhandler.StartInterx()
	if err != nil {
		return "", fmt.Errorf("unable to start interx: %w", err)
	}
	err = interxhelper.CheckInterxStart(ctx)
	if err != nil {
		return "", err
	}
	return "Sekai and Interx started successfully", nil
}
