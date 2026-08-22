package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"trainingApp/internal/user"

	"github.com/jmoiron/sqlx"
)

const userColumns = `id, name, second_name, email, password_hash, created_at`

type UserRepo struct{ db *sqlx.DB }

func NewUserRepo(db *sqlx.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, u user.User) (user.User, error) {
	const q = `INSERT INTO users (name, second_name, email, password_hash) VALUES ($1, $2, $3, $4)
RETURNING ` + userColumns
	var created user.User
	err := r.db.GetContext(ctx, &created, q, u.Name, u.SecondName, u.Email, u.Password)
	if err != nil {
		return user.User{}, fmt.Errorf("create user: %w", err)
	}
	return created, nil
}

func (r *UserRepo) Delete(ctx context.Context, id int64) error {
	const q = `DELETE FROM users WHERE id = $1 RETURNING id`
	var deleted int64
	err := r.db.GetContext(ctx, &deleted, q, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return user.ErrNotFound
		}
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

func (r *UserRepo) GetById(ctx context.Context, id int64) (user.User, error) {
	const q = `SELECT ` + userColumns + ` FROM users WHERE id = $1`
	var u user.User
	err := r.db.GetContext(ctx, &u, q, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return user.User{}, user.ErrNotFound
		}
		return user.User{}, fmt.Errorf("get user: %w", err)
	}
	return u, nil
}

func (r *UserRepo) List(ctx context.Context, limit int) ([]user.User, error) {
	const q = `SELECT ` + userColumns + ` FROM users ORDER BY id LIMIT $1`
	result := make([]user.User, 0, limit)
	if err := r.db.SelectContext(ctx, &result, q, limit); err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return result, nil
}

func (r *UserRepo) FindByEmail(ctx context.Context, email string) (user.User, error) {
	const q = `SELECT ` + userColumns + ` FROM users WHERE email = $1`
	var u user.User
	err := r.db.GetContext(ctx, &u, q, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return user.User{}, user.ErrNotFound
		}
		return user.User{}, fmt.Errorf("find user: %w", err)
	}
	return u, nil
}
