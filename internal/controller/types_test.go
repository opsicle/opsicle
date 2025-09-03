package controller

import (
	"opsicle/pkg/controller"
	"reflect"
	"testing"
)

type structInfo struct {
	Name         string
	FieldTypeMap map[string]string
}

func getStructFieldInfo(v any) structInfo {
	result := structInfo{FieldTypeMap: make(map[string]string)}

	val := reflect.ValueOf(v)
	typ := reflect.TypeOf(v)

	// If it's a pointer, resolve it to the element
	if typ.Kind() == reflect.Ptr {
		val = val.Elem()
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
			result.FieldTypeMap[*jsonTagValue] = field.Type.String()
		}
	}

	return result
}

func validateModelContract(i structInfo, j structInfo, t *testing.T) {
	for m, n := range i.FieldTypeMap {
		o, ok := j.FieldTypeMap[m]
		if !ok {
			t.Errorf(
				"%s[%s] doesn't exist in %s",
				i.Name,
				m,
				j.Name,
			)
			continue
		}
		if n != o {
			t.Errorf(
				"%s[%s]'s type[%s] doesn't match %s[%s]'s type[%s]",
				i.Name,
				m,
				n,
				j.Name,
				m,
				o,
			)
		}
	}
}

func TestOrgSdkContracts(t *testing.T) {
	currentStruct := getStructFieldInfo(handleCreateOrgUserV1Input{})
	contractStruct := getStructFieldInfo(controller.CreateOrgUserV1Input{})
	validateModelContract(currentStruct, contractStruct, t)

	currentStruct = getStructFieldInfo(handleCreateOrgV1Input{})
	contractStruct = getStructFieldInfo(controller.CreateOrgV1Input{})
	validateModelContract(currentStruct, contractStruct, t)

	currentStruct = getStructFieldInfo(handleUpdateOrgInvitationV1Input{})
	contractStruct = getStructFieldInfo(controller.UpdateOrgInvitationV1Input{})
	validateModelContract(currentStruct, contractStruct, t)

	currentStruct = getStructFieldInfo(handleUpdateOrgUserV1Input{})
	contractStruct = getStructFieldInfo(controller.UpdateOrgUserV1Input{})
	validateModelContract(currentStruct, contractStruct, t)
}
