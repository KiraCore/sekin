package commands

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
)

type InitCommand struct {
	commandName string
	Args        map[string]interface{}
}

func NewInitCommand(args map[string]interface{}) *InitCommand {
	return &InitCommand{
		Args:        args,
		commandName: "init",
	}
}

func (c *InitCommand) Execute() error {
	// Implementation of `sekaid init --overwrite ...`
	return postCommand("init", c.Args)
}

func (c *InitCommand) CommandName() string {
	return c.commandName
}

func postCommand(command string, args map[string]interface{}) error {
	slog.Info("POSTing the next command", "command", command, "args", args)
	body, err := json.Marshal(map[string]interface{}{
		"command": command,
		"args":    args,
	})
	if err != nil {
		slog.Error("marshaling the next command", "error", err)
		return err
	}

	// IMPORTANT!

	// TODO change URL based on config
	// TODO change `http.Post` to `http.NewRequestWithContext(ctx, "POST", ...)` if we need
	// TODO change error handling based not only on error type but on HTTP status code

	// IMPORTANT!

	_, err = http.Post("http://sekaid_rpc:8080/api/execute", "application/json", bytes.NewBuffer(body))
	if err != nil {
		slog.Error("POSTing the next command", "error", err)
		return err
	}

	return nil
}
