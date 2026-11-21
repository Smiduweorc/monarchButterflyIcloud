// Package types defines the shapes of iCloud Shared Album API responses.
package types

// APIResponse is the return type for a dediated get api response function.
type APIResponse struct {
	Photos     map[string]Image `json:"photos"`
	PhotoGuids []string         `json:"photoGuids"`
	Metadata   MetaData         `json:"metadata"`
}
