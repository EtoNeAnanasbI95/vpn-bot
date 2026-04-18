package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/EtoNeAnanasbI95/vpn-bot/internal/domain"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/repository"
)

type connRequestRepo struct{ db *sql.DB }

func NewConnRequestRepository(db *sql.DB) repository.ConnRequestRepository {
	return &connRequestRepo{db: db}
}

func (r *connRequestRepo) Create(ctx context.Context, req *domain.ConnRequest) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO connection_requests (uuid, user_id, admin_id, status, amount)
		VALUES (?, ?, 0, 'pending', 0)
	`, req.UUID, req.UserID)
	return err
}

func (r *connRequestRepo) GetByUUID(ctx context.Context, uuid string) (*domain.ConnRequest, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT uuid, user_id, admin_id, status, amount, created_at
		FROM connection_requests WHERE uuid = ?`, uuid)
	return scanConnRequest(row)
}

func (r *connRequestRepo) GetActiveByUserID(ctx context.Context, userID int64) (*domain.ConnRequest, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT uuid, user_id, admin_id, status, amount, created_at
		FROM connection_requests
		WHERE user_id = ? AND status != 'completed'
		ORDER BY created_at DESC LIMIT 1`, userID)
	return scanConnRequest(row)
}

// Claim atomically sets admin_id and status only if the request is still 'pending'.
// Returns true if the claim succeeded, false if already claimed by another admin.
func (r *connRequestRepo) Claim(ctx context.Context, uuid string, adminID int64, status domain.ConnRequestStatus) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE connection_requests SET admin_id = ?, status = ?
		WHERE uuid = ? AND status = 'pending'
	`, adminID, string(status), uuid)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (r *connRequestRepo) UpdateStatus(ctx context.Context, uuid string, status domain.ConnRequestStatus) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE connection_requests SET status = ? WHERE uuid = ?
	`, string(status), uuid)
	return err
}

func (r *connRequestRepo) SetAmount(ctx context.Context, uuid string, amount int) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE connection_requests SET amount = ? WHERE uuid = ?
	`, amount, uuid)
	return err
}

func scanConnRequest(row *sql.Row) (*domain.ConnRequest, error) {
	var req domain.ConnRequest
	var status, createdAt string
	if err := row.Scan(&req.UUID, &req.UserID, &req.AdminID, &status, &req.Amount, &createdAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("connection request not found")
		}
		return nil, err
	}
	req.Status = domain.ConnRequestStatus(status)
	req.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	return &req, nil
}
