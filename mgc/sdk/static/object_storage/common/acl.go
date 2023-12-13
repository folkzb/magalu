package common

type AccessControlPolicy struct {
	Owner             Owner             `xml:"Owner"`
	AccessControlList AccessControlList `xml:"AccessControlList"`

	XMLName struct{} `xml:"AccessControlPolicy" json:"-"`
}

type AccessControlList struct {
	Grant Grant `xml:"Grant"`
}

type Grant struct {
	Grantee    Grantee `xml:"Grantee"`
	Permission string  `xml:"Permission"`
}

type Grantee struct {
	DisplayName  string `xml:"DisplayName"`
	EmailAddress string `xml:"EmailAddress"`
	ID           string `xml:"ID"`
	URI          string `xml:"URI"`
}
