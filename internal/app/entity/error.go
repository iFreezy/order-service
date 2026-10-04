package entity

import "net/http"

type AppError struct {
	status  int
	message string
}

func NewAppError(status int, message string) *AppError {
	return &AppError{status: status, message: message}
}
func (e *AppError) Error() string   { return e.message }
func (e *AppError) HTTPStatus() int { return e.status }

var (
	ErrNotFound      = NewAppError(http.StatusNotFound, "not found")
	ErrAlreadyExists = NewAppError(http.StatusConflict, "already exists")
)
