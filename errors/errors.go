package errors

import "errors"

var (
	ErrNoHealthyTargets = errors.New("no healthy targets available")
	ErrUnknownAlg       = errors.New("unknown load balancer algorithm")
	ErrPrefixNotFound   = errors.New("prefix not found")
)
