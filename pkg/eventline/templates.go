package eventline

import (
	"fmt"
	htmltemplate "html/template"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	texttemplate "text/template"

	"github.com/exograd/eventline/pkg/utils"
)

var TemplateFuncMap = map[string]interface{}{
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

func LoadTextTemplates(dirPath string) (*texttemplate.Template, error) {
	rootTpl := texttemplate.New("")
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

			if !strings.HasSuffix(filePath, ".txt.gotpl") {
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

func LoadHTMLTemplates(dirPath string) (*htmltemplate.Template, error) {
	rootTpl := htmltemplate.New("")
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
