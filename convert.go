package prometheus

import "strings"

func mapToStr(m map[string]string) string {
	var builder strings.Builder
	builder.WriteString("{")
	first := true
	for key, value := range m {
		if first {
			first = false
		} else {
			builder.WriteString(",")
		}
		builder.WriteString(key)
		builder.WriteString("=")
		builder.WriteString("\"")
		builder.WriteString(value)
		builder.WriteString("\"")
	}
	builder.WriteString("}")
	return builder.String()
}
