package domain

type Platform struct {
	Key   string // e.g. "ios", "android" — used in callback_data
	Label string // e.g. "iOS", "Android" — shown in button
}
