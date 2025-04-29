package storage

type Storage interface {
	CheckLimit(ipAddress string) bool
}
