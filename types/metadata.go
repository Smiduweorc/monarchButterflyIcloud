package types

// MetaData holds stream-level metadata for a shared album.
type MetaData struct {
	StreamName    string      `json:"streamName"`
	UserFirstName string      `json:"userFirstName"`
	UserLastName  string      `json:"userLastName"`
	StreamCtag    string      `json:"streamCtag"`
	ItemsReturned int         `json:"itemsReturned"`
	Locations     interface{} `json:"locations"`
}
