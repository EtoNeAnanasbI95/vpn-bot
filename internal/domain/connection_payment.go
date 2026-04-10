package domain

import "time"

type ConnPayStatus string

const (
	ConnPayFree    ConnPayStatus = "free"    // no payment needed
	ConnPayUnpaid  ConnPayStatus = "unpaid"  // payment required, not yet done
	ConnPayPending ConnPayStatus = "pending" // user claims paid, awaiting admin confirmation
	ConnPayPaid    ConnPayStatus = "paid"    // admin confirmed
)

type ConnPayment struct {
	UUID       string
	UserID     int64
	AdminID    int64 // admin who issued the connection
	Status     ConnPayStatus
	LastPaidAt *time.Time
	CreatedAt  time.Time
}
