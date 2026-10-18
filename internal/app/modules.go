package app

import (
	"bot/internal/general"
	"bot/internal/moderation"
	"bot/internal/module"
)

// buildModules — единственное место, где включаются модули бота.
// Новый модуль: создайте папку internal/<имя>, реализуйте module.Module
// и добавьте строку сюда. Чтобы отключить модуль, удалите его строку.
func buildModules(d module.Deps) []module.Module {
	return []module.Module{
		general.New(d),
		moderation.New(d),
	}
}
