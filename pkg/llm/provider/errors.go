package provider

import "errors"

var (
	// ErrRateLimitExceeded is returned when the provider rejects a request due to quota limitations.
	ErrRateLimitExceeded = errors.New("rate limit exceeded")

	// ErrContextLengthExceeded is returned when the input messages exceed the model's context window.
	ErrContextLengthExceeded = errors.New("context length exceeded")

	// ErrProviderUnavailable is returned when the provider API is unreachable or returned a 5xx error.
	ErrProviderUnavailable = errors.New("provider unavailable")
)
