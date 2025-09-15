package user

import (
	"context"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           int64
	Username     string
	PasswordHash string
	CreatedAt    time.Time
}

type UserRepository interface {
	Create(ctx context.Context, username, password string) (int64, error)
	GetByID(ctx context.Context, id int64) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	VerifyPassword(ctx context.Context, username, password string) (*User, error)
}

type userRepositoryImpl struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) UserRepository {
	return &userRepositoryImpl{db: db}
}

func (r *userRepositoryImpl) Create(ctx context.Context, username, password string) (int64, error) {
	// hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}

	var id int64
	err = r.db.QueryRowContext(ctx,
		`INSERT INTO users (username, password_hash) VALUES ($1, $2) RETURNING id`,
		username, string(hash)).Scan(&id)
	return id, err
}

func (r *userRepositoryImpl) GetByID(ctx context.Context, id int64) (*User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, created_at FROM users WHERE id=$1`, id)
	u := &User{}
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt); err != nil {
		return nil, err
	}
	return u, nil
}

func (r *userRepositoryImpl) GetByUsername(ctx context.Context, username string) (*User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, created_at FROM users WHERE username=$1`, username)
	u := &User{}
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt); err != nil {
		return nil, err
	}
	return u, nil
}

func (r *userRepositoryImpl) VerifyPassword(ctx context.Context, username, password string) (*User, error) {
	u, err := r.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return nil, errors.New("invalid credentials")
	}
	return u, nil
}
