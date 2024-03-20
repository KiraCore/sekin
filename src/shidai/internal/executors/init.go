package executors

import (
	"log/slog"

	"shidai/internal/commands"
)

type InitExecutor struct {
	commands []commands.Command
}

func NewInitExecutor(args map[string]interface{}) *InitExecutor {
	return &InitExecutor{
		commands: []commands.Command{
			// TODO add connection between `args` to passed the map of args
			commands.NewInitCommand(map[string]interface{}{
				"chain-id":   "testnet-1",
				"moniker":    "KIRA TEST LOCAL VALIDATOR NODE",
				"home":       "/sekai",
				"log_format": "",
				"log_level":  "",
				"trace":      false,
				"overwrite":  true,
			}),
			// TODO add other commands for initialize new sekaid network
			// There is only one
			// sekaid init \
			//     --chain-id=testnet-1 \
			//     --moniker="KIRA TEST LOCAL VALIDATOR NODE" \
			//     --home=/sekai \
			//     --log_format="" \
			//     --log_level="" \
			//     --overwrite
		},
	}
}

func (e *InitExecutor) Execute() error {
	// Implementation of `shidai init ...`
	// Actually there is not CLI commands.
	// It's just a handler for initialization

	// TODO temporary printing logs
	slog.Info("Executing", "command", "init")

	for index, command := range e.commands {
		slog.Info("Running command", "index", index, "command", command.CommandName())

		err := command.Execute()
		if err != nil {
			return err
		}
	}

	return nil
}
