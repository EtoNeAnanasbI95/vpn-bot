package domain

import "time"

type ConnRequestStatus string

const (
	ConnReqPending         ConnRequestStatus = "pending"          // ожидает ответа администратора
	ConnReqAwaitingPayment ConnRequestStatus = "awaiting_payment" // администратор выставил счёт, ждём оплаты
	ConnReqPaymentPending  ConnRequestStatus = "payment_pending"  // пользователь нажал "проверить оплату"
	ConnReqCompleted       ConnRequestStatus = "completed"        // подключение выдано
)

type ConnRequest struct {
	UUID      string
	UserID    int64
	AdminID   int64
	Status    ConnRequestStatus
	Amount    int
	CreatedAt time.Time
}
