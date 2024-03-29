package registry

import "context"

type CommandExecutor interface {
	Execute(context.Context) error
}

var executorRegistry = make(map[string]CommandExecutor)

func RegisterExecutor(name string, handler CommandExecutor) {
	executorRegistry[name] = handler
}

func GetCommandExecutor(name string) (CommandExecutor, bool) {
	handler, exists := executorRegistry[name]
	return handler, exists
}
