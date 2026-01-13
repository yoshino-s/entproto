package struct_converter

import (
	"bytes"
	"text/template"
)

func t(temp string, data map[string]any) string {
	t, err := template.New("test").Parse(temp)
	if err != nil {
		panic(err)
	}
	var buf bytes.Buffer
	err = t.Execute(&buf, data)
	if err != nil {
		panic(err)
	}
	return buf.String()
}
