package domain

import "errors"

// Sentinel errors returned by the repository layer; handlers map these to
// API-contract error codes.
var (
	ErrNotFound      = errors.New("resource not found")
	ErrAlreadyExists = errors.New("resource already exists")
	ErrConflict      = errors.New("conflicting resource state")
)
