package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"trainingApp/internal/user"
)

type UserRepo struct{ db *sql.DB }

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, u user.User) (user.User, error) {
	const q = `INSERT INTO users (name, second_name, email, password_hash) VALUES ($1, $2, $3, $4) RETURNING id, created_at`
	err := r.db.QueryRowContext(ctx, q, u.Name, u.SecondName, u.Email, u.Password).Scan(&u.ID, &u.CreatedAt)
	if err != nil {
		return user.User{}, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

func (r *UserRepo) Delete(ctx context.Context, id int64) error {
	const q = `DELETE FROM users WHERE id = $1 RETURNING id`
	err := r.db.QueryRowContext(ctx, q, id).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return user.ErrNotFound
		}
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

func (r *UserRepo) GetById(ctx context.Context, id int64) (user.User, error) {
	const q = `SELECT id, name, second_name, email, password_hash, created_at FROM users WHERE id = $1`
	u := user.User{}
	err := r.db.QueryRowContext(ctx, q, id).
		Scan(&u.ID, &u.Name, &u.SecondName, &u.Email, &u.Password, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return user.User{}, user.ErrNotFound
		}
		return user.User{}, fmt.Errorf("get user: %w", err)
	}
	return u, nil
}

func (r *UserRepo) List(ctx context.Context, limit int) ([]user.User, error) {
	const q = `SELECT id, name, second_name, email, password_hash, created_at FROM users LIMIT $1`
	rows, err := r.db.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()
	result := make([]user.User, 0, limit)
	for rows.Next() {
		var u user.User
		if err := rows.Scan(&u.ID, &u.Name, &u.SecondName, &u.Email, &u.Password, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("list users: %w", err)
		}
		result = append(result, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return result, nil
}
func (r *UserRepo) FindByEmail(ctx context.Context, email string) (user.User, error) {
	const q = `SELECT id, name, second_name, email, password_hash, created_at FROM users WHERE email = $1`
	u := user.User{}
	err := r.db.QueryRowContext(ctx, q, email).
		Scan(&u.ID, &u.Name, &u.SecondName, &u.Email, &u.Password, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return user.User{}, user.ErrNotFound
		}
		return user.User{}, fmt.Errorf("find user: %w", err)
	}
	return u, nil
}
