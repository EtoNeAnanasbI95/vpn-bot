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
	ActionAdmBcastAll   = "adm_bcast_all"
	ActionAdmBcastUser  = "adm_bcast_user"
	ActionAdmUserList        = "adm_user_list"
	ActionAdmUserDelete      = "adm_user_del"
	ActionAdmFreeFriendList   = "adm_ff_list"   // show current free friends
	ActionAdmFreeFriendAdd    = "adm_ff_add"    // show non-friends to add
	ActionAdmFreeFriendToggle = "adm_ff_toggle" // toggle free-friend for a user
	ActionAdmPayDateList      = "adm_pd_list"   // list users for pay-date management
	ActionAdmPayDateUser      = "adm_pd_user"   // list connections for a user
	ActionAdmPayDateConn      = "adm_pd_conn"   // start session to enter date for a connection
	ActionMainMenu           = "main_menu"
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
func AdmFreeFriendToggle(userID int64) string {
	return fmt.Sprintf("%s%s%d", ActionAdmFreeFriendToggle, sep, userID)
}
func AdmPayDateUser(userID int64) string {
	return fmt.Sprintf("%s%s%d", ActionAdmPayDateUser, sep, userID)
}
func AdmPayDateConn(connUUID string, userID int64) string {
	return fmt.Sprintf("%s%s%s%s%d", ActionAdmPayDateConn, sep, connUUID, sep, userID)
}
