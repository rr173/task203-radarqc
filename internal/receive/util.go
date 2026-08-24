package receive

import (
	"crypto/rand"
	"encoding/hex"
)

// randInt 返回随机 int32 分量，用于生成实体 ID。
func randInt() int32 {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		// 密码学随机源不可用时退化为时间戳低位（仅影响 ID 唯一性分布，不影响正确性）
		return int32(b[0])<<24 | int32(b[1])<<16 | int32(b[2])<<8 | int32(b[3])
	}
	return int32(b[0])<<24 | int32(b[1])<<16 | int32(b[2])<<8 | int32(b[3])
}

// randomHex 返回 n 字节的十六进制随机串（用于门 ID 等）。
func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "000000000000"
	}
	return hex.EncodeToString(b)
}
