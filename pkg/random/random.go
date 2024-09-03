package random

import (
	"fmt"
	"time"

	"math/rand"
)

const (
	LatinLower uint8 = 1 << iota
	LatinUpper
	Numbers
)

// String generates a random string of a given length, contains
// latin lowercase, uppercase characters and numbers
func String(length int) string {
	return StringWith(length, LatinLower|LatinUpper|Numbers)
}

// StringWith generates a random string with given options
func StringWith(length int, opts uint8) string {
	if opts == 0 {
		return ""
	}

	var charset string
	if opts&LatinLower == LatinLower {
		charset = fmt.Sprintf("%s%s", charset, "abcdefghijklmnopqrstuvwxyz")
	}
	if opts&LatinUpper == LatinUpper {
		charset = fmt.Sprintf("%s%s", charset, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	}
	if opts&Numbers == Numbers {
		charset = fmt.Sprintf("%s%s", charset, "0123456789")
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[r.Intn(len(charset))]
	}
	return string(b)
}
