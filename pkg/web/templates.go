package web

import (
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/exograd/eventline/pkg/utils"
)

var TemplateFuncMap = template.FuncMap{
	"add": func(a, b int) int {
		return a + b
	},

	"sub": func(a, b int) int {
		return a - b
	},

	"toSentence": utils.ToSentence,

	"join": strings.Join,

	"stringMember": func(s string, ss []string) bool {
		for _, s2 := range ss {
			if s == s2 {
				return true
			}
		}

		return false
	},
}

func LoadTemplates(dirPath string) (*template.Template, error) {
	rootTpl := template.New("")
	rootTpl = rootTpl.Option("missingkey=error")
	rootTpl = rootTpl.Funcs(TemplateFuncMap)

	err := filepath.Walk(dirPath,
		func(filePath string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			if !strings.HasSuffix(filePath, ".html.gotpl") {
				return nil
			}

			start := len(dirPath) + 1
			end := len(filePath) - len(".gotpl")
			tplName := filePath[start:end]

			tplData, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("cannot read %q: %w", filePath, err)
			}

			tmpl := rootTpl.New(tplName)
			if _, err := tmpl.Parse(string(tplData)); err != nil {
				return fmt.Errorf("cannot parse %q: %w", filePath, err)
			}

			return nil
		})
	if err != nil {
		return nil, err
	}

	return rootTpl, nil
}
