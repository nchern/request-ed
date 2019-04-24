package app

import (
	"bytes"
	"reflect"
	"regexp"
	"testing"
)

const (
	expectedBody = `{"size":10,"query":{"term":{"field":"value"}}}`
)

func TestParse(t *testing.T) {
	src := MustAsset(TemplateAsset)

	result, err := ParseRequest(bytes.NewBuffer(src))
	if err != nil {
		t.Errorf("ParseScript returned %s", err)
	}

	if err := result.Validate(); err != nil {
		t.Errorf("result.Validate returned %s", err)
	}

	if result.Host != "localhost:9200" {
		t.Errorf("Incorrect host: '%s'", result.Host)
	}

	if result.Method != "GET" {
		t.Errorf("Incorrect method: '%s'", result.Method)
	}

	if result.URI != "/foo/_search" {
		t.Errorf("Incorrect URI: '%s'", result.URI)
	}
	if result.URL() != "http://localhost:9200/foo/_search" {
		t.Errorf("Incorrect Url(): '%s'", result.URL())
	}

	json, err := result.JSON()
	if err != nil {
		t.Errorf("Malformed json: %s", err)
	}
	rx := regexp.MustCompile(`\s`)
	json = rx.ReplaceAllString(json, "")
	if json != expectedBody {
		t.Errorf("Unexpected json: [%s]", json)
	}
}

func TestParseEmptyRequestBody(t *testing.T) {
	selected := "POST /foo/bar"
	result, err := ParseRequest(bytes.NewBufferString(selected))
	if err != nil {
		t.Errorf("ParseScript returned %s", err)
	}
	body, err := result.JSON()
	if err != nil {
		t.Errorf("result.JSON returned %s", err)
	}
	if body != "" {
		t.Errorf("expected '' got %s", body)
	}
}

func TestParsePartial(t *testing.T) {
	selected := `
	POST /foo/bar
	{"size":10,"query":{"term":{"field":"value"}}}
	`
	result, err := ParseRequest(bytes.NewBufferString(selected))
	if err != nil {
		t.Errorf("ParseScript returned %s", err)
	}

	if result.Method != "POST" {
		t.Errorf("Incorrect method: '%s'", result.Method)
	}
	if result.URI != "/foo/bar" {
		t.Errorf("Incorrect URI: '%s'", result.URI)
	}
	if _, err := result.JSON(); err != nil {
		t.Errorf("Must be valid json but got: %s", err)
	}

}

func TestParseErrors(t *testing.T) {
	if _, err := ParseRequest(bytes.NewBufferString("GET")); err == nil {
		t.Errorf("Must return an error")
	}

	result, err := ParseRequest(bytes.NewBufferString(""))
	if err != nil {
		t.Errorf("ParseScript returned %s", err)
	}
	if err := result.Validate(); err == nil {
		t.Errorf("Must be invalid")
	}
}

func TestParseJsonTopLevelArray(t *testing.T) {
	selected := `
	POST /foo/bar
	[
		{"size":10}
	]
	`
	result, err := ParseRequest(bytes.NewBuffer([]byte(selected)))
	if err != nil {
		t.Errorf("ParseScript returned %s", err)
	}

	if result.Method != "POST" {
		t.Errorf("Incorrect method: '%s'", result.Method)
	}
	if result.URI != "/foo/bar" {
		t.Errorf("Incorrect URI: '%s'", result.URI)
	}
	if _, err := result.JSON(); err != nil {
		t.Errorf("Must be valid json but got: %s %s", err, reflect.TypeOf(err))
	}
}

type blockingReader struct{}

func (r *blockingReader) Read(p []byte) (n int, err error) {
	select {}
}

type panicWriter struct{}

func (w *panicWriter) Write(p []byte) (n int, err error) {
	panic("must not be called")
}

func TestTryParseAsync(t *testing.T) {
	// should read input and copy it to the output
	selected := "POST /foo/bar\n{}"
	var out bytes.Buffer
	var src = bytes.NewBufferString(selected)

	_, err := TryParseAsync(src, &out)
	if err != nil {
		t.Errorf("ParseScript returned %s", err)
	}
	if out.String() != selected {
		t.Errorf("expected: %s; actual: %s", selected, out.String())
	}

	// should handle blocked stream
	if _, err := TryParseAsync(&blockingReader{}, &panicWriter{}); err != errTimeout {
		t.Errorf("expected: %v; actual: %v", errTimeout, err)
	}

}
