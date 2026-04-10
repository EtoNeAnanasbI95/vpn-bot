package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/EtoNeAnanasbI95/vpn-bot/internal/domain"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/repository"
)

type connPayRepo struct{ db *sql.DB }

func NewConnectionPaymentRepository(db *sql.DB) repository.ConnectionPaymentRepository {
	return &connPayRepo{db: db}
}

func (r *connPayRepo) Create(ctx context.Context, p *domain.ConnPayment) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO connection_payments (uuid, user_id, admin_id, status, created_at)
		VALUES (?, ?, ?, ?, datetime('now'))
		ON CONFLICT(uuid) DO NOTHING
	`, p.UUID, p.UserID, p.AdminID, string(p.Status))
	return err
}

func (r *connPayRepo) GetByUUID(ctx context.Context, uuid string) (*domain.ConnPayment, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT uuid, user_id, admin_id, status, last_paid_at, created_at
		FROM connection_payments WHERE uuid = ?`, uuid)
	return scanConnPay(row)
}

func (r *connPayRepo) GetAllUnpaid(ctx context.Context) ([]*domain.ConnPayment, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT uuid, user_id, admin_id, status, last_paid_at, created_at
		FROM connection_payments WHERE status = 'unpaid'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanConnPays(rows)
}

func (r *connPayRepo) GetOverdue(ctx context.Context, olderThan time.Time) ([]*domain.ConnPayment, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT uuid, user_id, admin_id, status, last_paid_at, created_at
		FROM connection_payments
		WHERE status = 'unpaid' AND created_at <= ?`,
		olderThan.UTC().Format("2006-01-02 15:04:05"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanConnPays(rows)
}

func (r *connPayRepo) SetStatus(ctx context.Context, uuid string, status domain.ConnPayStatus) error {
	_, err := r.db.ExecContext(ctx, `UPDATE connection_payments SET status = ? WHERE uuid = ?`, string(status), uuid)
	return err
}

func (r *connPayRepo) GetAdminPaymentInfo(ctx context.Context, adminID int64) (string, error) {
	var info string
	err := r.db.QueryRowContext(ctx, `SELECT payment_info FROM admin_profiles WHERE admin_id = ?`, adminID).Scan(&info)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return info, err
}

func (r *connPayRepo) SetAdminPaymentInfo(ctx context.Context, adminID int64, info string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO admin_profiles (admin_id, payment_info) VALUES (?, ?)
		ON CONFLICT(admin_id) DO UPDATE SET payment_info = excluded.payment_info
	`, adminID, info)
	return err
}

func (r *connPayRepo) SetLastPaidAt(ctx context.Context, uuid string, userID, adminID int64, paidAt *time.Time) error {
	var val interface{}
	if paidAt != nil {
		val = paidAt.UTC().Format("2006-01-02 15:04:05")
	}
	// Upsert: create the row if absent (marking it as paid), then set last_paid_at.
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO connection_payments (uuid, user_id, admin_id, status, last_paid_at, created_at)
		VALUES (?, ?, ?, 'paid', ?, datetime('now'))
		ON CONFLICT(uuid) DO UPDATE SET last_paid_at = excluded.last_paid_at
	`, uuid, userID, adminID, val)
	return err
}

func (r *connPayRepo) GetConnsWithDuePaidReminder(ctx context.Context) ([]*domain.ConnPayment, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT cp.uuid, cp.user_id, cp.admin_id, cp.status, cp.last_paid_at, cp.created_at
		FROM connection_payments cp
		JOIN users u ON u.id = cp.user_id
		WHERE cp.last_paid_at IS NOT NULL
		  AND u.is_free_friend = 0
		  AND date(cp.last_paid_at, '+1 month') <= date('now')`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanConnPays(rows)
}

func scanConnPay(row *sql.Row) (*domain.ConnPayment, error) {
	var p domain.ConnPayment
	var status, createdAt string
	var lastPaidAt sql.NullString
	if err := row.Scan(&p.UUID, &p.UserID, &p.AdminID, &status, &lastPaidAt, &createdAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("connection payment not found")
		}
		return nil, err
	}
	p.Status = domain.ConnPayStatus(status)
	if lastPaidAt.Valid {
		t, _ := time.Parse("2006-01-02 15:04:05", lastPaidAt.String)
		p.LastPaidAt = &t
	}
	p.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	return &p, nil
}

func scanConnPays(rows *sql.Rows) ([]*domain.ConnPayment, error) {
	var out []*domain.ConnPayment
	for rows.Next() {
		var p domain.ConnPayment
		var status, createdAt string
		var lastPaidAt sql.NullString
		if err := rows.Scan(&p.UUID, &p.UserID, &p.AdminID, &status, &lastPaidAt, &createdAt); err != nil {
			return nil, err
		}
		p.Status = domain.ConnPayStatus(status)
		if lastPaidAt.Valid {
			t, _ := time.Parse("2006-01-02 15:04:05", lastPaidAt.String)
			p.LastPaidAt = &t
		}
		p.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		out = append(out, &p)
	}
	return out, rows.Err()
}
