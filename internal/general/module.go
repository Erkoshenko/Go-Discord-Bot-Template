// Package general — базовые команды, не относящиеся к отдельной функции бота.
package general

import (
	"bot/internal/commands"
	"bot/internal/module"
)

type Module struct {
	module.Base
}

func New(module.Deps) *Module { return &Module{} }

func (m *Module) Name() string { return "general" }

func (m *Module) Commands() []commands.Command {
	return []commands.Command{&pingCommand{}}
}
