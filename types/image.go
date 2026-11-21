package types

import "time"

type Image struct {
	BatchGUID            string                `json:"batchGuid"`
	Derivatives          map[string]ImageInfo  `json:"derivatives"` // according to several other projects, yes this is in fact the name of the field, credits: https://github.com/harperreed/icloud-album-parser
	ContributorLastName  string                `json:"contributorLastName"`
	BatchDateCreated     time.Time             `json:"batchDateCreated"`
	ContributorFirstName string                `json:"contributorFirstName"`
	ContributorFullName  string                `json:"contributorFullName"`
	Caption              string                `json:"caption"`
	Height               int                   `json:"Height"`
	Width                int                   `json:"Width"`
	MediaAssetType       *string               `json:"MediaAssetType"`
	PhotoGuid            string                `json:"PhotoGuid"`
	DateCreated          time.Time             `json:"DateCreated"`
}
