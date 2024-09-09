package sha256

import (
	"crypto/sha256"
	"encoding/hex"
)

func String(data string) string {
	passwordHash := sha256.New()
	passwordHash.Write([]byte(data))

	return hex.EncodeToString(passwordHash.Sum(nil))
}
