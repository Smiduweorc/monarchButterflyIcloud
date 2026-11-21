package monarchbutterflyicloud

import (
	"errors"
	"net/url"
	"strings"
	"testing"
)

func TestBaseURL(t *testing.T) {
	cases := []struct {
		name, token, want string
	}{
		// 'A' prefix: one digit, token[1].
		{"an A token reads one partition digit", "A0aaaa", "https://p00-sharedstreams.icloud.com/A0aaaa/sharedstreams/"},
		{"an A token with digit 9 is zero padded", "A9", "https://p09-sharedstreams.icloud.com/A9/sharedstreams/"},
		{"an A token with digit A is partition 10", "AA", "https://p10-sharedstreams.icloud.com/AA/sharedstreams/"},
		{"an A token with digit z is partition 61", "Az", "https://p61-sharedstreams.icloud.com/Az/sharedstreams/"},
		// Any other prefix: two digits, token[1:3].
		{"a B token reads two partition digits", "B0z5qAGN1JIFd3y", "https://p61-sharedstreams.icloud.com/B0z5qAGN1JIFd3y/sharedstreams/"},
		{"a two digit partition of 00 is p00", "B00", "https://p00-sharedstreams.icloud.com/B00/sharedstreams/"},
		{"a two digit partition of 10 is p10", "B0A", "https://p10-sharedstreams.icloud.com/B0A/sharedstreams/"},
		{"the largest two digit partition is 3843", "Bzz", "https://p3843-sharedstreams.icloud.com/Bzz/sharedstreams/"},
		{"a lowercase a prefix is not the A rule", "a10", "https://p62-sharedstreams.icloud.com/a10/sharedstreams/"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := baseURL(c.token)
			if err != nil {
				t.Fatalf("baseURL(%q): %v", c.token, err)
			}
			if got != c.want {
				t.Fatalf("baseURL(%q) = %q, want %q", c.token, got, c.want)
			}
		})
	}
}

func TestBaseURLRefusals(t *testing.T) {
	cases := []struct{ name, token string }{
		{"an empty token is refused", ""},
		{"an A with no digit is refused", "A"},
		{"a B token with one digit is refused", "B0"},
		{"a single character is refused", "B"},
		{"a slash cannot reach the URL path", "B0z/../x"},
		{"a query character is refused", "B0z?x=1"},
		{"a fragment character is refused", "B0z#x"},
		{"a percent escape is refused", "B0z%2F"},
		{"a space is refused", "B0z 5"},
		{"a non ASCII letter is refused", "B0z\u00e9"},
		{"a NUL byte is refused", "B0z\x00"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := baseURL(c.token)
			var e *Error
			if !errors.As(err, &e) || e.Code != CodeInvalidToken || e.Op != "token" {
				t.Fatalf("baseURL(%q) error = %v, want CodeInvalidToken", c.token, err)
			}
		})
	}
}

func FuzzBaseURL(f *testing.F) {
	for _, seed := range []string{"", "A", "A0", "B0z5qAGN1JIFd3y", "Bzz", "B0/", "\xff\xfe"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, token string) {
		got, err := baseURL(token)
		if err != nil {
			var e *Error
			if !errors.As(err, &e) || e.Code != CodeInvalidToken {
				t.Fatalf("baseURL(%q) returned a non-refusal error %v", token, err)
			}
			return
		}
		u, perr := url.Parse(got)
		if perr != nil {
			t.Fatalf("baseURL(%q) = %q, which does not parse: %v", token, got, perr)
		}
		if u.Scheme != "https" || !strings.HasSuffix(u.Host, "-sharedstreams.icloud.com") || u.User != nil {
			t.Fatalf("baseURL(%q) = %q points somewhere other than iCloud", token, got)
		}
		if u.Path != "/"+token+"/sharedstreams/" || u.RawQuery != "" || u.Fragment != "" {
			t.Fatalf("baseURL(%q) = %q changed the path", token, got)
		}
	})
}
