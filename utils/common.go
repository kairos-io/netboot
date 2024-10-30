// Copyright 2024 Kairos contributors

package utils

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

func ExpandCmdline(tpl string, funcs template.FuncMap) (string, error) {
	tmpl, err := template.New("cmdline").Option("missingkey=error").Funcs(funcs).Parse(tpl)
	if err != nil {
		return "", fmt.Errorf("parsing cmdline %q: %s", tpl, err)
	}
	var out bytes.Buffer
	if err = tmpl.Execute(&out, nil); err != nil {
		return "", fmt.Errorf("expanding cmdline template %q: %s", tpl, err)
	}
	cmdline := strings.TrimSpace(out.String())
	if strings.Contains(cmdline, "\n") {
		return "", fmt.Errorf("cmdline %q contains a newline", cmdline)
	}
	return cmdline, nil
}
