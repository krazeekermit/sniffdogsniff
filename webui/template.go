package webui

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"path/filepath"
	"text/template"
)

//go:embed static/*
//go:embed templates/*

var embeddedFs embed.FS

type argsMap map[string]any

/* initialise templates with useful functions*/
func getFunctionsMap() template.FuncMap {
	return template.FuncMap{
		"loop": func(to int) <-chan int {
			ch := make(chan int)
			go func() {
				for i := 0; i < to; i++ {
					ch <- i
				}
				close(ch)
			}()
			return ch
		},
	}
}

func renderTemplate2(w http.ResponseWriter, templateName string, variablesMap map[string]any) {
	tmpl, err := template.New(templateName).Funcs(getFunctionsMap()).Parse(readTemplate(templateName))

	if err != nil {
		w.Write([]byte(fmt.Sprintf("<h1> panic HTTP server</h1><h3>%s</h3>", err.Error())))
	}
	err = tmpl.Execute(w, variablesMap)
	if err != nil {
		w.Write([]byte(fmt.Sprintf("<h1> panic HTTP server</h1><h3>%s</h3>", err.Error())))
	}
}

func renderTemplate(w http.ResponseWriter, templateName string, v any) {
	tmpl, err := template.New(templateName).Parse(readTemplate(templateName))

	if err != nil {
		w.Write([]byte(fmt.Sprintf("<h1> panic HTTP server</h1><h3>%s</h3>", err.Error())))
	}
	err = tmpl.Execute(w, v)
	if err != nil {
		w.Write([]byte(fmt.Sprintf("<h1> panic HTTP server</h1><h3>%s</h3>", err.Error())))
	}
}

func readTemplate(templateName string) string {
	data, err := embeddedFs.ReadFile(filepath.Join("templates/", templateName))
	if err != nil {
		panic("Template error Reading...")
	}
	return string(data)
}

func staticDir() fs.FS {
	fsys := fs.FS(embeddedFs)
	html, err := fs.Sub(fsys, "static")
	if err != nil {
		panic("Embedded files corruption!")
	}
	return html
}
