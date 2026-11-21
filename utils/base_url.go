// Package utils is a collection of helpers that will be used by other parts of the library.
package utils

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
