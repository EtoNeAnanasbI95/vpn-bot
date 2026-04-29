package callback

import (
	"fmt"
	"strings"
)

const sep = "|"

// Actions — all possible values for callback_data[0].
const (
	ActionConnList      = "conn_list"
	ActionConnQR        = "conn_qr"
	ActionConnPay       = "conn_pay"       // user opens payment info
	ActionConnPaid      = "conn_paid"      // user claims paid
	ActionAdmConnPayOK  = "adm_conn_payok" // admin confirms connection payment
	ActionGuideList     = "guide_list"
	ActionGuideGet      = "guide_get"
	ActionAdmMenu       = "adm_menu"
	ActionAdmPayList    = "adm_pay_list"
	ActionAdmPayConfirm = "adm_pay_confirm"
	ActionAdmPayUnmark  = "adm_pay_unmark"
	ActionAdmConnUsers  = "adm_conn_users"
	ActionAdmConnList   = "adm_conn_list"
	ActionAdmConnAdd    = "adm_conn_add"
	ActionAdmConnCreate = "adm_conn_create" // finalize connection after payment type chosen
	ActionAdmConnDel    = "adm_conn_del"
	ActionAdmConnToggle = "adm_conn_toggle"
	ActionAdmSetPayInfo = "adm_set_pay_info"
	ActionAdmBcastMenu       = "adm_bcast_menu"   // show broadcast type menu
	ActionAdmBcastAll        = "adm_bcast_all"
	ActionAdmBcastUser       = "adm_bcast_user"
	ActionAdmBcastSelect     = "adm_bcast_sel"    // open multi-select list
	ActionAdmBcastToggle     = "adm_bcast_tog"    // toggle a user in the selection
	ActionAdmBcastConfirm    = "adm_bcast_csel"   // confirm selection → ask for message
	ActionAdmUserList        = "adm_user_list"
	ActionAdmUserDelete      = "adm_user_del"
	ActionAdmFreeFriendList   = "adm_ff_list"   // show current free friends
	ActionAdmFreeFriendAdd    = "adm_ff_add"    // show non-friends to add
	ActionAdmFreeFriendToggle = "adm_ff_toggle" // toggle free-friend for a user
	ActionAdmConnNewUser      = "adm_conn_new"  // manually add a new user by Telegram ID
	ActionAdmCancel           = "adm_cancel"    // cancel active session and navigate back
	ActionAdmPayDateUsers = "adm_pd_users" // show user list for pay-date assignment
	ActionAdmPayDateConns = "adm_pd_conns" // show connection list for a user
	ActionAdmPayDateSet   = "adm_pd_set"   // start day-of-month input for a connection
	ActionMainMenu           = "main_menu"

	// Connection request flow (user-initiated).
	ActionConnRequest       = "conn_request"    // user requests a new connection
	ActionAdmReqFree        = "adm_req_free"    // admin: approve as friend (free)
	ActionAdmReqPaid        = "adm_req_paid"    // admin: approve as paid
	ActionAdmReqPriceBase   = "adm_req_base"    // admin: use base price 300₽
	ActionAdmReqPriceCustom = "adm_req_custom"  // admin: enter custom price
	ActionConnReqCheckPay   = "conn_req_check"  // user: notify admin of payment
	ActionAdmReqConfirmPay  = "adm_req_confirm" // admin: confirm payment and issue connection
)

func Encode(parts ...string) string {
	return strings.Join(parts, sep)
}

func Decode(data string) []string {
	return strings.SplitN(data, sep, 10)
}

// Helpers for building callback strings.

func ConnQR(uuid string) string          { return Encode(ActionConnQR, uuid) }
func ConnPay(uuid string) string         { return Encode(ActionConnPay, uuid) }
func ConnPaid(uuid string) string        { return Encode(ActionConnPaid, uuid) }
func AdmConnPayOK(uuid string) string    { return Encode(ActionAdmConnPayOK, uuid) }
func AdmConnCreate(isFree bool) string {
	v := "0"
	if isFree {
		v = "1"
	}
	return Encode(ActionAdmConnCreate, v)
}
func GuideGet(key string) string        { return Encode(ActionGuideGet, key) }
func AdmPayConfirm(userID int64) string { return fmt.Sprintf("%s%s%d", ActionAdmPayConfirm, sep, userID) }
func AdmPayUnmark(userID int64) string  { return fmt.Sprintf("%s%s%d", ActionAdmPayUnmark, sep, userID) }
func AdmConnList(userID int64) string   { return fmt.Sprintf("%s%s%d", ActionAdmConnList, sep, userID) }
func AdmConnAdd(userID int64) string    { return fmt.Sprintf("%s%s%d", ActionAdmConnAdd, sep, userID) }
func AdmConnDel(userID int64, uuid string) string {
	return fmt.Sprintf("%s%s%d%s%s", ActionAdmConnDel, sep, userID, sep, uuid)
}
func AdmConnToggle(uuid string, enable bool) string {
	v := 0
	if enable {
		v = 1
	}
	return fmt.Sprintf("%s%s%s%s%d", ActionAdmConnToggle, sep, uuid, sep, v)
}
func AdmUserList(page int) string    { return fmt.Sprintf("%s%s%d", ActionAdmUserList, sep, page) }
func AdmUserDelete(userID int64) string { return fmt.Sprintf("%s%s%d", ActionAdmUserDelete, sep, userID) }
func AdmBcastUser(userID int64) string {
	return fmt.Sprintf("%s%s%d", ActionAdmBcastUser, sep, userID)
}
func AdmBcastToggle(userID int64) string {
	return fmt.Sprintf("%s%s%d", ActionAdmBcastToggle, sep, userID)
}
func AdmFreeFriendToggle(userID int64) string {
	return fmt.Sprintf("%s%s%d", ActionAdmFreeFriendToggle, sep, userID)
}
func AdmReqFree(reqUUID string) string        { return Encode(ActionAdmReqFree, reqUUID) }
func AdmReqPaid(reqUUID string) string        { return Encode(ActionAdmReqPaid, reqUUID) }
func AdmReqPriceBase(reqUUID string) string   { return Encode(ActionAdmReqPriceBase, reqUUID) }
func AdmReqPriceCustom(reqUUID string) string { return Encode(ActionAdmReqPriceCustom, reqUUID) }
func ConnReqCheckPay(reqUUID string) string   { return Encode(ActionConnReqCheckPay, reqUUID) }
func AdmReqConfirmPay(reqUUID string) string  { return Encode(ActionAdmReqConfirmPay, reqUUID) }
func AdmPayDateConns(userID int64) string     { return fmt.Sprintf("%s%s%d", ActionAdmPayDateConns, sep, userID) }
func AdmPayDateSet(uuid string) string        { return Encode(ActionAdmPayDateSet, uuid) }
