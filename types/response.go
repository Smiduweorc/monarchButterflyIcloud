package types

// the content from the parsed json response in the derivative field
type RawDerivative struct {
	Checksum string  `json:"checksum"`
	FileSize string  `json:"fileSize"`
	Width    string  `json:"width"`
	Height   string  `json:"height"`
	URL      *string `json:"url,omitempty"`
}

// RawPhoto struct from the parsed response
type RawPhoto struct {
	BatchGUID            string                   `json:"batchGUID"`
	Derivative           map[string]RawDerivative `json:"derivative"`
	ContributorLastName  string                   `json:"contributorLastName"`
	BatchDateCreated     string                   `json:"batchDateCreated"`
	ContributorFirstName string                   `json:"contributorFirstName"`
	ContributorFullName  string                   `json:"contributorFullName"`
	Caption              string                   `json:"caption"`
	Height               string                   `json:"height"`
	Width                string                   `json:"width"`
	MediaAssetType       *string                  `json:"mediaAssetType"` // occasionally returns null for some reason
	PhotoGuid            string                   `json:"photoGuid"`
	DateCreated          string                   `json:"dateCreated"`
}

// WebstreamResponse for the /webstream API response
type WebstreamResponse struct {
	Photos        []RawPhoto  `json:"photos"`
	StreamName    string      `json:"streamName"`
	UserFirstName string      `json:"userFirstName"`
	UserLastName  string      `json:"userLastName"`
	StreamCtag    string      `json:"streamCtag"`
	ItemsReturned string      `json:"itemsReturned"`
	Locations     interface{} `json:"locations"`
}
