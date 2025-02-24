package config

import "errors"

var (
	ErrInvalidConfig         = errors.New("invalid configuration")
	ErrInvalidDatabaseConfig = errors.New("invalid database configuration")
)
