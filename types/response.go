package types

// the content from the parsed json response in the derivative field
type RawDerivative struct {
	Checksum string  `json:"checksum"`
	FileSize string  `json:"fileSize"`
	Width    string  `json:"width"`
	Height   string  `json:"height"`
	URL      *string `json:"url,omitempty"`
}
