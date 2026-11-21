package monarchbutterflyicloud

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"time"
)

// Album is a public iCloud Shared Album.
type Album struct {
	Name           string
	OwnerFirstName string
	OwnerLastName  string
	// Photos are in the order iCloud lists them. Videos are included; their
	// MediaAssetType is "video".
	Photos []Photo
}

// Photo is one item in an album.
type Photo struct {
	GUID                 string
	BatchGUID            string
	Caption              string
	ContributorFirstName string
	ContributorLastName  string
	ContributorFullName  string
	// MediaAssetType is "video" for videos and empty for photos, where
	// iCloud sends null.
	MediaAssetType string
	Width          int
	Height         int
	Created        time.Time
	BatchCreated   time.Time
	// Derivatives are the renditions iCloud returned a download URL for,
	// sorted by height, smallest first. A rendition with no URL in the
	// /webasseturls response is left out.
	Derivatives []Derivative
}

// Derivative is one rendition of a photo, such as a thumbnail or the full
// size original.
type Derivative struct {
	// Key is iCloud's name for the rendition, such as "720" or
	// "PosterFrame".
	Key      string
	Checksum string
	FileSize int64
	Width    int
	Height   int
	// URL is a signed download link on Apple's CDN.
	URL string
}

// The wire shapes below are what /webstream and /webasseturls send. iCloud
// sends every number as a JSON string.

type wireWebstream struct {
	StreamName    string      `json:"streamName"`
	UserFirstName string      `json:"userFirstName"`
	UserLastName  string      `json:"userLastName"`
	Photos        []wirePhoto `json:"photos"`
}

type wirePhoto struct {
	PhotoGUID            string                    `json:"photoGuid"`
	BatchGUID            string                    `json:"batchGuid"`
	Caption              string                    `json:"caption"`
	ContributorFirstName string                    `json:"contributorFirstName"`
	ContributorLastName  string                    `json:"contributorLastName"`
	ContributorFullName  string                    `json:"contributorFullName"`
	MediaAssetType       *string                   `json:"mediaAssetType"`
	Width                string                    `json:"width"`
	Height               string                    `json:"height"`
	DateCreated          string                    `json:"dateCreated"`
	BatchDateCreated     string                    `json:"batchDateCreated"`
	Derivatives          map[string]wireDerivative `json:"derivatives"`
}

type wireDerivative struct {
	Checksum string `json:"checksum"`
	FileSize string `json:"fileSize"`
	Width    string `json:"width"`
	Height   string `json:"height"`
}

type wireAssetURLs struct {
	Items map[string]struct {
		URLLocation string `json:"url_location"`
		URLPath     string `json:"url_path"`
	} `json:"items"`
}

// fieldParser collects the first parse failure so a photo can be converted
// field by field and checked once.
type fieldParser struct {
	photo string
	err   error
}

func (p *fieldParser) fail(field, value string, err error) {
	if p.err == nil {
		p.err = &Error{
			Code:   CodeMalformedResponse,
			Op:     "webstream",
			Detail: fmt.Sprintf("photo %q: %s %q", p.photo, field, value),
			Err:    err,
		}
	}
}

func (p *fieldParser) int(field, value string) int {
	n, err := strconv.Atoi(value)
	if err != nil {
		p.fail(field, value, err)
	}
	return n
}

func (p *fieldParser) int64(field, value string) int64 {
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		p.fail(field, value, err)
	}
	return n
}

func (p *fieldParser) time(field, value string) time.Time {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		// Also accept a numeric offset without a colon ("+0000"), which
		// RFC 3339 does not allow.
		t, err = time.Parse("2006-01-02T15:04:05Z0700", value)
	}
	if err != nil {
		p.fail(field, value, err)
	}
	return t
}

// toPhoto converts one wire photo, attaching URLs from urls (keyed by
// derivative checksum).
func toPhoto(w wirePhoto, urls map[string]string) (Photo, error) {
	p := &fieldParser{photo: w.PhotoGUID}

	photo := Photo{
		GUID:                 w.PhotoGUID,
		BatchGUID:            w.BatchGUID,
		Caption:              w.Caption,
		ContributorFirstName: w.ContributorFirstName,
		ContributorLastName:  w.ContributorLastName,
		ContributorFullName:  w.ContributorFullName,
		Width:                p.int("width", w.Width),
		Height:               p.int("height", w.Height),
		Created:              p.time("dateCreated", w.DateCreated),
		BatchCreated:         p.time("batchDateCreated", w.BatchDateCreated),
	}
	if w.MediaAssetType != nil {
		photo.MediaAssetType = *w.MediaAssetType
	}

	for key, d := range w.Derivatives {
		url, ok := urls[d.Checksum]
		if !ok {
			continue
		}
		photo.Derivatives = append(photo.Derivatives, Derivative{
			Key:      key,
			Checksum: d.Checksum,
			FileSize: p.int64("derivatives."+key+".fileSize", d.FileSize),
			Width:    p.int("derivatives."+key+".width", d.Width),
			Height:   p.int("derivatives."+key+".height", d.Height),
			URL:      url,
		})
	}
	if p.err != nil {
		return Photo{}, p.err
	}

	// Map iteration order is random; sort so the result is the same on
	// every call. Two renditions can share a height, so Key breaks the tie.
	slices.SortFunc(photo.Derivatives, func(a, b Derivative) int {
		return cmp.Or(cmp.Compare(a.Height, b.Height), cmp.Compare(a.Key, b.Key))
	})

	return photo, nil
}
