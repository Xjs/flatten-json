package main

import (
	"bytes"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSkipExample(t *testing.T) {
	testEndToEnd(t, "testdata/skip-example/input.json", []string{"_data"}, "testdata/skip-example/output.csv")
}

func testEndToEnd(t *testing.T, inputFile string, skip []string, refFile string) {
	input, err := os.Open(inputFile)
	if err != nil {
		t.Fatalf("Open(%q) error = %v", inputFile, err)
	}

	records, err := readAndClose(input, skip)
	if err != nil {
		t.Fatalf("readAndClose(%q, %q) error = %v", inputFile, skip, err)
	}

	var buf bytes.Buffer

	if err := tabular(records, ',', &buf); err != nil {
		t.Fatalf("tabular(recs, &buf) error = %v", err)
	}

	ref, err := os.ReadFile(refFile)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", refFile, err)
	}

	if diff := cmp.Diff(string(buf.Bytes()), string(ref)); diff != "" {
		t.Errorf("process(%q) diff with %q (-want +got):\n%s", inputFile, refFile, diff)
	}
}
