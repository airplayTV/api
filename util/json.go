package util

import "encoding/json"

func ToString(v any, pretty ...bool) string {
	var buff []byte
	var err error
	if len(pretty) > 0 && pretty[0] == true {
		buff, err = json.MarshalIndent(v, "", "\t")
	} else {
		buff, err = json.Marshal(v)
	}
	if err != nil {
		return ""
	}
	return string(buff)
}
