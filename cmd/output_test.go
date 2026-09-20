package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/taichi/javcli/pkg/javdb"
)

func TestWriteErrorJSONLoginRequired(t *testing.T) {
	var buf bytes.Buffer
	err := writeErrorJSON(&buf, &javdb.LoginRequiredError{Message: "Unauthorized"})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["code"].(float64) != 401 || got["message"] != "Unauthorized" {
		t.Fatalf("%v", got)
	}
}

func TestWriteErrorJSONGeneric(t *testing.T) {
	var buf bytes.Buffer
	if err := writeErrorJSON(&buf, errors.New("boom")); err != nil {
		t.Fatal(err)
	}
	var got map[string]string
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["error"] != "boom" {
		t.Fatalf("%v", got)
	}
}

func TestFirstNonEmpty(t *testing.T) {
	if got := firstNonEmpty("", "env", "cfg"); got != "env" {
		t.Fatalf("%q", got)
	}
	if got := firstNonEmpty("", "", "cfg"); got != "cfg" {
		t.Fatalf("%q", got)
	}
}

func TestEnvTruthy(t *testing.T) {
	if !envTruthy("true") || !envTruthy("1") || envTruthy("no") || envTruthy("") {
		t.Fatal("envTruthy")
	}
}

func TestWriteJSONIndent(t *testing.T) {
	var buf bytes.Buffer
	if err := writeJSON(&buf, map[string]string{"a": "b"}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("\n  ")) {
		t.Fatalf("expected indent: %s", buf.String())
	}
}
