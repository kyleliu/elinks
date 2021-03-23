package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"math/big"
	"time"
)

var LogQuiet = false
var LogQuietDump = ""

func LogEnable(enable bool) {
	if enable {
		LogQuiet = false
		if LogQuietDump != "" {
			fmt.Print(LogQuietDump)
			LogQuietDump = ""
		}
	} else {
		LogQuiet = true
		LogQuietDump = ""
	}
}

func LogPrintln(a ...interface{}) {
	var buf bytes.Buffer
	for argNum, arg := range a {
		if argNum > 0 {
			buf.WriteByte(' ')
		}
		buf.WriteString(fmt.Sprint(arg))
	}

	if LogQuiet {
		LogQuietDump += fmt.Sprintln(time.Now().Format("2006-01-02 15:04:05"), buf.String())
	} else {
		fmt.Println(time.Now().Format("2006-01-02 15:04:05"), buf.String())
	}
}

func LogPrintf(format string, a ...interface{}) {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf(format, a...))

	if LogQuiet {
		LogQuietDump += fmt.Sprintln(time.Now().Format("2006-01-02 15:04:05"), buf.String())
	} else {
		fmt.Print(time.Now().Format("2006-01-02 15:04:05"), buf.String())
	}
}

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
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func PKCS7UnPadding(origData []byte, blockSize int) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])

	// 若padding长度不在0到blockSize范围内
	// 说明该段密文根本没有padding过
	if unpadding <= 0 || unpadding >= blockSize {
		return origData
	}

	// padding字节一定都相等
	padchar := origData[length-1]
	for i := 1; i < unpadding; i++ {
		if padchar != origData[length-1 - i] {
			return origData
		}
	}

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
	for i := 0; i < blockSize; i++ {
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
	for i := 0; i < blockSize; i++ {
		iv[i] = 0
	}
	blockMode := cipher.NewCBCDecrypter(block, iv)
	origData := make([]byte, len(crypted))
	blockMode.CryptBlocks(origData, crypted)
	origData = PKCS7UnPadding(origData, blockSize)
	return origData, nil
}

func FW(s string, l int) string {
	w := 0
	for _, c := range []rune(s) {
		if IsFullwidth(c) {
			w += 2
		} else {
			w += 1
		}
	}
	for w < l {
		s += " "
		w++
	}

	return s
}
