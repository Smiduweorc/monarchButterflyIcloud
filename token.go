package monarchbutterflyicloud

import (
	"fmt"
	"strings"
)

const base62Digits = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// baseURL derives the shared-streams endpoint for an album token. The token
// encodes its server partition in base62: one digit after a leading 'A',
// otherwise the two digits after the first character.
//
// The token is placed in the URL path, so every character is checked against
// the base62 alphabet first; anything else could rewrite the path.
func baseURL(token string) (string, error) {
	for i := 0; i < len(token); i++ {
		if strings.IndexByte(base62Digits, token[i]) < 0 {
			return "", &Error{
				Code:   CodeInvalidToken,
				Op:     "token",
				Detail: fmt.Sprintf("byte %d (%q) is not in [0-9A-Za-z]", i, token[i]),
			}
		}
	}

	digits := 2
	if strings.HasPrefix(token, "A") {
		digits = 1
	}
	if len(token) < 1+digits {
		return "", &Error{
			Code:   CodeInvalidToken,
			Op:     "token",
			Detail: fmt.Sprintf("%q is too short to hold a server partition", token),
		}
	}

	partition := 0
	for i := 1; i <= digits; i++ {
		partition = partition*62 + strings.IndexByte(base62Digits, token[i])
	}

	return fmt.Sprintf("https://p%02d-sharedstreams.icloud.com/%s/sharedstreams/", partition, token), nil
}

// redirectedBaseURL is the endpoint on the host named by a 330 response.
func redirectedBaseURL(host, token string) string {
	return "https://" + host + "/" + token + "/sharedstreams/"
}
