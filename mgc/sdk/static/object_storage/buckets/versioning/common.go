package versioning

type versioningConfiguration struct {
	Status    string `xml:"Status"`
	MfaDelete string `xml:"MfaDelete,omitempty"`

	Namespace string   `xml:"xmlns,omitempty,attr" json:"-"`
	XMLName   struct{} `xml:"VersioningConfiguration" json:"-"`
}
