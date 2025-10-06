package xjson

import "encoding/json"

func Marshal(v any) []byte {
	bytes, _ := json.Marshal(v)
	return bytes
}

func MarshalString(v any) string {
	bytes, _ := json.Marshal(v)
	return string(bytes)
}
