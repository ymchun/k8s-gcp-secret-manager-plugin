package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"

	"github.com/secure-io/sio-go/sioutil"
	"golang.org/x/crypto/chacha20poly1305"
)

type SecretPayload struct {
	Salt       []byte `json:"s"`
	Nonce      []byte `json:"n"`
	Ciphertext []byte `json:"c"`
}

type Secret struct {
	Key []byte // the master key a.k.a key encrypt key
}

func generateDataKey(key, salt []byte) ([]byte, error) {
	hash := hmac.New(sha256.New, key)
	_, err := hash.Write(salt)

	if err != nil {
		return nil, err
	}

	return hash.Sum(nil), nil
}

func getAEAD(key []byte) (cipher.AEAD, error) {
	var aead cipher.AEAD
	var err error

	if sioutil.NativeAES() {
		block, err := aes.NewCipher(key)

		if err != nil {
			return nil, err
		}

		aead, err = cipher.NewGCM(block)

		if err != nil {
			return nil, err
		}
	} else {
		aead, err = chacha20poly1305.New(key)

		if err != nil {
			return nil, err
		}
	}

	return aead, nil
}

func (s *Secret) Destroy() {
	s.Key, _ = sioutil.Random(len(s.Key))
}

func (s *Secret) Encrypt(plaintext []byte) ([]byte, error) {
	salt, err := sioutil.Random(16)

	if err != nil {
		return nil, err
	}

	dataKey, err := generateDataKey(s.Key, salt)

	if err != nil {
		return nil, err
	}

	aead, err := getAEAD(dataKey)

	if err != nil {
		return nil, err
	}

	nonce, err := sioutil.Random(aead.NonceSize())

	if err != nil {
		return nil, err
	}

	ciphertext := aead.Seal(nil, nonce, plaintext, nil)

	payload, err := json.Marshal(SecretPayload{
		Salt:       salt,
		Nonce:      nonce,
		Ciphertext: ciphertext,
	})

	if err != nil {
		return nil, err
	}

	return payload, nil
}

func (s *Secret) Decrypt(ciphertext []byte) ([]byte, error) {
	var secretPayload SecretPayload
	err := json.Unmarshal(ciphertext, &secretPayload)

	if err != nil {
		return nil, err
	}

	dataKey, err := generateDataKey(s.Key, secretPayload.Salt)

	if err != nil {
		return nil, err
	}

	aead, err := getAEAD(dataKey)

	if err != nil {
		return nil, err
	}

	plaintext, err := aead.Open(nil, secretPayload.Nonce, secretPayload.Ciphertext, nil)

	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
