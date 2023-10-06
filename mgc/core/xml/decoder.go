package xml

import (
	"encoding/xml"
	"fmt"
	"io"
	"reflect"
	"strings"

	"golang.org/x/exp/slices"
)

type Decoder struct {
	impl                  *xml.Decoder
	disallowUnknownFields bool
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{impl: xml.NewDecoder(r)}
}

func (d *Decoder) DisallowUnknownFields() {
	d.disallowUnknownFields = true
}

func (d *Decoder) decodeStructRigid(value reflect.Value) error {
	t := value.Type()
	structFields := make([]reflect.StructField, t.NumField())
	hasXmlnsCatcher := false
	hasExtraElemCatcher := false
	hasExtraAttrCatcher := false
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		structFields[i] = reflect.StructField{
			Name: field.Name,
			Type: field.Type,
			Tag:  field.Tag,
		}
		xmlTag, ok := field.Tag.Lookup("xml")
		if !ok {
			continue
		}
		xmlTagValues := strings.Split(xmlTag, ",")
		hasXmlnsTag := slices.Contains(xmlTagValues, "xmlns")
		hasAnyTag := slices.Contains(xmlTagValues, "any")
		hasAttrTag := slices.Contains(xmlTagValues, "attr")

		if hasXmlnsTag {
			hasXmlnsCatcher = true
		}
		// These need to be mutually exclusive
		if hasAnyTag && hasAttrTag {
			hasExtraAttrCatcher = true
		} else if hasAnyTag {
			hasExtraElemCatcher = true
		}
	}

	extraType := reflect.SliceOf(reflect.TypeOf(byte(0)))
	// These fields will capture any remaining XML elements and attributes.
	if !hasXmlnsCatcher {
		// Xmlns will avoid failures due to lack of Xmlns catching in original struct. Most structs won't have this, as it's
		// not very useful, we shouldn't fail because of the lack of Xmlns catches...
		structFields = append(structFields, reflect.StructField{Name: "ExtraXmlns__", Type: extraType, Tag: `xml:"xmlns,attr"`})
	}
	if !hasExtraElemCatcher {
		structFields = append(structFields, reflect.StructField{Name: "ExtraElem__", Type: extraType, Tag: `xml:",any"`})
	}
	if !hasExtraAttrCatcher {
		structFields = append(structFields, reflect.StructField{Name: "ExtraAttr__", Type: extraType, Tag: `xml:",any,attr"`})
	}

	newStructType := reflect.StructOf(structFields)
	newStructPtrValue := reflect.New(newStructType)

	err := d.impl.Decode(newStructPtrValue.Interface())
	if err != nil {
		return err
	}

	newStructValue := newStructPtrValue.Elem()
	var extraElem, extraAttr []byte
	if !hasExtraElemCatcher {
		extraElem = newStructValue.FieldByName("ExtraElem__").Interface().([]byte)
	}
	if !hasExtraAttrCatcher {
		extraAttr = newStructValue.FieldByName("ExtraAttr__").Interface().([]byte)
	}
	if len(extraElem) != 0 || len(extraAttr) != 0 {
		return fmt.Errorf(
			"struct %T does not properly match structure of XML document. Missing elements: %v Missing attributes: %v",
			value.Interface(),
			string(extraElem[:]),
			string(extraAttr[:]),
		)
	}

	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		if !field.CanSet() {
			return fmt.Errorf("unable to decode")
		}

		field.Set(newStructValue.Field(i))
	}

	return nil
}

func (d *Decoder) Decode(v any) error {
	ptrValue := reflect.ValueOf(v)
	if ptrValue.Type().Kind() != reflect.Pointer {
		return fmt.Errorf("target passed to 'Decode' must be a pointer, got %T instead", v)
	}

	value := ptrValue.Elem()
	kind := value.Kind()

	switch kind {
	case reflect.String, reflect.Slice:
		return d.impl.Decode(v)
	case reflect.Struct:
		if d.disallowUnknownFields {
			return d.decodeStructRigid(value)
		} else {
			return d.impl.Decode(v)
		}
	case reflect.Interface:
		// Empty name == any
		if value.Type().Name() == "" {
			return d.impl.Decode(v)
		}
	}

	return fmt.Errorf("target passed to 'Decode' must be a pointer to a struct, a string or a slice. got %T instead", v)
}
