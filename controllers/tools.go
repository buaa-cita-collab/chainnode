package controllers

import (
	"bytes"
	"encoding/hex"
	"golang.org/x/crypto/sha3"
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

// Compute sha3_256 to a 0x started hex string
func sha3_256HexString(data string) string {
	d := sha3.Sum256([]byte(data))
	h := "0x" + hex.EncodeToString(d[:])
	return h
}
