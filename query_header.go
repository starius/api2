package api2

import (
	"encoding"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
)

func writeQueryAndHeader(objPtr interface{}, query url.Values, header http.Header) error {
	objType := reflect.TypeOf(objPtr).Elem()
	objValue := reflect.ValueOf(objPtr).Elem()
	for i := 0; i < objType.NumField(); i++ {
		field := objType.Field(i)

		headerKey := field.Tag.Get("header")
		queryKey := field.Tag.Get("query")
		if headerKey == "" && (query == nil || queryKey == "") {
			continue
		}

		fieldObj := objValue.Field(i).Interface()
		value := ""
		if marshaler, ok := fieldObj.(encoding.TextMarshaler); ok {
			valueBytes, err := marshaler.MarshalText()
			if err != nil {
				return fmt.Errorf("failed to marshal value for field %s: %w", field.Name, err)
			}
			value = string(valueBytes)
		} else {
			value = fmt.Sprintf("%v", fieldObj)
		}

		if headerKey != "" {
			header.Set(headerKey, value)
		} else if query != nil && queryKey != "" {
			query.Set(queryKey, value)
		}
	}
	return nil
}

func parseQueryAndHeader(objPtr interface{}, query url.Values, header http.Header) error {
	objType := reflect.TypeOf(objPtr).Elem()
	objValue := reflect.ValueOf(objPtr).Elem()
	for i := 0; i < objType.NumField(); i++ {
		field := objType.Field(i)

		value := ""
		if headerKey := field.Tag.Get("header"); headerKey != "" {
			value = header.Get(headerKey)
		} else if query != nil {
			if queryKey := field.Tag.Get("query"); queryKey != "" {
				value = query.Get(queryKey)
			}
		}
		if value == "" {
			continue
		}

		var err error
		fieldPtr := objValue.Field(i).Addr().Interface()
		if unmarshaler, ok := fieldPtr.(encoding.TextUnmarshaler); ok {
			err = unmarshaler.UnmarshalText([]byte(value))
		} else if fieldStrPtr, ok := fieldPtr.(*string); ok {
			*fieldStrPtr = value
		} else {
			_, err = fmt.Sscanf(value, "%v", fieldPtr)
		}
		if err != nil {
			return fmt.Errorf("failed to parse value %q for field %s: %w", value, field.Name, err)
		}
	}
	return nil
}
