package commands

type Command interface {
	CommandName() string
	Execute() error
}
