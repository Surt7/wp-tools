package generator

import (
	"crypto/rand"
	"math/big"
)

func GenerateRandomPassword(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*-_"

	if length <= 0 {
		length = 24
	}
	bytes := make([]byte, length)
	charsetLen := big.NewInt(int64(len(charset)))

	for i := range bytes {
		randomIndex, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", err
		}
		bytes[i] = charset[randomIndex.Int64()]
	}
	return string(bytes), nil
}
