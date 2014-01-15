package encoder

// Borrowed from https://github.com/PuerkitoBio/martini-api-example

import (
	// "bytes"
	"encoding/json"
	// "encoding/xml"
	// "fmt"
	"reflect"
)

// An Encoder implements an encoding format of values to be sent as response to
// requests on the API endpoints.
type Encoder interface {
	Encode(v ...interface{}) ([]byte, error)
}

// Because `panic`s are caught by martini's Recovery handler, it can be used
// to return server-side errors (500). Some helpful text message should probably
// be sent, although not the technical error (which is printed in the log).
func Must(data []byte, err error) []byte {
	if err != nil {
		panic(err)
	}
	return data
}

type JsonEncoder struct{}

// jsonEncoder is an Encoder that produces JSON-formatted responses.
func (_ JsonEncoder) Encode(v ...interface{}) ([]byte, error) {
	var data interface{} = v
	var result interface{}

	if v == nil {
		// So that empty results produces `[]` and not `null`
		data = []interface{}{}
	} else if len(v) == 1 {
		data = v[0]
	}

	t := reflect.TypeOf(data)

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() == reflect.Struct {
		result = copyStruct(reflect.ValueOf(data).Elem(), t).Interface()
	} else {
		result = data
	}

	b, err := json.Marshal(result)

	return b, err
}

func copyStruct(v reflect.Value, t reflect.Type) reflect.Value {
	result := reflect.New(t).Elem()

	for i := 0; i < v.NumField(); i++ {
		if tag := t.Field(i).Tag.Get("out"); tag == "false" {
			continue
		}

		if v.Field(i).Kind() == reflect.Struct {
			result.Field(i).Set(copyStruct(v.Field(i), t.Field(i).Type))
			continue
		}

		result.Field(i).Set(v.Field(i))
	}

	return result
}

type XmlEncoder struct{}

// xmlEncoder is an Encoder that produces XML-formatted responses.
/*
func (_ XmlEncoder) Encode(v ...interface{}) (string, error) {
	var buf bytes.Buffer
	if _, err := buf.Write([]byte(xml.Header)); err != nil {
		return "", err
	}
	if _, err := buf.Write([]byte("<albums>")); err != nil {
		return "", err
	}
	b, err := xml.Marshal(v)
	if err != nil {
		return "", err
	}
	if _, err := buf.Write(b); err != nil {
		return "", err
	}
	if _, err := buf.Write([]byte("</albums>")); err != nil {
		return "", err
	}
	return buf.String(), nil
}

type TextEncoder struct{}

// textEncoder is an Encoder that produces plain text-formatted responses.
func (_ TextEncoder) Encode(v ...interface{}) (string, error) {
	var buf bytes.Buffer
	for _, v := range v {
		if _, err := fmt.Fprintf(&buf, "%s\n", v); err != nil {
			return "", err
		}
	}
	return buf.String(), nil
}
*/
