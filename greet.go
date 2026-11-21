package golibtemplate

import (
	"errors"
	"strings"
)

// ErrEmptyName is returned by [GreetStrict] when the name is empty or only
// whitespace.
var ErrEmptyName = errors.New("golibtemplate: empty name")

// Greet returns a greeting for name. An empty name greets "world".
func Greet(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "world"
	}
	return "hello, " + name
}

// GreetStrict is like [Greet] but returns [ErrEmptyName] instead of falling
// back to a default name.
func GreetStrict(name string) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", ErrEmptyName
	}
	return Greet(name), nil
}
