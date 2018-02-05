package ld

import (
	"bytes"

	"github.com/hhatto/gorst"
	"gopkg.in/russross/blackfriday.v2"
)

func PreprocessMarkdown(text string) string {
	html := blackfriday.Run([]byte(text))
	// Repeat to times to heal broken HTML
	return HTML2Text(HTML2Text(string(html)))
}

func PreprocessRestructuredText(text string) string {
	parser := rst.NewParser(nil)
	input := bytes.NewBufferString(text)
	output := &bytes.Buffer{}
	parser.ReStructuredText(input, rst.ToHTML(output))
	// Repeat to times to heal broken HTML
	return HTML2Text(HTML2Text(output.String()))
}

func PreprocessHtml(text string) string {
	return HTML2Text(HTML2Text(text))
}
