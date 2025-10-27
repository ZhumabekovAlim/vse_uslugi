package pay

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
)

// VerifyHMAC validates a signature using HMAC-SHA256.
func VerifyHMAC(body []byte, signature, secret string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(body)
    expected := mac.Sum(nil)

    sigBytes, err := hex.DecodeString(signature)
    if err != nil {
        return false
    }
    return hmac.Equal(expected, sigBytes)
}
