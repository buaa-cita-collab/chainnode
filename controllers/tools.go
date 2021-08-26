package controllers

import (
	"bytes"
	"text/template"
)

// Generate string according to template and data
func templateBuilder(
	templateString string,
	data interface{},
) (string, error) {
	temp, err := template.New("temp").Parse(templateString)
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	err = temp.Execute(buf, data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
