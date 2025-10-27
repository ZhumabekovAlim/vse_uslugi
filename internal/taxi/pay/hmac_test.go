package pay

import "testing"

func TestVerifyHMAC(t *testing.T) {
    body := []byte("{\"ok\":true}")
    secret := "secret"
    signature := "f6b4a2841c93f8bf2fb8f2c13d8fb0b6c8e8019f09ee405d248daa8385fad638"
    if !VerifyHMAC(body, signature, secret) {
        t.Fatal("expected signature to be valid")
    }
    if VerifyHMAC(body, "deadbeef", secret) {
        t.Fatal("unexpected valid signature")
    }
}
