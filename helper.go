package main

import (
	"fmt"
	"reflect"
	"strings"
)

func UniqStringSlice(elements []string) []string {
	encountered := make(map[string]bool)
	var result []string
	for v := range elements {
		if encountered[elements[v]] == true {
			continue
		}
		encountered[elements[v]] = true
		result = append(result, elements[v])
	}
	return result
}

func ContainsStringSlice(slice []string, str string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}
	return false
}

func GetValueByJSONTag(obj reflect.Value, tagName string) reflect.Value {
	if obj.IsZero() {
		return reflect.Value{}
	}
	if obj.Kind() == reflect.Ptr {
		obj = obj.Elem()
	}
	tp := obj.Type()
	for i := 0; i < tp.NumField(); i++ {
		field := tp.Field(i)
		tag := field.Tag.Get("json")
		tag = strings.Split(tag, ",")[0]
		if tag == tagName {
			return obj.Field(i)
		}
	}
	return reflect.Value{}
}

func SetValueByJSONTag(obj reflect.Value, tagName string, value interface{}) error {
	structValue := obj.Elem()
	structType := structValue.Type()

	for i := 0; i < structValue.NumField(); i++ {
		field := structType.Field(i)
		tag := field.Tag.Get("json")
		tag = strings.Split(tag, ",")[0]
		if tag == tagName {
			fieldValue := structValue.Field(i)
			if !fieldValue.CanSet() {
				return fmt.Errorf("cannot set %s field value", field.Name)
			}

			val := reflect.ValueOf(value)
			if fieldValue.Type() != val.Type() {
				return fmt.Errorf("provided value type didn't match obj field type")
			}

			fieldValue.Set(val)
			return nil
		}
	}

	return fmt.Errorf("no such json tag in obj: %s", tagName)
}
