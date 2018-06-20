package aes

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
	"github.com/harveywangdao/road/log/logger"
)

const (
	ivDefValue = "0102030405060708"
)

func AesEncrypt(plaintext []byte, key []byte) ([]byte, error) {
	logger.Debug("AesEncrypt plaintext =", plaintext)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.New("invalid decrypt key")
	}

	blockSize := block.BlockSize()

	plaintext = PKCS5Padding(plaintext, blockSize)
	logger.Debug("AesEncrypt plaintext =", plaintext)

	iv := []byte(ivDefValue)

	blockMode := cipher.NewCBCEncrypter(block, iv)

	ciphertext := make([]byte, len(plaintext))

	blockMode.CryptBlocks(ciphertext, plaintext)

	logger.Debug("AesEncrypt ciphertext =", ciphertext)

	return ciphertext, nil
}

func AesDecrypt(ciphertext []byte, key []byte) ([]byte, error) {
	logger.Debug("AesDecrypt ciphertext =", ciphertext)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.New("invalid decrypt key")
	}

	blockSize := block.BlockSize() //16

	if len(ciphertext) < blockSize {
		return nil, errors.New("ciphertext too short")
	}

	iv := []byte(ivDefValue)

	if len(ciphertext)%blockSize != 0 {
		return nil, errors.New("ciphertext is not a multiple of the block size")
	}

	blockModel := cipher.NewCBCDecrypter(block, iv)

	plaintext := make([]byte, len(ciphertext))

	blockModel.CryptBlocks(plaintext, ciphertext)
	logger.Debug("AesDecrypt plaintext =", plaintext)
	plaintext, err = PKCS5UnPadding(plaintext, blockSize)
	if err != nil {
		return nil, errors.New("data error!")
	}

	logger.Debug("AesDecrypt plaintext =", plaintext)

	return plaintext, nil
}

func PKCS5Padding(src []byte, blockSize int) []byte {
	padding := blockSize - len(src)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, padtext...)
}

func PKCS5UnPadding(src []byte, blockSize int) ([]byte, error) {
	length := len(src)
	unpadding := int(src[length-1])

	logger.Debug("length =", length)
	logger.Debug("unpadding =", unpadding)

	if unpadding > blockSize {
		return nil, errors.New("Data error!")
	}

	return src[:(length - unpadding)], nil
}

func testAes() {
	c, _ := AesEncrypt([]byte("bfjkaaaaaaaabcdf"), []byte("1234567890123456"))
	p, _ := AesDecrypt(c, []byte("1234567890123456"))
	fmt.Println(string(p))
}
