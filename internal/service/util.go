package service

import (
	"crypto/rand"
	"encoding/hex"
)

// newSuffix 返回 6 字节十六进制随机串，用于生成实体 ID 后缀。
func newSuffix() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "000000000000"
	}
	return hex.EncodeToString(b)
}
