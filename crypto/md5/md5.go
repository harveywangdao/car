package md5

import (
	"crypto/md5"
)

func GenMd5(data []byte) []byte {
	m := md5.New()
	m.Write(data)
	return m.Sum(nil)
}
