package storage

type IPFakeStorage struct {
	//
}

func NewFakeStorage() *IPFakeStorage {
	return &IPFakeStorage{}
}

func (r *IPFakeStorage) CheckLimit(ipAddress string) bool {
	return false
}

func (r *IPFakeStorage) Close() error {
	return nil
}
