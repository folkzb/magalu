package core

import (
	"reflect"
	"strconv"
	"strings"
)

type MgcTag [2]string

// Standard way to get the tags, splitting by comma
func GetMgcTags(t reflect.StructTag) []MgcTag {
	tags := strings.Split(t.Get("mgc"), ",")
	result := make([]MgcTag, len(tags))
	for i, t := range tags {
		split := strings.SplitN(t, "=", 2)
		if len(split) != 2 {
			result[i] = MgcTag{split[0]}
		} else {
			result[i] = MgcTag(split)
		}

	}
	return result
}

func GetMgcTag(t reflect.StructTag, name string) (MgcTag, bool) {
	for _, mgcTag := range GetMgcTags(t) {
		if mgcTag.Name() == name {
			return mgcTag, true
		}
	}
	return MgcTag{}, false
}

func (t MgcTag) Name() string {
	return t[0]
}

func (t MgcTag) Value() (string, bool) {
	return t[1], t[1] != ""
}

func GetMgcTagBool(t reflect.StructTag, name string) bool {
	mgcTag, ok := GetMgcTag(t, name)
	if !ok {
		return false
	}

	value, ok := mgcTag.Value()
	if !ok {
		// ",tag"
		return true
	}

	// ",tag=false" ",tag=true"
	boolValue, _ := strconv.ParseBool(value)
	return boolValue
}
