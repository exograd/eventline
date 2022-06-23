package web

import (
	"bytes"
	"html/template"
	"strings"
)

type View struct {
	RootTemplate *template.Template

	Title      string
	Menu       *Menu
	Breadcrumb *Breadcrumb
	Tabs       *Tabs
	Body       Content

	// Set internally
	Context interface{}

	PageId   string
	BodyData template.HTML
}

type Content interface {
	Render(interface{}) ([]byte, error)
}

func (v *View) Render(ctx interface{}) ([]byte, error) {
	v.Context = ctx

	if v.PageId == "" {
		if bodyTemplate, ok := v.Body.(*Template); ok {
			name := bodyTemplate.Name
			idx := strings.IndexByte(name, '.')
			v.PageId = name[:idx]
		}
	}

	bodyData, err := v.Body.Render(ctx)
	if err != nil {
		return nil, err
	}
	v.BodyData = template.HTML(bodyData)

	template := Template{
		RootTemplate: v.RootTemplate,
		Name:         "page.html",
		Data:         v,
	}

	return template.Render(ctx)
}

type RawContent []byte

func (c RawContent) Render(interface{}) ([]byte, error) {
	return []byte(c), nil
}

type Template struct {
	RootTemplate *template.Template
	Name         string
	Data         interface{}
}

func NewTemplate(rootTemplate *template.Template, name string, data interface{}) *Template {
	return &Template{
		RootTemplate: rootTemplate,
		Name:         name,
		Data:         data,
	}
}

func (t *Template) Render(ctx interface{}) ([]byte, error) {
	var buf bytes.Buffer

	data := struct {
		Context interface{}
		Data    interface{}
	}{
		Context: ctx,
		Data:    t.Data,
	}

	if err := t.RootTemplate.ExecuteTemplate(&buf, t.Name, data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
