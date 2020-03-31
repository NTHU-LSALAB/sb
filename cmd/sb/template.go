package main

import (
	"html/template"
)

var htmlTemplate = template.Must(template.New("template.html").Parse(htmlTemplateString))
