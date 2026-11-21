// Package monarchbutterflyicloud reads public iCloud Shared Albums: the
// album's name and owner, and for every photo and video its metadata and a
// download URL for each rendition.
//
// It talks to the same undocumented endpoints the icloud.com web viewer uses
// (/webstream and /webasseturls). Nothing is authenticated, so only albums
// shared as a public website work.
package monarchbutterflyicloud
