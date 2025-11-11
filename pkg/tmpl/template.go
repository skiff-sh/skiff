package tmpl

import (
	"io"
	"strings"
	"text/template"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var caser = cases.Title(language.English)

type Template interface {
	Render(d map[string]any, in io.Writer) error
}

type Factory interface {
	NewTemplate(tmpl []byte) (Template, error)
}

func NewGoFactory() Factory {
	return &goFactory{}
}

type goFactory struct {
}

func (g *goFactory) NewTemplate(tmpl []byte) (Template, error) {
	t, err := template.New("").Funcs(template.FuncMap{
		"lower": strings.ToLower,
		"upper": strings.ToUpper,
		"capitalize": func(s string) string {
			if len(s) == 0 {
				return s
			}

			return caser.String(s[:1]) + s[1:]
		},
	}).Parse(string(tmpl))
	if err != nil {
		return nil, err
	}

	return &goTemplate{
		T: t,
	}, nil
}

type goTemplate struct {
	T *template.Template
}

func (g *goTemplate) Render(d map[string]any, in io.Writer) error {
	return g.T.Execute(in, d)
}
