package domain

import "time"

type Session struct {
	ExpiresAt time.Time
}
