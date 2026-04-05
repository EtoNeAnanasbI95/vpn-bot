package domain

import (
	"fmt"
	"time"
)

type Payment struct {
	ID          int64
	UserID      int64
	PeriodYear  int
	PeriodMonth int
	PaidAt      *time.Time
	ConfirmedBy *int64
	CreatedAt   time.Time
}

func (p *Payment) IsPaid() bool {
	return p.PaidAt != nil
}

func (p *Payment) PeriodLabel() string {
	months := [...]string{
		"Январь", "Февраль", "Март", "Апрель", "Май", "Июнь",
		"Июль", "Август", "Сентябрь", "Октябрь", "Ноябрь", "Декабрь",
	}
	if p.PeriodMonth < 1 || p.PeriodMonth > 12 {
		return fmt.Sprintf("%d/%d", p.PeriodMonth, p.PeriodYear)
	}
	return fmt.Sprintf("%s %d", months[p.PeriodMonth-1], p.PeriodYear)
}
