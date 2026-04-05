package domain

// Connection is a view-model for a single VPN client on a 3x-ui inbound.
// Client data lives in 3x-ui; payment status lives in the local DB.
type Connection struct {
	UUID      string        // XUIClient.ID
	UserID    int64         // parsed from XUIClient.TgId
	Label     string        // XUIClient.Comment
	Link      string        // VLESS URI, reconstructed from inbound streamSettings
	IsActive  bool          // XUIClient.Enable
	PayStatus ConnPayStatus // from connection_payments table; default "free" if no record
	AdminID   int64         // who issued this connection (for payment info lookup)
}
