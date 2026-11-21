package monarchbutterflyicloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

const testAlbum = "B0z5qAGN1JIFd3y"

// request is what the fake iCloud saw: the host the client meant to reach,
// before the transport pointed it at the test server.
type request struct {
	host, path, body string
	header           http.Header
}

type fake struct {
	mu       sync.Mutex
	requests []request
}

func (f *fake) seen() []request {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]request(nil), f.requests...)
}

// rewrite sends every request to target and keeps the original host in the
// Host header, so the handler can see which iCloud partition was asked for.
type rewrite struct{ target *url.URL }

func (rt rewrite) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())
	r.Host = req.URL.Host
	r.URL.Scheme = rt.target.Scheme
	r.URL.Host = rt.target.Host
	return http.DefaultTransport.RoundTrip(r)
}

// build starts a fake iCloud served by handle and returns a Client wired to it.
func build(t *testing.T, handle http.HandlerFunc) (*Client, *fake) {
	t.Helper()
	f := &fake{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		f.requests = append(f.requests, request{host: r.Host, path: r.URL.Path, body: string(body), header: r.Header.Clone()})
		f.mu.Unlock()
		handle(w, r)
	}))
	t.Cleanup(srv.Close)
	target, _ := url.Parse(srv.URL)
	c, err := NewClient(&http.Client{Timeout: time.Minute, Transport: rewrite{target}})
	if err != nil {
		t.Fatal(err)
	}
	return c, f
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// photo is a /webstream photo as iCloud sends it, numbers as strings.
func photo(guid string, derivatives map[string]any) map[string]any {
	return map[string]any{
		"photoGuid":            guid,
		"batchGuid":            "batch-" + guid,
		"caption":              "caption " + guid,
		"contributorFirstName": "Ada",
		"contributorLastName":  "Lovelace",
		"contributorFullName":  "Ada Lovelace",
		"mediaAssetType":       nil,
		"width":                "4032",
		"height":               "3024",
		"dateCreated":          "2024-03-09T18:43:02Z",
		"batchDateCreated":     "2024-03-10T08:00:00Z",
		"derivatives":          derivatives,
	}
}

func derivative(checksum, height string) map[string]any {
	return map[string]any{"checksum": checksum, "fileSize": "123456", "width": "1000", "height": height}
}

func stream(photos ...map[string]any) map[string]any {
	if photos == nil {
		photos = []map[string]any{}
	}
	return map[string]any{
		"streamName":    "Garden",
		"userFirstName": "Grace",
		"userLastName":  "Hopper",
		"streamCtag":    "ctag",
		"itemsReturned": fmt.Sprint(len(photos)),
		"locations":     map[string]any{},
		"photos":        photos,
	}
}

// assets answers /webasseturls with a URL for every checksum in checksums.
func assets(checksums ...string) map[string]any {
	items := map[string]any{}
	for _, c := range checksums {
		items[c] = map[string]any{"url_location": "cvws.icloud-content.com", "url_path": "/S/" + c + "/file.jpg"}
	}
	return map[string]any{"items": items, "locations": map[string]any{}}
}

// serve answers /webstream with ws and /webasseturls with as.
func serve(ws, as any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/webstream"):
			writeJSON(w, ws)
		case strings.HasSuffix(r.URL.Path, "/webasseturls"):
			writeJSON(w, as)
		default:
			http.NotFound(w, r)
		}
	}
}

// wantCode returns a copy of the *Error so errcheck does not read it as an
// unchecked error.
func wantCode(t *testing.T, err error, code Code) Error {
	t.Helper()
	var e *Error
	if !errors.As(err, &e) {
		t.Fatalf("error = %v (%T), want *Error with code %s", err, err, code)
	}
	if e.Code != code {
		t.Fatalf("error code = %s, want %s (%v)", e.Code, code, err)
	}
	return *e
}

func TestNewClient(t *testing.T) {
	t.Run("a nil http client is refused", func(t *testing.T) {
		_, err := NewClient(nil)
		wantCode(t, err, CodeInvalidClient)
	})
	t.Run("a client with no timeout is refused", func(t *testing.T) {
		_, err := NewClient(&http.Client{})
		wantCode(t, err, CodeInvalidClient)
	})
	t.Run("a negative timeout is refused", func(t *testing.T) {
		_, err := NewClient(&http.Client{Timeout: -time.Second})
		wantCode(t, err, CodeInvalidClient)
	})
	t.Run("http.DefaultClient is refused because it has no timeout", func(t *testing.T) {
		_, err := NewClient(http.DefaultClient)
		wantCode(t, err, CodeInvalidClient)
	})
	t.Run("a one nanosecond timeout is accepted", func(t *testing.T) {
		if _, err := NewClient(&http.Client{Timeout: time.Nanosecond}); err != nil {
			t.Fatal(err)
		}
	})
}

func TestAlbum(t *testing.T) {
	video := photo("p2", map[string]any{"PosterFrame": derivative("c3", "1080")})
	video["mediaAssetType"] = "video"
	video["dateCreated"] = "2024-03-09T18:43:02+0800"

	c, f := build(t, serve(
		stream(
			photo("p1", map[string]any{
				"2342": derivative("c2", "2048"),
				"342":  derivative("c1", "342"),
				"1":    derivative("no-url", "100"),
			}),
			video,
		),
		assets("c1", "c2", "c3"),
	))

	album, err := c.Album(context.Background(), testAlbum)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("album metadata comes from the stream", func(t *testing.T) {
		if album.Name != "Garden" || album.OwnerFirstName != "Grace" || album.OwnerLastName != "Hopper" {
			t.Fatalf("album = %+v", album)
		}
	})

	t.Run("photos keep the order iCloud lists them in", func(t *testing.T) {
		if len(album.Photos) != 2 || album.Photos[0].GUID != "p1" || album.Photos[1].GUID != "p2" {
			t.Fatalf("photos = %+v", album.Photos)
		}
	})

	t.Run("string numbers and dates are parsed", func(t *testing.T) {
		p := album.Photos[0]
		if p.Width != 4032 || p.Height != 3024 {
			t.Fatalf("size = %dx%d", p.Width, p.Height)
		}
		if want := time.Date(2024, 3, 9, 18, 43, 2, 0, time.UTC); !p.Created.Equal(want) {
			t.Fatalf("Created = %v, want %v", p.Created, want)
		}
		if want := time.Date(2024, 3, 10, 8, 0, 0, 0, time.UTC); !p.BatchCreated.Equal(want) {
			t.Fatalf("BatchCreated = %v, want %v", p.BatchCreated, want)
		}
		if p.BatchGUID != "batch-p1" || p.Caption != "caption p1" ||
			p.ContributorFirstName != "Ada" || p.ContributorLastName != "Lovelace" || p.ContributorFullName != "Ada Lovelace" {
			t.Fatalf("photo = %+v", p)
		}
	})

	t.Run("a date with a colonless offset is parsed", func(t *testing.T) {
		if want := time.Date(2024, 3, 9, 10, 43, 2, 0, time.UTC); !album.Photos[1].Created.Equal(want) {
			t.Fatalf("Created = %v, want %v", album.Photos[1].Created, want)
		}
	})

	t.Run("a null mediaAssetType is empty and a video says video", func(t *testing.T) {
		if album.Photos[0].MediaAssetType != "" || album.Photos[1].MediaAssetType != "video" {
			t.Fatalf("types = %q, %q", album.Photos[0].MediaAssetType, album.Photos[1].MediaAssetType)
		}
	})

	t.Run("derivatives are sorted by height and carry their joined URL", func(t *testing.T) {
		want := []Derivative{
			{Key: "342", Checksum: "c1", FileSize: 123456, Width: 1000, Height: 342, URL: "https://cvws.icloud-content.com/S/c1/file.jpg"},
			{Key: "2342", Checksum: "c2", FileSize: 123456, Width: 1000, Height: 2048, URL: "https://cvws.icloud-content.com/S/c2/file.jpg"},
		}
		got := album.Photos[0].Derivatives
		if len(got) != len(want) {
			t.Fatalf("derivatives = %+v", got)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("derivative %d = %+v, want %+v", i, got[i], want[i])
			}
		}
	})

	t.Run("a derivative iCloud gave no URL for is left out", func(t *testing.T) {
		for _, d := range album.Photos[0].Derivatives {
			if d.Checksum == "no-url" {
				t.Fatalf("derivative without a URL kept: %+v", d)
			}
		}
	})

	reqs := f.seen()
	t.Run("the token's partition picks the host", func(t *testing.T) {
		if len(reqs) != 2 {
			t.Fatalf("requests = %+v", reqs)
		}
		for _, r := range reqs {
			if r.host != "p61-sharedstreams.icloud.com" {
				t.Fatalf("host = %q", r.host)
			}
		}
		if reqs[0].path != "/"+testAlbum+"/sharedstreams/webstream" || reqs[1].path != "/"+testAlbum+"/sharedstreams/webasseturls" {
			t.Fatalf("paths = %q, %q", reqs[0].path, reqs[1].path)
		}
	})

	t.Run("webstream asks for the whole stream", func(t *testing.T) {
		if reqs[0].body != `{"streamCtag":null}` {
			t.Fatalf("body = %q", reqs[0].body)
		}
	})

	t.Run("webasseturls asks for every photo in order", func(t *testing.T) {
		if reqs[1].body != `{"photoGuids":["p1","p2"]}` {
			t.Fatalf("body = %q", reqs[1].body)
		}
	})

	t.Run("requests carry the icloud.com web viewer headers", func(t *testing.T) {
		want := map[string]string{
			"Origin":       "https://www.icloud.com",
			"Referer":      "https://www.icloud.com/sharedalbum/",
			"Content-Type": "text/plain",
			"Accept":       "*/*",
		}
		for _, r := range reqs {
			for k, v := range want {
				if got := r.header.Get(k); got != v {
					t.Fatalf("%s %s = %q, want %q", r.path, k, got, v)
				}
			}
			if !strings.HasPrefix(r.header.Get("User-Agent"), "Mozilla/5.0 ") {
				t.Fatalf("User-Agent = %q", r.header.Get("User-Agent"))
			}
		}
	})
}

func TestAlbumDerivativeHeightTie(t *testing.T) {
	// Six renditions of one height: without the key tie-break, map order
	// would put them in the right order about once in 720 runs.
	keys := []string{"a", "b", "c", "d", "e", "f"}
	derivs := map[string]any{}
	var checksums []string
	for _, k := range keys {
		derivs[k] = derivative("c"+k, "720")
		checksums = append(checksums, "c"+k)
	}
	c, _ := build(t, serve(stream(photo("p1", derivs)), assets(checksums...)))
	album, err := c.Album(context.Background(), testAlbum)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, d := range album.Photos[0].Derivatives {
		got = append(got, d.Key)
	}
	if strings.Join(got, "") != "abcdef" {
		t.Fatalf("renditions of the same height are ordered by key, got %v", got)
	}
}

func TestAlbumBatching(t *testing.T) {
	cases := []struct {
		photos  int
		batches []int
	}{
		{0, nil},
		{1, []int{1}},
		{24, []int{24}},
		{25, []int{25}},
		{26, []int{25, 1}},
		{50, []int{25, 25}},
		{51, []int{25, 25, 1}},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%d photos make %d webasseturls requests", tc.photos, len(tc.batches)), func(t *testing.T) {
			var photos []map[string]any
			var guids []string
			for i := range tc.photos {
				guid := fmt.Sprintf("g%02d", i)
				guids = append(guids, guid)
				photos = append(photos, photo(guid, map[string]any{"1": derivative("c"+guid, "100")}))
			}
			c, f := build(t, serve(stream(photos...), assets()))
			album, err := c.Album(context.Background(), testAlbum)
			if err != nil {
				t.Fatal(err)
			}
			if len(album.Photos) != tc.photos {
				t.Fatalf("photos = %d", len(album.Photos))
			}

			var batches [][]string
			for _, r := range f.seen() {
				if strings.HasSuffix(r.path, "/webasseturls") {
					var b struct{ PhotoGuids []string }
					if err := json.Unmarshal([]byte(r.body), &b); err != nil {
						t.Fatal(err)
					}
					batches = append(batches, b.PhotoGuids)
				}
			}
			if len(batches) != len(tc.batches) {
				t.Fatalf("batches = %v", batches)
			}
			var all []string
			for i, b := range batches {
				if len(b) != tc.batches[i] {
					t.Fatalf("batch %d has %d guids, want %d", i, len(b), tc.batches[i])
				}
				all = append(all, b...)
			}
			if strings.Join(all, ",") != strings.Join(guids, ",") {
				t.Fatalf("guids sent = %v, want %v", all, guids)
			}
		})
	}
}

// moved answers the first /webstream on the token's own partition with a 330
// built by respond, and everything else with a one-photo album.
func moved(respond func(w http.ResponseWriter)) http.HandlerFunc {
	ok := serve(stream(photo("p1", map[string]any{"1": derivative("c1", "100")})), assets("c1"))
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Host == "p61-sharedstreams.icloud.com" {
			respond(w)
			return
		}
		ok(w, r)
	}
}

func TestAlbumRedirect(t *testing.T) {
	t.Run("a 330 with the host in the body moves every later request there", func(t *testing.T) {
		c, f := build(t, moved(func(w http.ResponseWriter) {
			w.WriteHeader(330)
			_, _ = io.WriteString(w, `{"X-Apple-MMe-Host":"p42-sharedstreams.icloud.com"}`)
		}))
		if _, err := c.Album(context.Background(), testAlbum); err != nil {
			t.Fatal(err)
		}
		reqs := f.seen()
		if len(reqs) != 3 {
			t.Fatalf("requests = %+v", reqs)
		}
		for _, r := range reqs[1:] {
			if r.host != "p42-sharedstreams.icloud.com" || !strings.HasPrefix(r.path, "/"+testAlbum+"/sharedstreams/") {
				t.Fatalf("request went to %s%s", r.host, r.path)
			}
		}
	})

	t.Run("a 330 with the host only in a header is followed", func(t *testing.T) {
		c, f := build(t, moved(func(w http.ResponseWriter) {
			w.Header().Set("X-Apple-MMe-Host", "p07-sharedstreams.icloud.com")
			w.WriteHeader(330)
		}))
		if _, err := c.Album(context.Background(), testAlbum); err != nil {
			t.Fatal(err)
		}
		if got := f.seen()[1].host; got != "p07-sharedstreams.icloud.com" {
			t.Fatalf("host = %q", got)
		}
	})

	t.Run("the body host wins over the header host", func(t *testing.T) {
		c, f := build(t, moved(func(w http.ResponseWriter) {
			w.Header().Set("X-Apple-MMe-Host", "p07-sharedstreams.icloud.com")
			w.WriteHeader(330)
			_, _ = io.WriteString(w, `{"X-Apple-MMe-Host":"p42-sharedstreams.icloud.com"}`)
		}))
		if _, err := c.Album(context.Background(), testAlbum); err != nil {
			t.Fatal(err)
		}
		if got := f.seen()[1].host; got != "p42-sharedstreams.icloud.com" {
			t.Fatalf("host = %q", got)
		}
	})

	t.Run("a 330 with no host is refused", func(t *testing.T) {
		c, f := build(t, moved(func(w http.ResponseWriter) { w.WriteHeader(330) }))
		_, err := c.Album(context.Background(), testAlbum)
		if e := wantCode(t, err, CodeBadRedirect); e.Status != 330 || e.Op != "webstream" {
			t.Fatalf("error = %+v", e)
		}
		if n := len(f.seen()); n != 1 {
			t.Fatalf("%d requests sent after a refused redirect", n)
		}
	})

	for _, host := range []string{
		"evil.example/x",
		"user@evil.example",
		"evil.example:8443",
		"evil.example?x",
		"evil.example#x",
		`evil.example\x`,
		"evil example",
	} {
		t.Run(fmt.Sprintf("a 330 host of %q is refused", host), func(t *testing.T) {
			c, f := build(t, moved(func(w http.ResponseWriter) {
				w.WriteHeader(330)
				writeJSON(w, map[string]string{"X-Apple-MMe-Host": host})
			}))
			_, err := c.Album(context.Background(), testAlbum)
			wantCode(t, err, CodeBadRedirect)
			if n := len(f.seen()); n != 1 {
				t.Fatalf("%d requests sent after a refused redirect", n)
			}
		})
	}

	t.Run("a second 330 is refused rather than followed", func(t *testing.T) {
		c, f := build(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(330)
			_, _ = io.WriteString(w, `{"X-Apple-MMe-Host":"p42-sharedstreams.icloud.com"}`)
		})
		_, err := c.Album(context.Background(), testAlbum)
		if e := wantCode(t, err, CodeBadRedirect); e.Status != 330 {
			t.Fatalf("status = %d", e.Status)
		}
		if n := len(f.seen()); n != 2 {
			t.Fatalf("requests = %d, want 2", n)
		}
	})

	t.Run("a 330 from webasseturls is an unexpected status", func(t *testing.T) {
		c, _ := build(t, func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/webasseturls") {
				w.WriteHeader(330)
				return
			}
			writeJSON(w, stream(photo("p1", nil)))
		})
		_, err := c.Album(context.Background(), testAlbum)
		if e := wantCode(t, err, CodeUnexpectedStatus); e.Status != 330 || e.Op != "webasseturls" {
			t.Fatalf("error = %+v", e)
		}
	})
}

func TestAlbumStatus(t *testing.T) {
	for _, tc := range []struct {
		status int
		ok     bool
	}{
		{200, true},
		{299, true},
		{300, false},
		{404, false},
		{500, false},
	} {
		for _, op := range []string{"webstream", "webasseturls"} {
			t.Run(fmt.Sprintf("%s answering %d", op, tc.status), func(t *testing.T) {
				ok := serve(stream(photo("p1", nil)), assets())
				c, _ := build(t, func(w http.ResponseWriter, r *http.Request) {
					if strings.HasSuffix(r.URL.Path, "/"+op) {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(tc.status)
						if strings.HasSuffix(r.URL.Path, "/webstream") {
							_ = json.NewEncoder(w).Encode(stream(photo("p1", nil)))
						} else {
							_ = json.NewEncoder(w).Encode(assets())
						}
						return
					}
					ok(w, r)
				})
				_, err := c.Album(context.Background(), testAlbum)
				if tc.ok {
					if err != nil {
						t.Fatalf("a %d is a success: %v", tc.status, err)
					}
					return
				}
				if e := wantCode(t, err, CodeUnexpectedStatus); e.Status != tc.status || e.Op != op {
					t.Fatalf("error = %+v", e)
				}
			})
		}
	}
}

func TestAlbumMalformed(t *testing.T) {
	bodies := map[string]string{
		"an empty body":            ``,
		"a truncated body":         `{"photos":[`,
		"HTML instead of JSON":     `<html>maintenance</html>`,
		"photos that are a string": `{"photos":"none"}`,
	}
	for name, body := range bodies {
		t.Run("webstream with "+name+" is malformed", func(t *testing.T) {
			c, _ := build(t, func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, body) })
			_, err := c.Album(context.Background(), testAlbum)
			if e := wantCode(t, err, CodeMalformedResponse); e.Op != "webstream" || e.Err == nil {
				t.Fatalf("error = %+v", e)
			}
		})
		t.Run("webasseturls with "+name+" is malformed", func(t *testing.T) {
			ws := stream(photo("p1", nil))
			c, _ := build(t, func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "/webstream") {
					writeJSON(w, ws)
					return
				}
				_, _ = io.WriteString(w, strings.Replace(body, "photos", "items", 1))
			})
			_, err := c.Album(context.Background(), testAlbum)
			if e := wantCode(t, err, CodeMalformedResponse); e.Op != "webasseturls" {
				t.Fatalf("error = %+v", e)
			}
		})
	}

	fields := []struct {
		name, detail string
		mutate       func(p map[string]any)
	}{
		{"a non numeric width", `photo "p1": width "wide"`, func(p map[string]any) { p["width"] = "wide" }},
		{"an empty height", `photo "p1": height ""`, func(p map[string]any) { p["height"] = "" }},
		{"an unparseable dateCreated", `photo "p1": dateCreated "yesterday"`, func(p map[string]any) { p["dateCreated"] = "yesterday" }},
		{"an empty batchDateCreated", `photo "p1": batchDateCreated ""`, func(p map[string]any) { p["batchDateCreated"] = "" }},
		{"a derivative fileSize past int64", `photo "p1": derivatives.1.fileSize "9223372036854775808"`, func(p map[string]any) {
			p["derivatives"] = map[string]any{"1": map[string]any{"checksum": "c1", "fileSize": "9223372036854775808", "width": "1", "height": "1"}}
		}},
		{"a derivative height with a decimal", `photo "p1": derivatives.1.height "1.5"`, func(p map[string]any) {
			p["derivatives"] = map[string]any{"1": map[string]any{"checksum": "c1", "fileSize": "1", "width": "1", "height": "1.5"}}
		}},
	}
	for _, tc := range fields {
		t.Run(tc.name+" names the photo and the field", func(t *testing.T) {
			p := photo("p1", nil)
			tc.mutate(p)
			c, _ := build(t, serve(stream(p), assets("c1")))
			_, err := c.Album(context.Background(), testAlbum)
			e := wantCode(t, err, CodeMalformedResponse)
			if e.Detail != tc.detail || e.Err == nil {
				t.Fatalf("detail = %q, want %q (err %v)", e.Detail, tc.detail, e.Err)
			}
		})
	}

	t.Run("with two bad fields the first one in the photo is reported", func(t *testing.T) {
		p := photo("p1", nil)
		p["width"] = "wide"
		p["dateCreated"] = "yesterday"
		c, _ := build(t, serve(stream(p), assets()))
		_, err := c.Album(context.Background(), testAlbum)
		if e := wantCode(t, err, CodeMalformedResponse); e.Detail != `photo "p1": width "wide"` {
			t.Fatalf("detail = %q", e.Detail)
		}
	})

	t.Run("a bad number on a derivative without a URL is not checked", func(t *testing.T) {
		p := photo("p1", map[string]any{"1": map[string]any{"checksum": "gone", "fileSize": "x", "width": "x", "height": "x"}})
		c, _ := build(t, serve(stream(p), assets()))
		if _, err := c.Album(context.Background(), testAlbum); err != nil {
			t.Fatal(err)
		}
	})
}

func TestAlbumInvalidToken(t *testing.T) {
	c, f := build(t, serve(stream(), assets()))
	_, err := c.Album(context.Background(), "B0z/../../evil")
	wantCode(t, err, CodeInvalidToken)
	if n := len(f.seen()); n != 0 {
		t.Fatalf("%d requests sent for an invalid token", n)
	}
}

func TestAlbumEmpty(t *testing.T) {
	c, f := build(t, serve(stream(), assets()))
	album, err := c.Album(context.Background(), testAlbum)
	if err != nil {
		t.Fatal(err)
	}
	if album.Photos == nil || len(album.Photos) != 0 {
		t.Fatalf("an empty album has an empty, non-nil Photos: %#v", album.Photos)
	}
	if n := len(f.seen()); n != 1 {
		t.Fatalf("an empty album made %d requests, want only webstream", n)
	}
}

func TestAlbumTransportErrors(t *testing.T) {
	t.Run("a cancelled context comes back as the context error", func(t *testing.T) {
		c, f := build(t, serve(stream(), assets()))
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := c.Album(ctx, testAlbum)
		var e *Error
		if !errors.Is(err, context.Canceled) || errors.As(err, &e) {
			t.Fatalf("error = %v (%T), want the transport's context.Canceled", err, err)
		}
		if n := len(f.seen()); n != 0 {
			t.Fatalf("requests = %d", n)
		}
	})

	t.Run("a connection error mid body comes back unchanged, not as malformed", func(t *testing.T) {
		broken := errors.New("connection reset")
		c, err := NewClient(&http.Client{Timeout: time.Minute, Transport: failingBody{broken}})
		if err != nil {
			t.Fatal(err)
		}
		_, err = c.Album(context.Background(), testAlbum)
		var e *Error
		if !errors.Is(err, broken) || errors.As(err, &e) {
			t.Fatalf("error = %v (%T), want the transport's own error", err, err)
		}
	})
}

// failingBody answers with a 200 whose body breaks off partway with err.
type failingBody struct{ err error }

func (f failingBody) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{},
		Body:       io.NopCloser(io.MultiReader(strings.NewReader(`{"streamName":"Garden","photos":[`), errReader(f))),
		Request:    req,
	}, nil
}

type errReader struct{ err error }

func (r errReader) Read([]byte) (int, error) { return 0, r.err }

func TestErrorMessage(t *testing.T) {
	cause := errors.New("boom")
	cases := []struct {
		err  *Error
		want string
	}{
		{&Error{Code: CodeInvalidToken, Op: "token"}, "monarchbutterflyicloud: token: invalid_token"},
		{&Error{Code: CodeUnexpectedStatus, Op: "webstream", Status: 503}, "monarchbutterflyicloud: webstream: unexpected_status (HTTP 503)"},
		{&Error{Code: CodeMalformedResponse, Op: "webstream", Detail: `photo "p1": width "x"`, Err: cause}, `monarchbutterflyicloud: webstream: malformed_response: photo "p1": width "x": boom`},
	}
	for _, c := range cases {
		if got := c.err.Error(); got != c.want {
			t.Errorf("Error() = %q, want %q", got, c.want)
		}
	}
	if !errors.Is(cases[2].err, cause) {
		t.Error("Unwrap does not reach the cause")
	}
}
