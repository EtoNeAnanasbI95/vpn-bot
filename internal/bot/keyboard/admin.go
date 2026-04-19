package keyboard

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/callback"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/domain"
)

// AdminPanel is the main admin panel inline keyboard.
func AdminPanel() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📢 Рассылка", callback.ActionAdmBcastMenu),
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
			tgbotapi.NewInlineKeyboardButtonData("👤 Одному клиенту", callback.Encode(callback.ActionAdmBcastUser)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("☑️ Выбранным клиентам", callback.ActionAdmBcastSelect),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("← Назад", callback.ActionAdmMenu),
		),
	)
}

// BroadcastMultiSelect shows all users with checkboxes. selectedIDs is a set of toggled IDs.
func BroadcastMultiSelect(users []*domain.User, selectedIDs map[int64]bool) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, u := range users {
		icon := "⬜"
		if selectedIDs[u.ID] {
			icon = "✅"
		}
		label := icon + " " + u.DisplayName()
		if u.Username != "" {
			label += " (@" + u.Username + ")"
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, callback.AdmBcastToggle(u.ID)),
		))
	}
	count := len(selectedIDs)
	confirmLabel := fmt.Sprintf("✅ Отправить выбранным (%d)", count)
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(confirmLabel, callback.ActionAdmBcastConfirm),
	))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("← Назад", callback.ActionAdmMenu),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
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

// ConnUserListWithAdd is like UserListForAction but adds a "➕ Новый клиент" button.
func ConnUserListWithAdd(users []*domain.User, actionFn func(int64) string) tgbotapi.InlineKeyboardMarkup {
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
		tgbotapi.NewInlineKeyboardButtonData("➕ Новый клиент", callback.ActionAdmConnNewUser),
	))
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
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("← Отмена", callback.ActionAdmCancel),
		),
	)
}

// CancelKeyboard returns a single-button keyboard for cancelling an active session.
func CancelKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("← Отмена", callback.ActionAdmCancel),
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

// ConnRequestAdminMenu shows the initial response options for a user's connection request.
func ConnRequestAdminMenu(reqUUID string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💚 Друг — бесплатно", callback.AdmReqFree(reqUUID)),
			tgbotapi.NewInlineKeyboardButtonData("💳 Платно", callback.AdmReqPaid(reqUUID)),
		),
	)
}

// ConnRequestPriceMenu shows price selection after admin chose "Платно".
func ConnRequestPriceMenu(reqUUID string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✏️ Своя цена", callback.AdmReqPriceCustom(reqUUID)),
			tgbotapi.NewInlineKeyboardButtonData("💵 Базовая 300₽", callback.AdmReqPriceBase(reqUUID)),
		),
	)
}

// ConnRequestConfirmPayButton shown to the admin when user claims they paid.
func ConnRequestConfirmPayButton(reqUUID string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Подтвердить оплату и выдать подключение", callback.AdmReqConfirmPay(reqUUID)),
		),
	)
}

