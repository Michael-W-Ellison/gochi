package hub

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
)

// Saves are encrypted before they are uploaded, with AES-256-GCM and a key
// derived from the player's account key. The hub stores only a hash of the
// account key, so the hub's files alone can't be read: a copy of its disk
// or a backup doesn't reveal anyone's pets. (A hub that is changed to
// record account keys as they arrive could still read them; that is why
// the hub should be run by someone the players trust, and over HTTPS.)
// Each save is bound to its pet's key, so a sealed save can't be passed
// off as another pet's.

// sealedSave is an encrypted save as the hub stores it
type sealedSave struct {
	Sealed int    `json:"gochi_sealed"` // Format version: 1
	Nonce  []byte `json:"nonce"`        // Encoded as base64
	Data   []byte `json:"data"`
}

// sealVersion is the current sealed format
const sealVersion = 1

// ErrCannotOpen is returned for a save that can't be decrypted with the
// account key, for example one uploaded by another account
var ErrCannotOpen = errors.New("this save can't be decrypted with your account key")

// saveCipher returns the cipher for an account key
func saveCipher(accountKey string) (cipher.AEAD, error) {
	mac := hmac.New(sha256.New, []byte(accountKey))
	mac.Write([]byte("gochi save encryption v1"))
	block, err := aes.NewCipher(mac.Sum(nil))
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// SealSave encrypts a pet's save for upload
func SealSave(accountKey, petKey string, save []byte) (json.RawMessage, error) {
	aead, err := saveCipher(accountKey)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return json.Marshal(sealedSave{Sealed: sealVersion, Nonce: nonce, Data: aead.Seal(nil, nonce, save, []byte(petKey))})
}

// OpenSave decrypts a downloaded save. Saves uploaded before encryption
// are returned as they are.
func OpenSave(accountKey, petKey string, stored json.RawMessage) ([]byte, error) {
	var sealed sealedSave
	if err := json.Unmarshal(stored, &sealed); err != nil || sealed.Sealed == 0 {
		return stored, nil // Not sealed
	}
	if sealed.Sealed != sealVersion {
		return nil, errors.New("this save was encrypted by a newer version of Gochi")
	}
	aead, err := saveCipher(accountKey)
	if err != nil {
		return nil, err
	}
	if len(sealed.Nonce) != aead.NonceSize() {
		return nil, ErrCannotOpen
	}
	save, err := aead.Open(nil, sealed.Nonce, sealed.Data, []byte(petKey))
	if err != nil {
		return nil, ErrCannotOpen
	}
	return save, nil
}
