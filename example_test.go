package golibtemplate_test

import (
	"errors"
	"fmt"

	golibtemplate "github.com/Smiduweorc/goLibTemplates"
)

// Examples are compiled and run by `go test`, and rendered on pkg.go.dev.

func ExampleGreet() {
	fmt.Println(golibtemplate.Greet("gopher"))
	fmt.Println(golibtemplate.Greet(""))
	// Output:
	// hello, gopher
	// hello, world
}

func ExampleGreetStrict() {
	_, err := golibtemplate.GreetStrict("")
	fmt.Println(errors.Is(err, golibtemplate.ErrEmptyName))
	// Output: true
}
