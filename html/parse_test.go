package html_test

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/html"
)

func TestParseFindsElementText(t *testing.T) {
	t.Parallel()

	doc, err := html.Parse([]byte(`<div id="box">Hello</div>`))
	if err != nil {
		t.Fatal(err)
	}

	element, ok := doc.Find("box")
	if !ok {
		t.Fatal("box not found")
	}

	if element.Tag != "div" || element.Text != "Hello" {
		t.Fatalf("element = %+v", element)
	}
}

func TestParseNilSource(t *testing.T) {
	t.Parallel()

	doc, err := html.Parse(nil)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := doc.Find("box"); ok {
		t.Fatal("empty document found a box")
	}
}
