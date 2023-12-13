package common

// Container for the owner's display name and ID.
type Owner struct {
	DisplayName string `xml:"DisplayName"`
	ID          string `xml:"ID" type:"ID"`
}
