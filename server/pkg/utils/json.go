package utils

import (
	"encoding/json"
)

// JSONEncode 将对象编码为 JSON 字符串
func JSONEncode(v interface{}) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// JSONDecode 将 JSON 字符串解码为对象
func JSONDecode(jsonStr string, v interface{}) error {
	return json.Unmarshal([]byte(jsonStr), v)
}
