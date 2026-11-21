package monarchbutterflyicloud_test

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	monarchbutterflyicloud "github.com/Smiduweorc/monarchButterflyIcloud"
)

// The token is the part of a share link after the '#':
// https://www.icloud.com/sharedalbum/#B0z5qAGN1JIFd3y
func ExampleClient_Album() {
	c, err := monarchbutterflyicloud.NewClient(&http.Client{Timeout: 20 * time.Second})
	if err != nil {
		log.Fatal(err)
	}
	album, err := c.Album(context.Background(), "B0z5qAGN1JIFd3y")
	if err != nil {
		log.Fatal(err)
	}
	for _, p := range album.Photos {
		for _, d := range p.Derivatives {
			fmt.Println(p.GUID, d.Height, d.URL)
		}
	}
}
