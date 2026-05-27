package jet

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"github.com/cespare/xxhash/v2"
	"io"
)

const (
	ErrCodeCryptoEncrypt = "CRP-001"
	ErrCodeCryptoDecrypt = "CRP-002"
)

var (
	ErrCryptoEncrypt = func(ctx context.Context, cause error) error {
		return NewAppErrBuilder(ErrCodeCryptoEncrypt, "encryption error").C(ctx).Wrap(cause).Err()
	}
	ErrCryptoDecrypt = func(ctx context.Context, cause error) error {
		return NewAppErrBuilder(ErrCodeCryptoDecrypt, "decryption error").C(ctx).Wrap(cause).Err()
	}
)

func EncryptString(ctx context.Context, key, val string) (string, error) {
	byteMsg := []byte(val)
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", ErrCryptoEncrypt(ctx, err)
	}

	cipherText := make([]byte, aes.BlockSize+len(byteMsg))
	iv := cipherText[:aes.BlockSize]
	if _, err = io.ReadFull(rand.Reader, iv); err != nil {
		return "", ErrCryptoEncrypt(ctx, err)
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(cipherText[aes.BlockSize:], byteMsg)

	return base64.StdEncoding.EncodeToString(cipherText), nil
}

func DecryptString(ctx context.Context, key, val string) (string, error) {
	cipherText, err := base64.StdEncoding.DecodeString(val)
	if err != nil {
		return "", ErrCryptoDecrypt(ctx, err)
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", ErrCryptoDecrypt(ctx, err)
	}

	if len(cipherText) < aes.BlockSize {
		return "", ErrCryptoDecrypt(ctx, err)
	}

	iv := cipherText[:aes.BlockSize]
	cipherText = cipherText[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(cipherText, cipherText)

	return string(cipherText), nil
}

func HashObj(obj any) uint64 {

	if obj == nil {
		return 0
	}

	// marshal obj
	data, _ := json.Marshal(obj)
	if len(data) == 0 {
		return 0
	}

	return xxhash.Sum64(data)

}
