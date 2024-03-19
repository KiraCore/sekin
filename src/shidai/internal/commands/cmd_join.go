package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	interxhandler "shidai/internal/interxHandler"
	sekaihandler "shidai/internal/sekaiHandler"

	// "shidai/utils/nodeInit"
	utilsTypes "shidai/utils/types"

	// "shidai/utils/config"

	joinermanager "shidai/utils/joinerHandler"
	"shidai/utils/mnemonicController"
	"shidai/utils/osUtils"
	"strconv"
	"time"
)

// Example of json body request
//
//	{
//	    "command": "join",
//	    "args": {
//	        "ip": "10.43.239.82",
//	        "interxPort": 11000,
//	        "rpcPort": 26657,
//	        "p2pPort": 26656,
//	        "sekaidAddress": "sekai.local",
//	        "interxAddress":"interx.local",
//	        "mnemonic": "bargain erosion electric skill extend aunt unfold cricket spice sudden insane shock purpose trumpet holiday tornado fiction check pony acoustic strike side gold resemble"
//	    }
//	}
type JoinCommandHandler struct {
	IPToJoin               string `json:"ip"`            // ip to join
	InterxPort             int    `json:"interxPort"`    // interx port of the node you joining to (default 11000)
	RpcPortToJoin          int    `json:"rpcPort"`       // sekaid's rpc port of the node you joining to (default 26657)
	P2PPortToJoin          int    `json:"p2pPort"`       // sekaid's grpc port of the node you joining to (default 26656)
	Mnemonic               string `json:"mnemonic"`      //
	SekaiContainerAddress  string `json:"sekaiAddress"`  // hostname value from docker compose
	InterxContainerAddress string `json:"interxAddress"` // hostname value from docker compose
	EnableInterx           bool   `json:"enableInterx"`  // if true initializing interx and starts it, else ignores it
}

func (j *JoinCommandHandler) HandleCommand(args map[string]interface{}) error {
	jsonData, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("error marshalling map to JSON: %w", err)
	}
	// var handler JoinCommandHandler
	err = json.Unmarshal(jsonData, j)
	if err != nil {
		return fmt.Errorf("error unmarshalling JSON to struct: %w", err)
	}
	err = j.ValidateJoinCommand()
	if err != nil {
		return fmt.Errorf("error validating command arguments: %w", err)
	}
	defaultRyokaiConfig := utilsTypes.DefaultShidaiConfig()
	defaultRyokaiConfig.SekaiContainerAddress = j.SekaiContainerAddress
	defaultRyokaiConfig.InterxContainerAddress = j.InterxContainerAddress
	err = j.InitJoinerNode(defaultRyokaiConfig)
	if err != nil {
		return fmt.Errorf("error when joining: %w", err)
	}
	return nil
}

func (j *JoinCommandHandler) ValidateJoinCommand() error {
	check := osUtils.ValidateIP(j.IPToJoin)
	if !check {
		return fmt.Errorf("<%v> in not a valid ip", j.IPToJoin)
	}
	check = osUtils.ValidatePort(strconv.Itoa(j.P2PPortToJoin))
	if !check {
		return fmt.Errorf("<%v> in not a valid port", j.P2PPortToJoin)
	}

	check = osUtils.ValidatePort(strconv.Itoa(j.RpcPortToJoin))
	if !check {
		return fmt.Errorf("<%v> in not a valid port", j.RpcPortToJoin)
	}

	check = osUtils.ValidatePort(strconv.Itoa(j.InterxPort))
	if !check {
		return fmt.Errorf("<%v> in not a valid port", j.InterxPort)
	}

	return nil
}

// func (j *JoinCommandHandler) InitJoinerNode(sekaidHome, interxdHome string) error {
func (j *JoinCommandHandler) InitJoinerNode(cfg *utilsTypes.ShidaiConfig) error {
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Minute*5)

	defer cancelFunc()

	err := j.cleanUpSekaidAndInterxHome(cfg.SekaidHome, cfg.InterxHome)
	if err != nil {
		return fmt.Errorf("unable to clean up sekai and interx homes: %w", err)
	}

	var masterMnemonic string

	tc := joinermanager.TargetSeedKiraConfig{
		IpAddress:     j.IPToJoin,
		InterxPort:    strconv.Itoa(j.InterxPort),
		SekaidRPCPort: strconv.Itoa(j.RpcPortToJoin),
		SekaidP2PPort: strconv.Itoa(j.P2PPortToJoin),
	}
	// TODO: should we generate mnemonic or force user to set Mnemonic?
	// Generate masterMnemonic if current mnemonic is empty
	if j.Mnemonic == "" {
		bip39m, err := mnemonicController.GenerateMnemonic()
		if err != nil {
			return fmt.Errorf("unable to generate masterMnemonic: %w", err)
		}
		masterMnemonic = bip39m.String()
	} else {
		err := mnemonicController.ValidateMnemonic(j.Mnemonic)
		if err != nil {
			return fmt.Errorf("unable to validate mnemonic: %w", err)
		}
		masterMnemonic = j.Mnemonic
	}

	// //Generate master mnemonic set
	masterMnemonicsSet, err := mnemonicController.GenerateMnemonicsFromMaster(masterMnemonic)
	if err != nil {
		return fmt.Errorf("unable to generate master mnemonic set: %w", err)
	}

	err = sekaihandler.InitSekaiJoiner(ctx, &tc, cfg, masterMnemonicsSet)
	if err != nil {
		return fmt.Errorf("unable to init sekai: %w", err)
	}
	err = sekaihandler.StartSekai(cfg)
	if err != nil {
		return fmt.Errorf("unable to start sekai: %w", err)
	}

	// Important! Need small delay between sekaid and interx start
	// (when generating new network needed to wait for first block to be produced)
	// Change this to sekaid health check, if blocks are producing you can init interx
	time.Sleep(time.Second * 5)
	err = interxhandler.InitInterx(ctx, cfg, masterMnemonicsSet)
	if err != nil {
		return fmt.Errorf("unable to init interx: %w", err)
	}
	if j.EnableInterx {
		err = interxhandler.StartInterx(cfg)
		if err != nil {
			return fmt.Errorf("unable to start interx: %w", err)
		}
	}

	return nil
}

// func cleanup and create folder for shidai
func (j *JoinCommandHandler) cleanUpSekaidAndInterxHome(sekaidHome, interxdHome string) error {
	// TODO: shutdown sekaid and interx docker container
	check := osUtils.FileExist(sekaidHome)
	if check {
		err := os.RemoveAll(sekaidHome)
		if err != nil {
			return err
		}
	}

	check = osUtils.FileExist(interxdHome)
	if check {
		err := os.RemoveAll(interxdHome)
		if err != nil {
			return err
		}
	}

	return nil
}
