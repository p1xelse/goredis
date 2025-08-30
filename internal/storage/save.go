package storage

type saveInfo struct {
	Data           map[string]string `json:"data"`
	ExpirationKeys map[string]int64  `json:"expiration_keys"`
}
