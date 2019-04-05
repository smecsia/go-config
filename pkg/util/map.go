package util

import (
	"fmt"
	"strconv"
	"strings"
)

// GetValue allows to extract value from the provided object by the "path" inside of its structure
// Example:
// 		With the following object: map[string]interface{}{"a": map[string]interface{}{"b": "c"}}
//      the value "c" can be reached by "a.b.c"
func GetValue(key string, value interface{}) (res interface{}, err error) {
	if value == nil {
		return nil, nil
	}
	overallKey := key
	keys := strings.Split(key, ".")
	for _, key := range keys {
		value, err = getValPart(key, value, overallKey)
		if err != nil {
			return value, err
		}
	}
	return value, err
}

func getValPart(key string, value interface{}, overallKey string) (res interface{}, err error) {
	var (
		i  int64
		ok bool
	)
	switch value.(type) {
	case map[string]map[string]interface{}:
		if res, ok = value.(map[string]map[string]interface{})[key]; !ok {
			err = fmt.Errorf("key not present. [key:%s] of [path:%s]", key, overallKey)
		}
	case map[string]string:
		if res, ok = value.(map[string]string)[key]; !ok {
			err = fmt.Errorf("key not present. [key:%s] of [path:%s]", key, overallKey)
		}
	case map[string]interface{}:
		if res, ok = value.(map[string]interface{})[key]; !ok {
			err = fmt.Errorf("key not present. [key:%s] of [path:%s]", key, overallKey)
		}
	case []interface{}:
		if i, err = strconv.ParseInt(key, 10, 64); err == nil {
			array := value.([]interface{})
			if int(i) < len(array) {
				res = array[i]
			} else {
				err = fmt.Errorf("index out of bounds. [index:%d] [array:%v] of [path:%s]", i, array, overallKey)
			}
		}
	default:
		err = fmt.Errorf("unsupported value type for key [key:%s] of [path:%s] [value:%v]", key, overallKey, value)
	}
	return res, err
}
