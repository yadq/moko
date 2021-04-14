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
		// As json.Marshal does not support map[interface{}]interface{}, should convert key to string type
		m := make(map[string]interface{}, v.Len())
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
	case reflect.Slice:
		s := make([]interface{}, v.Len())
		for idx := 0; idx < v.Len(); idx++ {
			sv, err := convertMapKeyToString(v.Index(idx).Interface())
			if err != nil {
				return nil, err
			}
			s[idx] = sv
		}
		return s, nil
	}

	return d, nil
}
