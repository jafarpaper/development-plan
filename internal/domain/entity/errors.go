package entity

import "errors"

var (
	ErrInvalidActivityName     = errors.New("invalid activity name")
	ErrInvalidCompanyID        = errors.New("invalid company id")
	ErrInvalidObjectName       = errors.New("invalid object name")
	ErrInvalidObjectID         = errors.New("invalid object id")
	ErrInvalidFormattedMessage = errors.New("invalid formatted message")
	ErrActivityLogNotFound     = errors.New("activity log not found")
	ErrInvalidActor            = errors.New("invalid actor")
)
