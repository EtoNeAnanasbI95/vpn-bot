package domain

import "time"

type User struct {
	ID        int64
	Username  string
	FirstName string
	LastName  string
	AdminID   int64
	CreatedAt time.Time
}

func (u *User) DisplayName() string {
	name := u.FirstName
	if u.LastName != "" {
		name += " " + u.LastName
	}
	return name
}

func (u *User) Mention() string {
	if u.Username != "" {
		return "@" + u.Username
	}
	return u.DisplayName()
}

// TelegramUser is the data arriving from a Telegram update.
type TelegramUser struct {
	ID        int64
	Username  string
	FirstName string
	LastName  string
}
