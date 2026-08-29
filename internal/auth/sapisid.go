package auth

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"time"
)

// GenerateSAPISIDHash computes the SHA-1 SAPISIDHASH header value required for authenticated Google APIs.
func GenerateSAPISIDHash(sapisid string) string {
	if sapisid == "" {
		return ""
	}
	ts := time.Now().Unix()
	payload := fmt.Sprintf("%d %s https://gemini.google.com", ts, sapisid)
	hash := sha1.Sum([]byte(payload))
	digest := hex.EncodeToString(hash[:])

	return fmt.Sprintf("SAPISIDHASH %d_%s", ts, digest)
}
