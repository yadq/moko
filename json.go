package main

import (
	"encoding/json"
	"reflect"
)

// convert interface{} to json bytes
func MarshalJSON(d interface{}) ([]byte, error) {
	v, err := convertMapKeyToString(d)
	if err != nil {
		return nil, err
	}

	return json.Marshal(v)
}

func convertMapKeyToString(d interface{}) (interface{}, error) {
	switch v := reflect.ValueOf(d); v.Kind() {
	case reflect.Map:
		m := make(map[string]interface{})
		iter := v.MapRange()
		for iter.Next() {
			ik := iter.Key().Interface()
			iv, err := convertMapKeyToString(iter.Value().Interface())
			if err != nil {
				return nil, err
			}
			m[ik.(string)] = iv
		}
		return m, nil
	case reflect.Array:
	}

	return d, nil
}
