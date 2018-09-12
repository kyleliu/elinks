package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"math/big"
)

// Returns the appropriate base-two padded
// bytes (assuming the underlying big representation remains b2c)
func BigBytes(i *big.Int) (buff []byte) {
	ib := i.Bytes()
	shift := 0
	shiftByte := byte(0)
	switch i.Cmp(big.NewInt(0)) {
	case 1:
		// Positive must be padded if high-bit is 1
		if ib[0]&0x80 == 0x80 {
			shift = 1
		}
	case -1:
		// Negative numbers with a leading high-bit will also need
		// to be padded, but with a single bit tagging its 'negativity'
		if ib[0]&0x80 == 0x80 {
			shift = 1
			shiftByte = 0x80

		}
	}
	buff = make([]byte, len(ib)+shift)
	if shift == 1 {
		buff[0] = shiftByte
	}
	copy(buff[shift:], ib)

	return
}

// Encode a BigInt to a base64 number;
// Also does appropriate b2 padding of the
// integer (if it's not negative)
func BigIntToB64(i *big.Int) string {
	b := BigBytes(i)
	buff := make([]byte, base64.StdEncoding.EncodedLen(len(b)))
	base64.StdEncoding.Encode(buff, b)
	return string(buff)
}

// Decode BigInt from base64 string
func B64ToBigInt(in string, b *big.Int) (err error) {
	length := base64.StdEncoding.DecodedLen(len(in))
	buff := make([]byte, length)
	n, err := base64.StdEncoding.Decode(buff, bytes.NewBufferString(in).Bytes())
	//neg := false
	if err == nil {
		buff = buff[0:n]
		//if buff[0]&0x80 == 0x80 {
		//	neg = true
		//	buff[0] &= 0x7f
		//}
		b.SetBytes(buff)
		// In case the passed in big was negative...
		//b.Abs(b)
		//if neg {
		//	b.Neg(b)
		//}
	}
	return
}

func PKCS7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext) % blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func PKCS7UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

func AesEncrypt(origData, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	origData = PKCS7Padding(origData, blockSize)
	iv := make([]byte, blockSize)
	for i:= 0; i < blockSize; i++{
		iv[i] = 0
	}
	blockMode := cipher.NewCBCEncrypter(block, iv)
	crypted := make([]byte, len(origData))
	blockMode.CryptBlocks(crypted, origData)
	return crypted, nil
}

func AesDecrypt(crypted, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	iv := make([]byte, blockSize)
	for i:= 0; i < blockSize; i++{
		iv[i] = 0
	}
	blockMode := cipher.NewCBCDecrypter(block, iv)
	origData := make([]byte, len(crypted))
	blockMode.CryptBlocks(origData, crypted)
	origData = PKCS7UnPadding(origData)
	return origData, nil
}
