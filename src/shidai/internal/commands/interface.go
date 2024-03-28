package commands

import "context"

type Command interface {
	Name() string
	Execute(context.Context) error
}
