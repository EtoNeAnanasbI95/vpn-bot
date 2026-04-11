package session

import "time"

// State represents what multi-step flow a user is currently in.
type State string

const (
	StateNone            State = ""
	StateBroadcastAll    State = "bcast_all"    // waiting for broadcast text (all users)
	StateBroadcastToUser State = "bcast_to_uid" // waiting for broadcast text (specific user)
	StateAddConnLabel       State = "add_conn_lbl"   // waiting for connection label
	StateAddConnPaymentType State = "add_conn_pay"   // waiting for admin to pick paid/free
	StateSetPaymentInfo     State = "set_pay_info"   // waiting for admin to enter payment credentials
	StateSetPayDate         State = "set_pay_date"       // waiting for admin to enter pay date (DD.MM.YYYY)
	StateBroadcastSelect    State = "bcast_sel"           // admin is picking users in multi-select
	StateBroadcastSelected  State = "bcast_sel_text"      // waiting for message text to send to selected users
)

// Data keys stored in Session.Data.
const (
	KeyTargetUserID  = "target_uid"
	KeyConnLabel     = "conn_lbl"
	KeyConnUserID    = "conn_uid"
	KeyConnTgTag     = "conn_tg_tag"
	KeyConnAdminID   = "conn_admin_id"
	KeyPayDateConnUUID    = "paydate_uuid"
	KeyPayDateConnUserID  = "paydate_uid"
	KeyPayDateConnAdminID = "paydate_aid"
	KeyBcastSelectedIDs   = "bcast_ids"  // comma-separated selected user IDs
	KeyBcastMsgID         = "bcast_msid" // message ID of the multi-select keyboard to edit
)

// Session holds transient per-user state between handler invocations.
type Session struct {
	State     State
	Data      map[string]string
	UpdatedAt time.Time
}

// Store is the interface for managing user sessions.
type Store interface {
	Get(userID int64) (*Session, bool)
	Set(userID int64, s *Session)
	Clear(userID int64)
}
