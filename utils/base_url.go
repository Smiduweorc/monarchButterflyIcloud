// Package utils is a collection of helpers that will be used by other parts of the library.
package utils

import (
	"strconv"
)

// Base62 shape for shortGUID.
type Base62 struct {
	index map[byte]int
}

// NewBase62 is a function that returns the pointer to a Base62 struct that is built in a map for quick lookups.
func NewBase62() *Base62 {
	characters := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	index := make(map[byte]int, len(characters))

	for i := 0; i < len(characters); i++ {
		index[characters[i]] = i
	}

	return &Base62{index: index}
}

// Decode added a decode function for a Base62.
func (b *Base62) Decode(s string) int {
	result := 0
	for i := 0; i < len(s); i++ {
		result = result*62 + b.index[s[i]]
	}
	return result
}

// GetBaseURL constructs the initial iCloud base URL based on the album token.
func GetBaseURL(token string) string {
	b62 := NewBase62()

	t := token[0]
	n := 0
	if t == 'A' {
		n = b62.Decode(string(token[1]))
	} else {
		n = b62.Decode(token[1:3])
	}

	serverPartition := n

	baseURL := "https://p"

	if serverPartition < 10 {
		baseURL += "0" + strconv.Itoa(serverPartition)
	} else {
		baseURL += strconv.Itoa(serverPartition)
	}
	baseURL += "-sharedstreams.icloud.com"
	baseURL += "/"
	baseURL += token // Using original token here as per TS, not 'r'
	baseURL += "/sharedstreams/"

	return baseURL
}
