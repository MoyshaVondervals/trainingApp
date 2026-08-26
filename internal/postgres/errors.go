package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	codeUniqueViolation     = "23505"
	codeForeignKeyViolation = "23503"
)

func pgErrorCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

func isUniqueViolation(err error) bool {
	return pgErrorCode(err) == codeUniqueViolation
}

func isForeignKeyViolation(err error) bool {
	return pgErrorCode(err) == codeForeignKeyViolation
}
