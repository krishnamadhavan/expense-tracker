package domain

import "errors"

// Sentinel domain errors (map to HTTP in adapters later).
var (
	ErrInvalidArgument      = errors.New("invalid argument")
	ErrInvalidAmount        = errors.New("invalid amount")
	ErrInvalidDirection     = errors.New("invalid direction")
	ErrInvalidTransfer      = errors.New("invalid transfer")
	ErrCategoryKindMismatch = errors.New("category kind does not match transaction direction")
	ErrMissingAccount       = errors.New("account is required")
	ErrMissingHousehold     = errors.New("household is required")
	ErrVoidedTransaction    = errors.New("transaction is voided")
	ErrNotFound             = errors.New("not found")
	ErrCrossHousehold       = errors.New("resource belongs to another household")
)
