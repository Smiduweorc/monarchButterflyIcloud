package golibtemplate

import (
	"errors"
	"testing"
)

func TestGreet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"named", "gopher", "hello, gopher"},
		{"trimmed", "  gopher\t", "hello, gopher"},
		{"empty", "", "hello, world"},
		{"whitespace", "   ", "hello, world"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Greet(tt.in); got != tt.want {
				t.Errorf("Greet(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestGreetStrict(t *testing.T) {
	t.Parallel()

	if _, err := GreetStrict(" "); !errors.Is(err, ErrEmptyName) {
		t.Fatalf("GreetStrict(\" \") error = %v, want %v", err, ErrEmptyName)
	}
	got, err := GreetStrict("gopher")
	if err != nil {
		t.Fatalf("GreetStrict(\"gopher\") unexpected error: %v", err)
	}
	if want := "hello, gopher"; got != want {
		t.Errorf("GreetStrict(\"gopher\") = %q, want %q", got, want)
	}
}

func FuzzGreet(f *testing.F) {
	for _, seed := range []string{"", "gopher", " \t\n", "héllo"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, name string) {
		if got := Greet(name); len(got) < len("hello, ") {
			t.Errorf("Greet(%q) = %q, too short", name, got)
		}
	})
}

func BenchmarkGreet(b *testing.B) {
	for b.Loop() {
		_ = Greet("gopher")
	}
}
