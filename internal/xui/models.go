package xui

import (
	"encoding/json"
	"strconv"
)

// FlexInt64 unmarshals from either a JSON number or a JSON string containing a number.
// 3x-ui stores tgId inconsistently across versions.
type FlexInt64 int64

func (f *FlexInt64) UnmarshalJSON(b []byte) error {
	// Try number first.
	var n int64
	if err := json.Unmarshal(b, &n); err == nil {
		*f = FlexInt64(n)
		return nil
	}
	// Fall back to string.
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if s == "" {
		*f = 0
		return nil
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}
	*f = FlexInt64(n)
	return nil
}

// apiResponse is the common envelope for all 3x-ui API responses.
type apiResponse struct {
	Success bool            `json:"success"`
	Msg     string          `json:"msg"`
	Obj     json.RawMessage `json:"obj"`
}

// Inbound represents a single inbound from /panel/api/inbounds/list.
type Inbound struct {
	ID             int    `json:"id"`
	Port           int    `json:"port"`
	Protocol       string `json:"protocol"`
	Enable         bool   `json:"enable"`
	Settings       string `json:"settings"`       // double-encoded JSON → InboundSettings
	StreamSettings string `json:"streamSettings"` // double-encoded JSON → StreamSettings
}

// InboundSettings is the decoded "settings" field.
type InboundSettings struct {
	Clients []XUIClient `json:"clients"`
}

// XUIClient is a client entry inside InboundSettings.
type XUIClient struct {
	ID         string `json:"id"`
	Flow       string `json:"flow"`
	Email      string `json:"email"`
	LimitIp    int    `json:"limitIp"`
	TotalGB    int64  `json:"totalGB"`
	ExpiryTime int64  `json:"expiryTime"`
	Enable     bool   `json:"enable"`
	TgId       FlexInt64 `json:"tgId"`
	SubId      string `json:"subId"`
	Comment    string `json:"comment"`
	Reset      int    `json:"reset"`
}

// StreamSettings is the decoded "streamSettings" field.
type StreamSettings struct {
	Network         string          `json:"network"`
	Security        string          `json:"security"`
	RealitySettings *RealityConfig  `json:"realitySettings"`
}

// RealityConfig holds the Reality-specific stream configuration.
type RealityConfig struct {
	ServerNames []string        `json:"serverNames"`
	ShortIds    []string        `json:"shortIds"`
	Settings    RealitySettings `json:"settings"`
}

// RealitySettings contains the public-facing Reality parameters
// needed to build a VLESS connection URI.
type RealitySettings struct {
	PublicKey   string `json:"publicKey"`
	Fingerprint string `json:"fingerprint"`
	SpiderX     string `json:"spiderX"`
}

// updateClientBody is the JSON body for /panel/api/inbounds/updateClient/{uuid}.
type updateClientBody struct {
	InboundID int    `json:"id"`
	Settings  string `json:"settings"` // JSON-encoded InboundSettings (double-encoded)
}
