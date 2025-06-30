package storage

import "io"

type Storage interface {
	// CheckLimit checks if the user has reached the limit of requests per second
	CheckLimit(ipAddress string) bool
	// Close for graceful shutdown
	io.Closer
}
