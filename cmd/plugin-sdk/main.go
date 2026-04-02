package main

import (
	"fmt"
	"os"
	"text/template"
)

const pluginTemplate = `package main

import (
	"context"
	"fmt"

	tkplugin "github.com/yourorg/toolkit/pkg/plugin"
)

// {{.TypeName}} implements the Tool interface.
type {{.TypeName}} struct {
	// Add your fields here
}

func (t *{{.TypeName}}) Metadata() tkplugin.ToolMetadata {
	return tkplugin.ToolMetadata{
		Name:        "{{.Name}}",
		Version:     "1.0.0",
		Description: "{{.Description}}",
		Author:      "{{.Author}}",
		Tags:        []string{ {{- range .Tags}}"{{.}}", {{end -}} },
		Category:    "{{.Category}}",
	}
}

func (t *{{.TypeName}}) Init(ctx context.Context, config map[string]interface{}) error {
	// Initialize your plugin here
	return nil
}

func (t *{{.TypeName}}) Execute(ctx context.Context, params map[string]interface{}) (*tkplugin.Result, error) {
	// Implement your tool logic here
	return &tkplugin.Result{
		Success: true,
		Data:    "result data",
	}, nil
}

func (t *{{.TypeName}}) Validate(params map[string]interface{}) error {
	// Validate input parameters
	return nil
}

func (t *{{.TypeName}}) Shutdown(ctx context.Context) error {
	// Cleanup resources
	return nil
}

func main() {
	tkplugin.Serve(&{{.TypeName}}{})
}
`

const manifestTemplate = `name: {{.Name}}
version: "1.0.0"
description: "{{.Description}}"
author: "{{.Author}}"
category: {{.Category}}
tags:
{{- range .Tags}}
  - {{.}}
{{- end}}
executable: {{.Name}}
enabled: true
`

type PluginTemplate struct {
	Name        string
	TypeName    string
	Description string
	Author      string
	Category    string
	Tags        []string
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: plugin-sdk <plugin-name> [description]")
		fmt.Println("Creates a new plugin scaffold in ./plugins/<name>/")
		os.Exit(1)
	}

	name := os.Args[1]
	description := "A toolkit plugin"
	if len(os.Args) > 2 {
		description = os.Args[2]
	}

	typeName := toCamelCase(name) + "Tool"

	tmpl := PluginTemplate{
		Name:        name,
		TypeName:    typeName,
		Description: description,
		Author:      "Your Name",
		Category:    "utility",
		Tags:        []string{"custom"},
	}

	dir := fmt.Sprintf("plugins/%s", name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create directory: %v\n", err)
		os.Exit(1)
	}

	// Write main.go
	mainFile, err := os.Create(fmt.Sprintf("%s/main.go", dir))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create main.go: %v\n", err)
		os.Exit(1)
	}
	defer mainFile.Close()

	t1 := template.Must(template.New("main").Parse(pluginTemplate))
	if err := t1.Execute(mainFile, tmpl); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write main.go: %v\n", err)
		os.Exit(1)
	}

	// Write plugin.yaml
	manifestFile, err := os.Create(fmt.Sprintf("%s/plugin.yaml", dir))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create plugin.yaml: %v\n", err)
		os.Exit(1)
	}
	defer manifestFile.Close()

	t2 := template.Must(template.New("manifest").Parse(manifestTemplate))
	if err := t2.Execute(manifestFile, tmpl); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write plugin.yaml: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Plugin scaffold created: %s/\n", dir)
	fmt.Println("Next steps:")
	fmt.Printf("  1. Edit %s/main.go to implement your logic\n", dir)
	fmt.Printf("  2. Update %s/plugin.yaml with your metadata\n", dir)
	fmt.Println("  3. Build: go build -o plugins/" + name + "/" + name + " plugins/" + name + "/main.go")
}

func toCamelCase(s string) string {
	result := ""
	capitalize := true
	for _, c := range s {
		if c == '-' || c == '_' {
			capitalize = true
			continue
		}
		if capitalize {
			if c >= 'a' && c <= 'z' {
				c -= 32
			}
			capitalize = false
		}
		result += string(c)
	}
	return result
}
