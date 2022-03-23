package reflect

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/dlshle/gommon/utils"
)

const (
	setterMethodPrefix = "Set"
)

func getReflectKind(o interface{}) reflect.Type {
	t := reflect.TypeOf(o)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func GetFieldByName(o interface{}, fieldName string) (reflect.StructField, error) {
	t := getReflectKind(o)
	field, found := t.FieldByName(fieldName)
	if !found {
		return reflect.StructField{}, fmt.Errorf("invalid field name " + fieldName)
	}
	return field, nil
}

func GetTagByField(o interface{}, fieldName string) (string, error) {
	t := getReflectKind(o)
	field, found := t.FieldByName(fieldName)
	if !found {
		return "", fmt.Errorf("invalid field name " + fieldName)
	}
	return string(field.Tag), nil
}

func GetFieldsAndTags(o interface{}) (map[string]string, error) {
	fields, err := GetFields(o)
	if err != nil {
		return nil, err
	}
	m := make(map[string]string)
	for i := range fields {
		m[fields[i].Name] = string(fields[i].Tag)
	}
	return m, nil
}

func GetFields(o interface{}) ([]reflect.StructField, error) {
	t := getReflectKind(o)
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("object is not of struct kind")
	}
	fields := make([]reflect.StructField, t.NumField())
	for k := 0; k < t.NumField(); k++ {
		fields[k] = t.Field(k)
	}
	return fields, nil
}

func GetFieldsAndValues(o interface{}) (map[string]interface{}, error) {
	t := getReflectKind(o)
	v := reflect.ValueOf(o)
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("object is not of struct kind")
	}
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("object is not of struct kind")
	}
	res := make(map[string]interface{})
	numFields := t.NumField()
	for k := 0; k < numFields; k++ {
		fName := t.Field(k).Name
		res[fName] = v.FieldByName(fName).Interface()
	}
	return res, nil
}

func GetValueByField(o interface{}, field string) (reflect.Value, error) {
	v := reflect.ValueOf(o)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("object is not of struct kind")
	}
	fv := v.FieldByName(field)
	if !fv.IsValid() {
		return reflect.Value{}, fmt.Errorf("invalid field name " + field)
	}
	return fv, nil
}

// setter = SetX
func tryToGetSetterAndSet(object reflect.Value, fieldName string, value interface{}) bool {
	setterMethodName := fmt.Sprintf("%s%s", setterMethodPrefix, utils.ToPascalCase(fieldName))
	mv := object.MethodByName(setterMethodName)
	if !(mv.IsValid() && mv.Kind() == reflect.Func) {
		return false
	}
	mv.Call([]reflect.Value{reflect.ValueOf(value)})
	if recover() != nil {
		return false
	}
	return true
}

func SetValueOnField(o interface{}, fieldName string, value interface{}) error {
	v, e := GetValueByField(o, fieldName)
	if e != nil {
		return e
	}
	if v.CanSet() {
		v.Set(reflect.ValueOf(value))
	} else {
		if !tryToGetSetterAndSet(reflect.ValueOf(o), fieldName, value) {
			return fmt.Errorf("can not set field " + fieldName)
		}
	}
	return nil
}

func GetFieldTypeByName(o interface{}, field string) (string, error) {
	t := getReflectKind(o)
	if t.Kind() != reflect.Struct {
		return "", fmt.Errorf("object is not of struct kind")
	}
	targetField, found := t.FieldByName(field)
	if !found {
		return "", fmt.Errorf("can not find field " + field)
	}
	typeString := targetField.Type.String()
	if strings.HasPrefix(typeString, "reflect.") {
		typeString = strings.TrimPrefix(typeString, "reflect.")
	}
	return typeString, nil
}

func GetObjectType(o interface{}) string {
	return getReflectKind(o).Name()
}
