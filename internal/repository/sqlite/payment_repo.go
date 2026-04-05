package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/EtoNeAnanasbI95/vpn-bot/internal/domain"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/repository"
)

type paymentRepo struct{ db *sql.DB }

func NewPaymentRepository(db *sql.DB) repository.PaymentRepository {
	return &paymentRepo{db: db}
}

func (r *paymentRepo) GetOrCreate(ctx context.Context, userID int64, year, month int) (*domain.Payment, error) {
	_, err := r.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO payments (user_id, period_year, period_month, created_at)
		VALUES (?, ?, ?, datetime('now'))
	`, userID, year, month)
	if err != nil {
		return nil, err
	}
	return r.GetByUserAndPeriod(ctx, userID, year, month)
}

func (r *paymentRepo) GetByUserAndPeriod(ctx context.Context, userID int64, year, month int) (*domain.Payment, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, period_year, period_month, paid_at, confirmed_by, created_at
		FROM payments WHERE user_id = ? AND period_year = ? AND period_month = ?
	`, userID, year, month)
	return scanPayment(row)
}

func (r *paymentRepo) GetUnpaidForPeriod(ctx context.Context, year, month int) ([]*domain.Payment, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, period_year, period_month, paid_at, confirmed_by, created_at
		FROM payments
		WHERE period_year = ? AND period_month = ? AND paid_at IS NULL
	`, year, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPayments(rows)
}

func (r *paymentRepo) MarkPaid(ctx context.Context, userID int64, year, month int, confirmedBy int64, paidAt time.Time) error {
	paidAtStr := paidAt.UTC().Format("2006-01-02 15:04:05")
	_, err := r.db.ExecContext(ctx, `
		UPDATE payments SET paid_at = ?, confirmed_by = ?
		WHERE user_id = ? AND period_year = ? AND period_month = ?
	`, paidAtStr, confirmedBy, userID, year, month)
	return err
}

func (r *paymentRepo) MarkUnpaid(ctx context.Context, userID int64, year, month int) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE payments SET paid_at = NULL, confirmed_by = NULL
		WHERE user_id = ? AND period_year = ? AND period_month = ?
	`, userID, year, month)
	return err
}

func scanPayment(row *sql.Row) (*domain.Payment, error) {
	var p domain.Payment
	var paidAt sql.NullString
	var confirmedBy sql.NullInt64
	var createdAt string
	if err := row.Scan(&p.ID, &p.UserID, &p.PeriodYear, &p.PeriodMonth, &paidAt, &confirmedBy, &createdAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if paidAt.Valid {
		t, _ := time.Parse("2006-01-02 15:04:05", paidAt.String)
		p.PaidAt = &t
	}
	if confirmedBy.Valid {
		p.ConfirmedBy = &confirmedBy.Int64
	}
	p.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	return &p, nil
}

func scanPayments(rows *sql.Rows) ([]*domain.Payment, error) {
	var payments []*domain.Payment
	for rows.Next() {
		var p domain.Payment
		var paidAt sql.NullString
		var confirmedBy sql.NullInt64
		var createdAt string
		if err := rows.Scan(&p.ID, &p.UserID, &p.PeriodYear, &p.PeriodMonth, &paidAt, &confirmedBy, &createdAt); err != nil {
			return nil, err
		}
		if paidAt.Valid {
			t, _ := time.Parse("2006-01-02 15:04:05", paidAt.String)
			p.PaidAt = &t
		}
		if confirmedBy.Valid {
			p.ConfirmedBy = &confirmedBy.Int64
		}
		p.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		payments = append(payments, &p)
	}
	return payments, rows.Err()
}
