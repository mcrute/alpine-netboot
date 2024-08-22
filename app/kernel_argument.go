package app

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"text/template"
)

type KernelArgument struct {
	Key      string `yaml:"key"`
	Value    string `yaml:"value"`
	Template string `yaml:"template"`
}

func conditionalQuote(key, value string) string {
	if strings.ContainsAny(value, " \t=") {
		return fmt.Sprintf(`%s=%s`, key, strconv.Quote(value))
	}
	return fmt.Sprintf(`%s=%s`, key, value)
}

func renderTemplateArg(tpl string, d *Distribution) (string, error) {
	t, err := template.New("t").Parse(tpl)
	if err != nil {
		return "", err
	}

	buf := &bytes.Buffer{}
	if err := t.Execute(buf, d); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (a KernelArgument) Render(d *Distribution) (value string, err error) {
	if a.Value != "" { // Value arguments
		value = a.Value
	} else if a.Template != "" { // Template Arguments
		if value, err = renderTemplateArg(a.Template, d); err != nil {
			return "", nil
		}
	} else { // Unary arguments
		return a.Key, nil
	}

	return conditionalQuote(a.Key, value), nil
}
