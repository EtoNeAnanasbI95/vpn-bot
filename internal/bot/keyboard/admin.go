package keyboard

import (
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/callback"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/domain"
)

// AdminPanel is the main admin panel inline keyboard.
func AdminPanel() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📢 Рассылка", callback.Encode(callback.ActionAdmBcastAll)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💳 Оплаты", callback.ActionAdmPayList),
			tgbotapi.NewInlineKeyboardButtonData("👥 Клиенты", callback.AdmUserList(0)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔗 Подключения", callback.ActionAdmConnUsers),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💚 Друзья", callback.ActionAdmFreeFriendList),
			tgbotapi.NewInlineKeyboardButtonData("📅 Даты оплат", callback.ActionAdmPayDateList),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💰 Мои реквизиты", callback.ActionAdmSetPayInfo),
		),
	)
}

// BroadcastMenu shows broadcast options.
func BroadcastMenu() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📢 Всем клиентам", callback.ActionAdmBcastAll),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👤 Конкретному клиенту", callback.Encode(callback.ActionAdmBcastUser)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("← Назад", callback.ActionAdmMenu),
		),
	)
}

// PaymentList builds inline keyboard with payment statuses.
func PaymentList(users []*domain.User, paidMap map[int64]bool) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, u := range users {
		status := "❌"
		if paidMap[u.ID] {
			status = "✅"
		}
		label := status + " " + u.DisplayName()
		if u.Username != "" {
			label += " (@" + u.Username + ")"
		}
		var action string
		if paidMap[u.ID] {
			action = callback.AdmPayUnmark(u.ID)
		} else {
			action = callback.AdmPayConfirm(u.ID)
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, action),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("← Назад", callback.ActionAdmMenu),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// UserListForAction builds a user selection keyboard for a given action prefix.
func UserListForAction(users []*domain.User, actionFn func(int64) string) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, u := range users {
		label := u.DisplayName()
		if u.Username != "" {
			label += " (@" + u.Username + ")"
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, actionFn(u.ID)),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("← Назад", callback.ActionAdmMenu),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// UserDetail shows actions for a specific user.
func UserDetail(u *domain.User) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔗 Подключения", callback.AdmConnList(u.ID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗑 Удалить клиента", callback.AdmUserDelete(u.ID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("← Назад", callback.AdmUserList(0)),
		),
	)
}

// ConnectionManage shows connection list with admin actions.
func ConnectionManage(userID int64, conns []*domain.Connection) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, c := range conns {
		status := "✅"
		toggleLabel := "🚫 Заблокировать"
		if !c.IsActive {
			status = "❌"
			toggleLabel = "✅ Разблокировать"
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(status+" "+c.Label, callback.AdmConnList(userID)),
		))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(toggleLabel, callback.AdmConnToggle(c.UUID, !c.IsActive)),
			tgbotapi.NewInlineKeyboardButtonData("🗑 Удалить", callback.AdmConnDel(userID, c.UUID)),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("➕ Добавить подключение", callback.AdmConnAdd(userID)),
	))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("← Назад", callback.ActionAdmConnUsers),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// ConnPaymentTypeSelect asks admin to choose paid or free.
func ConnPaymentTypeSelect() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💳 Платное", callback.AdmConnCreate(false)),
			tgbotapi.NewInlineKeyboardButtonData("🆓 Бесплатное", callback.AdmConnCreate(true)),
		),
	)
}

// ConnPayButton shows the "I paid" button to the user after viewing payment info.
func ConnPayButton(uuid string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Я оплатил", callback.ConnPaid(uuid)),
		),
	)
}

// ConnPayConfirmButton shows the "Confirm payment" button to the admin.
func ConnPayConfirmButton(uuid string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Подтвердить оплату", callback.AdmConnPayOK(uuid)),
		),
	)
}

// PayConfirmButton builds a single-button keyboard for payment confirmation.
func PayConfirmButton(userID int64) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Подтвердить оплату", callback.AdmPayConfirm(userID)),
		),
	)
}

// FreeFriendList shows current free friends with remove buttons + an Add button.
func FreeFriendList(friends []*domain.User) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, u := range friends {
		label := "💚 " + u.DisplayName()
		if u.Username != "" {
			label += " (@" + u.Username + ")"
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label+" ❌", callback.AdmFreeFriendToggle(u.ID)),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("➕ Добавить друга", callback.ActionAdmFreeFriendAdd),
	))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("← Назад", callback.ActionAdmMenu),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// FreeFriendAddList shows non-friend users that can be added as free friends.
func FreeFriendAddList(nonFriends []*domain.User) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, u := range nonFriends {
		label := u.DisplayName()
		if u.Username != "" {
			label += " (@" + u.Username + ")"
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, callback.AdmFreeFriendToggle(u.ID)),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("← Назад", callback.ActionAdmFreeFriendList),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// PayDateUserList shows users for pay-date management.
func PayDateUserList(users []*domain.User) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, u := range users {
		label := u.DisplayName()
		if u.Username != "" {
			label += " (@" + u.Username + ")"
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, callback.AdmPayDateUser(u.ID)),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("← Назад", callback.ActionAdmMenu),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// PayDateConnList shows a user's connections so admin can select one for pay-date tracking.
func PayDateConnList(conns []PayDateConn, userID int64) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, c := range conns {
		label := "🔗 " + c.Label
		if c.LastPaidAt != nil {
			label += " 📅 " + c.LastPaidAt.Format("02.01.2006")
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, callback.AdmPayDateConn(c.UUID, userID)),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("← Назад", callback.AdmPayDateUser(userID)),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// PayDateConn is a lightweight struct for displaying connection pay-date info.
type PayDateConn struct {
	UUID       string
	Label      string
	LastPaidAt *time.Time
}
