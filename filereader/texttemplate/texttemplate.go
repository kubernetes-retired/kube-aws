package texttemplate

import (
	"bytes"
	"github.com/Masterminds/sprig"
	"io/ioutil"
	"text/template"
)

func GetBytesBuffer(filename string, data interface{}) (*bytes.Buffer, error) {
	raw, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	tmpl, err := template.New(filename).Funcs(sprig.HermeticTxtFuncMap()).Parse(string(raw))
	if err != nil {
		return nil, err
	}
	var buff bytes.Buffer
	if err := tmpl.Execute(&buff, data); err != nil {
		return nil, err
	}
	return &buff, nil
}

func GetString(filename string, data interface{}) (string, error) {
	buf, err := GetBytesBuffer(filename, data)

	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
