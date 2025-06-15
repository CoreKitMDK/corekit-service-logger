package logger

import (
	"encoding/json"
	"fmt"
	"strings"
)

type IToString interface {
	ToString() string
}

func hasToStringMethod(value interface{}) bool {
	_, ok := value.(IToString)
	return ok
}

func tryToConvertToJSON(value interface{}) []byte {
	r, err := json.Marshal(value)
	if err != nil {
		return nil
	} else {
		return r
	}
}

func Stringify(args ...interface{}) string {
	var builder strings.Builder
	builder.Grow(512)

	for _, obj := range args {
		if hasToStringMethod(obj) {
			builder.WriteString(obj.(IToString).ToString())
		} else if jsn := tryToConvertToJSON(obj); jsn != nil {
			builder.WriteString(string(jsn))
		} else {
			builder.WriteString(fmt.Sprintf("%v ", obj))
		}
		builder.WriteString(" | ")
	}
	return builder.String()
}
