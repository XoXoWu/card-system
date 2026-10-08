package keygen

import (
	"crypto/rand"
	"math/big"
	"strings"
)

// 剔除 I / O / 0 / 1 的 31 字符集，见 PRD 附录卡密格式规范
const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func seg(n int) string {
	b := make([]byte, n)
	max := big.NewInt(int64(len(charset)))
	for i := range b {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			panic(err)
		}
		b[i] = charset[idx.Int64()]
	}
	return string(b)
}

// Gen 生成 前缀-XXXX-XXXX-XXXX 格式卡密；prefix 为空时输出 XXXX-XXXX-XXXX
func Gen(prefix string) string {
	key := seg(4) + "-" + seg(4) + "-" + seg(4)
	if prefix != "" {
		return strings.ToUpper(prefix) + "-" + key
	}
	return key
}

// Normalize 统一输入：去空格、去连字符、转大写，用于容忍用户输入差异
func Normalize(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		if r == ' ' || r == '-' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
