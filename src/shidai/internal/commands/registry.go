package commands

var commandRegistry = make(map[string]CommandHandler)

func RegisterCommand(name string, handler CommandHandler) {
	commandRegistry[name] = handler
}

func GetCommandHandler(name string) (CommandHandler, bool) {
	handler, exists := commandRegistry[name]
	return handler, exists
}

func init() {
	RegisterCommand("join", &JoinCommandHandler{})
}
