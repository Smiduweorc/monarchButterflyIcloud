package monarchbutterflyicloud

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
)

// assetURLBatch is how many photo GUIDs go into one /webasseturls request.
// 25 is the batch size used by the icloud-shared-album TypeScript library
// this package's request flow follows.
const assetURLBatch = 25

// statusMoved is the status iCloud answers with when an album lives on a
// different partition than its token suggests. net/http does not treat it
// as a redirect, so the response comes back to us unfollowed.
const statusMoved = 330

// Client reads public iCloud Shared Albums.
type Client struct {
	http *http.Client
}

// NewClient returns a Client that sends every request through hc.
//
// hc is required and its Timeout must be positive. It bounds each request,
// including reading the response body, so set it longer than the /webstream
// download takes for your largest album on your slowest network. Pass a
// context to Album for a deadline over the whole call.
func NewClient(hc *http.Client) (*Client, error) {
	if hc == nil {
		return nil, &Error{Code: CodeInvalidClient, Op: "NewClient", Detail: "hc is required; pass an *http.Client with a Timeout"}
	}
	if hc.Timeout <= 0 {
		return nil, &Error{Code: CodeInvalidClient, Op: "NewClient", Detail: "hc.Timeout must be positive; without one a stalled response hangs forever"}
	}
	return &Client{http: hc}, nil
}

// Album fetches the album behind token, the part of a share link after the
// '#' (https://www.icloud.com/sharedalbum/#B0z5qAGN1JIFd3y).
//
// It makes one /webstream request (two if iCloud redirects the album to
// another partition) and one /webasseturls request per 25 photos.
//
// Refusals are *Error. Errors from the http.Client, including the context
// being cancelled, are returned as the client produced them.
func (c *Client) Album(ctx context.Context, token string) (*Album, error) {
	base, err := baseURL(token)
	if err != nil {
		return nil, err
	}

	var stream wireWebstream
	base, err = c.webstream(ctx, base, token, &stream)
	if err != nil {
		return nil, err
	}

	urls := make(map[string]string)
	for batch := range slices.Chunk(stream.Photos, assetURLBatch) {
		if err := c.assetURLs(ctx, base, batch, urls); err != nil {
			return nil, err
		}
	}

	album := &Album{
		Name:           stream.StreamName,
		OwnerFirstName: stream.UserFirstName,
		OwnerLastName:  stream.UserLastName,
		Photos:         make([]Photo, 0, len(stream.Photos)),
	}
	for _, w := range stream.Photos {
		photo, err := toPhoto(w, urls)
		if err != nil {
			return nil, err
		}
		album.Photos = append(album.Photos, photo)
	}
	return album, nil
}

// webstream decodes the album's /webstream response into stream and returns
// the base URL that answered, following at most one 330.
func (c *Client) webstream(ctx context.Context, base, token string, stream *wireWebstream) (string, error) {
	body := []byte(`{"streamCtag":null}`)

	resp, err := c.post(ctx, base+"webstream", body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode == statusMoved {
		host, err := movedHost(resp)
		if err != nil {
			return "", err
		}
		base = redirectedBaseURL(host, token)
		if resp, err = c.post(ctx, base+"webstream", body); err != nil {
			return "", err
		}
		if resp.StatusCode == statusMoved {
			_ = resp.Body.Close()
			return "", &Error{Code: CodeBadRedirect, Op: "webstream", Status: statusMoved, Detail: "redirected a second time, from " + host}
		}
	}
	return base, decode(resp, "webstream", stream)
}

func (c *Client) assetURLs(ctx context.Context, base string, photos []wirePhoto, urls map[string]string) error {
	guids := make([]string, len(photos))
	for i, p := range photos {
		guids[i] = p.PhotoGUID
	}
	body, err := json.Marshal(map[string][]string{"photoGuids": guids})
	if err != nil {
		return err
	}

	resp, err := c.post(ctx, base+"webasseturls", body)
	if err != nil {
		return err
	}
	var assets wireAssetURLs
	if err := decode(resp, "webasseturls", &assets); err != nil {
		return err
	}
	for checksum, item := range assets.Items {
		urls[checksum] = "https://" + item.URLLocation + item.URLPath
	}
	return nil
}

// movedHost reads the new host from a 330 response and closes it. Existing
// clients disagree on whether the host is in the JSON body or in a header of
// the same name, so both are read, body first.
func movedHost(resp *http.Response) (string, error) {
	var moved struct {
		Host string `json:"X-Apple-MMe-Host"`
	}
	err := json.NewDecoder(resp.Body).Decode(&moved)
	_ = resp.Body.Close()
	host := moved.Host
	if host == "" {
		host = resp.Header.Get("X-Apple-MMe-Host")
	}

	// The host is spliced into a URL, so anything that could change the
	// scheme, the path or the credentials is refused.
	if host == "" || strings.ContainsAny(host, "/\\@?#: \t\r\n") {
		e := &Error{Code: CodeBadRedirect, Op: "webstream", Status: statusMoved, Detail: "no usable X-Apple-MMe-Host"}
		if host != "" {
			e.Detail += ": " + strconv.Quote(host)
		} else if err != nil {
			e.Err = err
		}
		return "", e
	}
	return host, nil
}

func (c *Client) post(ctx context.Context, url string, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	// The endpoints are undocumented. These are the headers the icloud.com
	// web viewer sends, and none of them is known to be optional.
	req.Header.Set("Origin", "https://www.icloud.com")
	req.Header.Set("Referer", "https://www.icloud.com/sharedalbum/")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.8")
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/56.0.2924.87 Safari/537.36")
	return c.http.Do(req)
}

// decode reads a 2xx JSON response into v and closes the body.
func decode(resp *http.Response, op string, v any) (err error) {
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return &Error{Code: CodeUnexpectedStatus, Op: op, Status: resp.StatusCode}
	}
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		var syntax *json.SyntaxError
		var typ *json.UnmarshalTypeError
		if errors.As(err, &syntax) || errors.As(err, &typ) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
			return &Error{Code: CodeMalformedResponse, Op: op, Err: err}
		}
		// A read error from the connection itself, such as the client
		// Timeout firing mid-body, is the transport's error.
		return err
	}
	return nil
}
