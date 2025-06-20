package templates

import (
	"bytes"
	"embed"
	"html/template"
	"io"
)

//go:embed *.html
var templates embed.FS

// TemplateData holds data for template rendering
type TemplateData struct {
	AuthURL string
}

// RenderTemplate renders an HTML template with the given data
func RenderTemplate(templateName string, data TemplateData) (string, error) {
	tmpl, err := template.ParseFS(templates, templateName)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// WriteTemplate writes a rendered template to the given writer
func WriteTemplate(w io.Writer, templateName string, data TemplateData) error {
	content, err := RenderTemplate(templateName, data)
	if err != nil {
		return err
	}

	_, err = w.Write([]byte(content))
	return err
}

// GetControlPanelHTML returns the control panel HTML content
func GetControlPanelHTML() (string, error) {
	content, err := templates.ReadFile("control_panel.html")
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// GetLoginPageHTML returns the login page HTML content
func GetLoginPageHTML() (string, error) {
	content, err := templates.ReadFile("login.html")
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// LoadLoginTemplate loads and parses the login template
func LoadLoginTemplate() (*template.Template, error) {
	return template.ParseFS(templates, "login.html")
}
