package texttemplate

import (
	"bytes"
	"fmt"
	"github.com/Masterminds/sprig"
	"io/ioutil"
	"text/template"
)

func Parse(filename string, funcs template.FuncMap) (*template.Template, error) {
	raw, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	funcs2 := template.FuncMap{
		"checkSizeLessThan": func(size int, content string) (string, error) {
			if len(content) >= size {
				return "", fmt.Errorf("Content length exceeds maximum size %d", size)
			}
			return content, nil
		},
	}

	return template.New(filename).Funcs(sprig.HermeticTxtFuncMap()).Funcs(funcs).Funcs(funcs2).Parse(string(raw))
}

func GetBytesBuffer(filename string, data interface{}) (*bytes.Buffer, error) {
	tmpl, err := Parse(filename, nil)
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
