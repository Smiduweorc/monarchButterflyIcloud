package types

type MetaData struct {
	StreamName    string      `json:"StreamName"`
	UserFirstName string      `json:"UserFirstName"`
	UserLastName  string      `json:"UserLastName"`
	StreamCtag    string      `json:"StreamCtag"`
	ItemsReturned int         `json:"ItemsReturned"`
	Locations     interface{} `json:"Locations"`
}
