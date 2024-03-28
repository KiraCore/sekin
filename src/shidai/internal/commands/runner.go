package commands

import "context"

type CommandRunner struct {
	commandName string
	Args        map[string]interface{}
}

func NewCommandRunner(cmd string, args map[string]interface{}) *CommandRunner {
	return &CommandRunner{
		Args:        args,
		commandName: cmd,
	}
}

func (c *CommandRunner) Execute(ctx context.Context) error {
	_, err := postCommand(ctx, c.commandName, c.Args)

	return err
}

func (c *CommandRunner) Name() string {
	return c.commandName
}
