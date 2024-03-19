package mnemonicController

import (
	"fmt"
	"log"
	"os"
	"shidai/utils/osUtils"

	vlg "github.com/PeepoFrog/validator-key-gen/MnemonicsGenerator"
	cosmosBIP39 "github.com/cosmos/go-bip39"

	kiraMnemonicGen "github.com/kiracore/tools/bip39gen/cmd"
	"github.com/kiracore/tools/bip39gen/pkg/bip39"
)

func GenerateMnemonicsFromMaster(masterMnemonic string) (*vlg.MasterMnemonicSet, error) {
	// log.Debugf("GenerateMnemonicFromMaster: masterMnemonic:\n%s", masterMnemonic)
	defaultPrefix := vlg.DefaultPrefix
	defaultPath := vlg.DefaultPath

	mnemonicSet, err := vlg.MasterKeysGen([]byte(masterMnemonic), defaultPrefix, defaultPath, "")
	if err != nil {
		return nil, err
	}
	// str := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n", mnemonicSet.SignerAddrMnemonic, mnemonicSet.ValidatorNodeMnemonic, mnemonicSet.ValidatorNodeId, mnemonicSet.ValidatorAddrMnemonic, mnemonicSet.ValidatorValMnemonic)
	// log.Infof("Master mnemonic:\n%s", str)
	return &mnemonicSet, nil
}

// GenerateMnemonic generates random bip 24 word mnemonic
func GenerateMnemonic() (masterMnemonic bip39.Mnemonic, err error) {
	log.Println("generating new mnemonic")
	masterMnemonic = kiraMnemonicGen.NewMnemonic()
	masterMnemonic.SetRandomEntropy(24)
	masterMnemonic.Generate()

	return masterMnemonic, nil
}

func ValidateMnemonic(mnemonic string) error {
	check := cosmosBIP39.IsMnemonicValid(mnemonic)
	if !check {
		return fmt.Errorf("mnemonic <%v> is not valid", mnemonic)
	}
	return nil
}

func SetSekaidPrivKeys(mnemonicSet *vlg.MasterMnemonicSet, sekaidHome string) error {
	// TODO path set as variables or constants
	sekaidConfigFolder := sekaidHome + "/config"
	fmt.Println(sekaidConfigFolder)

	//creating sekaid home
	err := os.Mkdir(sekaidHome, 0755)
	if err != nil {
		if !os.IsExist(err) {
			return fmt.Errorf("unable to create <%s> folder, err: %w", sekaidHome, err)
		}
	}
	//creating sekaid's config folder
	err = os.Mkdir(sekaidConfigFolder, 0755)
	if err != nil {
		if !os.IsExist(err) {
			return fmt.Errorf("unable to create <%s> folder, err: %w", sekaidConfigFolder, err)
		}
	}

	err = vlg.GeneratePrivValidatorKeyJson(mnemonicSet.ValidatorValMnemonic, sekaidConfigFolder+"/priv_validator_key.json", vlg.DefaultPrefix, vlg.DefaultPath)
	if err != nil {
		return fmt.Errorf("unable to generate priv_validator_key.json: %w", err)
	}
	err = vlg.GenerateValidatorNodeKeyJson(mnemonicSet.ValidatorNodeMnemonic, sekaidConfigFolder+"/node_key.json", vlg.DefaultPrefix, vlg.DefaultPath)
	if err != nil {
		return fmt.Errorf("unable to generate node_key.json: %w", err)
	}
	return nil
}

// sets empty state of validator into $sekaidHome/data/priv_validator_state.json
func SetEmptyValidatorState(sekaidHome string) error {
	emptyState := `
	{
		"height": "0",
		"round": 0,
		"step": 0
	}`
	sekaidDataFolder := sekaidHome + "/data"
	err := os.Mkdir(sekaidDataFolder, 0755)
	if err != nil {
		if !os.IsExist(err) {
			return fmt.Errorf("unable to create <%s> folder, err: %w", sekaidDataFolder, err)
		}
	}
	osUtils.CreateFileWithData(sekaidDataFolder+"/priv_validator_state.json", []byte(emptyState))
	fmt.Println(emptyState, sekaidDataFolder)
	return nil
}
