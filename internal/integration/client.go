package integration

import (
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
)

type RateLimitError struct {
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limit exceeded, retrying after %v", e.RetryAfter)
}

func NewClient() *resty.Client {
	c := resty.New().
		SetTimeout(5 * time.Second)
	return c
}
