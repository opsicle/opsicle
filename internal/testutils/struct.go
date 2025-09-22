package testutils

import (
	"reflect"
	"testing"
)

type structInfo struct {
	Name         string
	FieldTypeMap map[string]string
}

func getStructFieldInfo(v any) structInfo {
	result := structInfo{FieldTypeMap: make(map[string]string)}

	// val := reflect.ValueOf(v)
	typ := reflect.TypeOf(v)

	// If it's a pointer, resolve it to the element
	if typ.Kind() == reflect.Ptr {
		// val = val.Elem()
		typ = typ.Elem()
	}

	if typ.Kind() != reflect.Struct {
		return result
	}

	result.Name = typ.Name()

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldName := field.Name
		var jsonTagValue *string = nil
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" {
			jsonTagValue = &fieldName
		} else if jsonTag != "-" {
			jsonTagValue = &jsonTag
		}

		if jsonTagValue != nil {
			// using .Kind() here allows maps to be evaludated separately
			result.FieldTypeMap[*jsonTagValue] = field.Type.Kind().String()
		}
	}

	return result
}

func ValidateModelContract(i any, j any, t *testing.T) {
	structA := getStructFieldInfo(i)
	structB := getStructFieldInfo(j)
	for structAField, structAType := range structA.FieldTypeMap {
		structBType, ok := structB.FieldTypeMap[structAField]
		if !ok {
			t.Errorf(
				"%s[%s] doesn't exist in %s",
				structA.Name,
				structAField,
				structB.Name,
			)
			continue
		}
		if structAType != structBType {
			t.Errorf(
				"%s[%s]'s type[%s] doesn't match %s[%s]'s type[%s]",
				structA.Name,
				structAField,
				structAType,
				structB.Name,
				structAField,
				structBType,
			)
		}
	}
}
