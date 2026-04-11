package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/EtoNeAnanasbI95/vpn-bot/internal/domain"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/repository"
)

type userRepo struct{ db *sql.DB }

func NewUserRepository(db *sql.DB) repository.UserRepository {
	return &userRepo{db: db}
}

func (r *userRepo) Upsert(ctx context.Context, u *domain.User) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO users (id, username, first_name, last_name, admin_id, is_blocked, created_at)
		VALUES (?, ?, ?, ?, ?, 0, datetime('now'))
		ON CONFLICT(id) DO UPDATE SET
			username   = excluded.username,
			first_name = excluded.first_name,
			last_name  = excluded.last_name
	`, u.ID, u.Username, u.FirstName, u.LastName, u.AdminID)
	return err
}

func (r *userRepo) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, username, first_name, last_name, admin_id, is_free_friend, is_blocked, created_at
		FROM users WHERE id = ?`, id)
	return scanUser(row)
}

func (r *userRepo) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, username, first_name, last_name, admin_id, is_free_friend, is_blocked, created_at
		FROM users WHERE username = ?`, username)
	return scanUser(row)
}

func (r *userRepo) GetAll(ctx context.Context) ([]*domain.User, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, username, first_name, last_name, admin_id, is_free_friend, is_blocked, created_at
		FROM users ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUsers(rows)
}

func (r *userRepo) GetByAdminID(ctx context.Context, adminID int64) ([]*domain.User, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, username, first_name, last_name, admin_id, is_free_friend, is_blocked, created_at
		FROM users WHERE admin_id = ? ORDER BY first_name`, adminID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUsers(rows)
}

func (r *userRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	return err
}

func (r *userRepo) CountByAdminID(ctx context.Context, adminID int64) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE admin_id = ?`, adminID).Scan(&count)
	return count, err
}

func (r *userRepo) SetFreeFriend(ctx context.Context, userID int64, isFree bool) error {
	val := 0
	if isFree {
		val = 1
	}
	_, err := r.db.ExecContext(ctx, `UPDATE users SET is_free_friend = ? WHERE id = ?`, val, userID)
	return err
}

func (r *userRepo) GetFreeFriends(ctx context.Context) ([]*domain.User, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, username, first_name, last_name, admin_id, is_free_friend, is_blocked, created_at
		FROM users WHERE is_free_friend = 1 ORDER BY first_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUsers(rows)
}

func (r *userRepo) GetNonFriends(ctx context.Context) ([]*domain.User, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, username, first_name, last_name, admin_id, is_free_friend, is_blocked, created_at
		FROM users WHERE is_free_friend = 0 ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUsers(rows)
}

func scanUser(row *sql.Row) (*domain.User, error) {
	var u domain.User
	var isFreeFriend int
	var ignored int // is_blocked kept for backwards compat
	var createdAt string
	err := row.Scan(&u.ID, &u.Username, &u.FirstName, &u.LastName, &u.AdminID, &isFreeFriend, &ignored, &createdAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, err
	}
	u.IsFreeFriend = isFreeFriend == 1
	u.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	return &u, nil
}

func scanUsers(rows *sql.Rows) ([]*domain.User, error) {
	var users []*domain.User
	for rows.Next() {
		var u domain.User
		var isFreeFriend int
		var ignored int
		var createdAt string
		if err := rows.Scan(&u.ID, &u.Username, &u.FirstName, &u.LastName, &u.AdminID, &isFreeFriend, &ignored, &createdAt); err != nil {
			return nil, err
		}
		u.IsFreeFriend = isFreeFriend == 1
		u.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		users = append(users, &u)
	}
	return users, rows.Err()
}
