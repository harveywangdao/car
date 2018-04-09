package util

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"hcxy/iov/log/logger"
	"math/rand"
	"time"
)

func Crc32(data []byte) (uint32, error) {
	n := crc32.ChecksumIEEE(data)
	return n, nil
}

func DataXOR(s []byte) byte {
	var xor byte
	if len(s) > 0 {
		xor = s[0]
		for i := 1; i < len(s); i++ {
			xor ^= s[i]
		}
	}

	return xor
}

func Uint16ToByteSlice(n uint16) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, n)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	return buf.Bytes(), nil
}

func ByteSliceToUint16(s []byte) (uint16, error) {
	var n uint16
	buf := bytes.NewBuffer(s)
	err := binary.Read(buf, binary.LittleEndian, &n)
	if err != nil {
		logger.Error(err)
		return 0, err
	}

	return n, nil
}

func Uint32ToByteSlice(n uint32) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, n)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	return buf.Bytes(), nil
}

func ByteSliceToUint32(s []byte) (uint32, error) {
	var n uint32
	buf := bytes.NewBuffer(s)
	err := binary.Read(buf, binary.LittleEndian, &n)
	if err != nil {
		logger.Error(err)
		return 0, err
	}

	return n, nil
}

func Uint64ToByteSlice(n uint64) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, n)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	return buf.Bytes(), nil
}

func ByteSliceToUint64(s []byte) (uint64, error) {
	var n uint64
	buf := bytes.NewBuffer(s)
	err := binary.Read(buf, binary.LittleEndian, &n)
	if err != nil {
		logger.Error(err)
		return 0, err
	}

	return n, nil
}

func StructToByteSlice(v interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, v)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	return buf.Bytes(), nil
}

func ByteSliceToStruct(s []byte, v interface{}) error {
	buf := bytes.NewBuffer(s)
	err := binary.Read(buf, binary.LittleEndian, v)
	if err != nil {
		logger.Error(err)
		return err
	}

	return nil
}

func GenRandomString(l int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	bytes := []byte(str)
	result := []byte{}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < l; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}
	return string(result)
}

func GenRandUint32() uint32 {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return uint32(r.Uint32())
}

func Substr(str string, start int, length int) string {
	rs := []rune(str)
	rl := len(rs)
	end := 0

	if start < 0 {
		start = rl - 1 + start
	}
	end = start + length

	if start > end {
		start, end = end, start
	}

	if start < 0 {
		start = 0
	}

	if start > rl {
		start = rl
	}

	if end < 0 {
		end = 0
	}

	if end > rl {
		end = rl
	}

	return string(rs[start:end])
}
