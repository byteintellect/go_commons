package util

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

type Signer struct {
	key string
}

func (h *Signer) GenerateSignature(payload string) string {
	signer := hmac.New(sha256.New, []byte(h.key))
	signer.Write([]byte(payload))
	return hex.EncodeToString(signer.Sum(nil))
}

func (h *Signer) ValidateSignature(payload string, eSignature string) bool {
	cSignature := h.GenerateSignature(payload)
	return cSignature == eSignature
}
