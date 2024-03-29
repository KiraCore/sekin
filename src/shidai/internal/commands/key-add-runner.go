package commands

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"

	"shidai/internal/config"
)

const cmd = "keys-add"

type KeyAddRunner struct {
	commandName string
	Args        map[string]interface{}
}

var (
	mnemonicRegularExpression = regexp.MustCompile(`\\n\\n([a-z ]+)`)
	addressRegularExpression  = regexp.MustCompile(`-\saddress:\s([a-z0-9]+)`)
)

func NewKeyAddRunner(args map[string]interface{}) *KeyAddRunner {
	return &KeyAddRunner{
		commandName: cmd,
		Args:        args,
	}
}

// TODO look up to Dmitro's solution for adding keys
func (c *KeyAddRunner) Execute(ctx context.Context) error {
	cfg, ok := getConfig(ctx)
	if !ok {
		return fmt.Errorf("no config provided")
	}

	slog.Info("Config", "config", cfg)

	c.Args["home"] = cfg.Sekaid.Home

	bodyString, err := postCommand(ctx, c.commandName, c.Args)
	if err != nil {
		return fmt.Errorf("error posting command: %w", err)
	}

	err = mnemonicTODO(ctx, bodyString)
	if err != nil {
		return fmt.Errorf("error writing mnemonic to file: %w", err)
	}

	return nil
}

func (c *KeyAddRunner) Name() string {
	return c.commandName
}

// getConfig extracts the configuration from the given context.
func getConfig(ctx context.Context) (config.Config, bool) {
	cfg, ok := ctx.Value(config.ConfigContextKey).(config.Config)
	return cfg, ok
}

// TODO mnemonicTODO: write to file?
// TODO there is no business logic
// TODO this method outputs the MNEMONIC to the STDOUT
func mnemonicTODO(_ context.Context, output []byte) error {
	slog.Debug("Got", "output", string(output))

	match := mnemonicRegularExpression.FindStringSubmatch(string(output))
	if match == nil || len(match) < 2 {
		return fmt.Errorf("not found mnemonic")
	}

	slog.Info("Last pack of 24 words", "words", match[1])

	match = addressRegularExpression.FindStringSubmatch(string(output))
	if match == nil || len(match) < 2 {
		return fmt.Errorf("not found address")
	}

	slog.Info("Address for new key", "address", match[1])

	return nil
}
