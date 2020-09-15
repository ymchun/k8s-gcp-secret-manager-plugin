package secret

import (
	"bytes"
	"testing"

	"github.com/secure-io/sio-go/sioutil"
)

func TestGenerateDataKey(t *testing.T) {
	testMasterKey, err := sioutil.Random(16)

	if err != nil {
		t.Errorf("Failed to generate master key: %v", err)
	}

	salt, err := sioutil.Random(16)

	if err != nil {
		t.Errorf("Failed to generate salt: %v", err)
	}

	// make sure resulting key are the same if input is the same
	firstDEK, err := generateDataKey(testMasterKey, salt)

	if err != nil {
		t.Errorf("Failed to generate first DEK: %v", err)
	}

	secondDEK, err := generateDataKey(testMasterKey, salt)

	if err != nil {
		t.Errorf("Failed to generate second DEK: %v", err)
	}

	if result := bytes.Compare(firstDEK, secondDEK); result != 0 {
		t.Error("Data encryption key not match")
	}
}

func TestDestroy(t *testing.T) {
	testKey := []byte("testkey")
	secret := Secret{
		Key: testKey,
	}
	secret.Destroy()

	if result := bytes.Compare(testKey, secret.Key); result == 0 {
		t.Error("Secret destroy has no effect")
	}
}

func TestEncryptDecrypt(t *testing.T) {
	payload := []byte("hello world")
	secret := Secret{
		Key: []byte("testkey"),
	}

	ciphertext, err := secret.Encrypt(payload)

	if err != nil {
		t.Errorf("Failed to encrypt plaintext: %v", err)
	}

	plaintext, err := secret.Decrypt(ciphertext)

	if err != nil {
		t.Errorf("Failed to decrypt ciphertext: %v", err)
	}

	if result := bytes.Compare(payload, plaintext); result != 0 {
		t.Error("Payload not matched when encrypt then decrypt")
	}
}
