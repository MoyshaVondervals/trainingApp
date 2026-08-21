package user

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

var ErrNotFound = errors.New("user not found")

const (
	maxUserNameLen       = 100
	maxUserSecondNameLen = 100
	maxEmailNameLen      = 255
	minPasswordLen       = 8
	maxPasswordLen       = 72
)

type User struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	SecondName string    `json:"second_name"`
	Email      string    `json:"email"`
	Password   string    `json:"-"`
	CreatedAt  time.Time `json:"created_at"`
}

func (u *User) Validate() error {
	if strings.TrimSpace(u.Name) == "" {
		return errors.New("name is required")
	}
	if n := utf8.RuneCountInString(u.Name); n > maxUserNameLen {
		return fmt.Errorf("name must be at most %d characters, got %d", maxUserNameLen, n)
	}

	if strings.TrimSpace(u.SecondName) == "" {
		return errors.New("second name is required")
	}
	if n := utf8.RuneCountInString(u.SecondName); n > maxUserSecondNameLen {
		return fmt.Errorf("second name must be at most %d characters, got %d", maxUserSecondNameLen, n)
	}

	if err := ValidateEmail(u.Email); err != nil {
		return err
	}

	return nil
}

func ValidateEmail(email string) error {
	if strings.TrimSpace(email) == "" {
		return errors.New("email is required")
	}
	if n := utf8.RuneCountInString(email); n > maxEmailNameLen {
		return fmt.Errorf("email must be at most %d characters, got %d", maxEmailNameLen, n)
	}
	if !strings.Contains(email, "@") {
		return errors.New("email must contain @")
	}
	return nil
}

func ValidatePassword(plain string) error {
	if strings.TrimSpace(plain) == "" {
		return errors.New("password is required")
	}
	if n := len(plain); n < minPasswordLen {
		return fmt.Errorf("password must be at least %d characters, got %d", minPasswordLen, n)
	}
	if n := len(plain); n > maxPasswordLen {
		return fmt.Errorf("password must be at most %d bytes, got %d", maxPasswordLen, n)
	}
	return nil
}
