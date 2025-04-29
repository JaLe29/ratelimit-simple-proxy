package storage

type Storage interface {
	// CheckLimit checks if the user has reached the limit of requests per second
	CheckLimit(ipAddress string) bool
}
