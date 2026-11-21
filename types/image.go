package types

import "time"

// Image is a single photo entry in an iCloud Shared Album.
type Image struct {
	BatchGUID            string               `json:"batchGuid"`
	Derivatives          map[string]ImageInfo `json:"derivatives"` // according to several other projects, yes this is in fact the name of the field, credits: https://github.com/harperreed/icloud-album-parser
	ContributorLastName  string               `json:"contributorLastName"`
	BatchDateCreated     time.Time            `json:"batchDateCreated"`
	ContributorFirstName string               `json:"contributorFirstName"`
	ContributorFullName  string               `json:"contributorFullName"`
	Caption              string               `json:"caption"`
	Height               int                  `json:"height"`
	Width                int                  `json:"width"`
	MediaAssetType       *string              `json:"mediaAssetType"`
	PhotoGUID            string               `json:"photoGuid"`
	DateCreated          time.Time            `json:"dateCreated"`
}
