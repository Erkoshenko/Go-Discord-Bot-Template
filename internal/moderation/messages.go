package moderation

import "fmt"

// Тексты, которые модуль отправляет пользователю и в журнал.

func dmWarn(reason string) string {
	return fmt.Sprintf("⚠️ Вы получили предупреждение на сервере.\nПричина: %s", reason)
}

func dmTimeout(untilUnix int64, reason string) string {
	return fmt.Sprintf("🔇 Вы получили мут на сервере до <t:%d:f>.\nПричина: %s", untilUnix, reason)
}

func dmKick(reason string) string {
	return fmt.Sprintf("👢 Вы были выгнаны с сервера.\nПричина: %s", reason)
}

func dmBan(reason string) string {
	return fmt.Sprintf("🔨 Вы были забанены на сервере.\nПричина: %s", reason)
}

func logCase(c Case, title, extra string) string {
	return fmt.Sprintf("📋 **Дело #%d · %s**\nПользователь: <@%s>\nМодератор: <@%s>\n%sПричина: %s",
		c.ID, title, c.UserID, c.ModeratorID, extra, c.Reason)
}
