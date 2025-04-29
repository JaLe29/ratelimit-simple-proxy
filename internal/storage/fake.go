package storage

// IPRateLimiter představuje rate limiter na základě IP adres
type IPFakeStorage struct {
}

// NewIPRateLimiter vytvoří novou instanci rate limiteru
func NewFakeStorage() *IPFakeStorage {
	return &IPFakeStorage{}
}

func (r *IPFakeStorage) CheckLimit(ipAddress string) bool {
	return false
}
