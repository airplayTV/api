package util

import "encoding/json"

func ToString(v any) string {
	buff, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		return ""
	}
	return string(buff)
}
