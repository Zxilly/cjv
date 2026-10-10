package main

import (
	"path/filepath"
	"testing"
)

func TestGuestExitStatus(t *testing.T) {
	for _, tc := range []struct {
		name, output, body string
		code               int
		invalid            bool
	}{
		{"pass", "PASS\n\nMARK=0\n", "PASS\n", 0, false},
		{"failure", "FAIL\r\n\r\nMARK=23\r\n", "FAIL\n", 23, false},
		{"signal", "\nMARK=137\n", "", 137, false},
		{"missing", "[Fail] device disconnected", "", 0, true},
		{"truncated", "PASS\nMARK=", "", 0, true},
		{"trailing transport error", "PASS\nMARK=0\n[Fail]", "", 0, true},
		{"out of range", "\nMARK=256\n", "", 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body, code, err := result([]byte(tc.output), "MARK=")
			if tc.invalid {
				if err == nil {
					t.Fatal("accepted invalid status")
				}
				return
			}
			if err != nil || string(body) != tc.body || code != tc.code {
				t.Fatalf("got %q, %d, %v", body, code, err)
			}
		})
	}
}

func TestQuote(t *testing.T) {
	for input, want := range map[string]string{"": "''", "a b": "'a b'", "a'b": "'a'\"'\"'b'", "$(touch x);\n": "'$(touch x);\n'"} {
		if got := quote(input); got != want {
			t.Fatalf("quote(%q) = %q", input, got)
		}
	}
}

func TestSourceMapping(t *testing.T) {
	source := t.TempDir()
	got, err := guestDir(source, filepath.Join(source, "internal", "ohos"), "/data/local/tmp/test")
	if err != nil || got != "/data/local/tmp/test/src/internal/ohos" {
		t.Fatalf("%q: %v", got, err)
	}
	if _, err := guestDir(source, filepath.Dir(source), "/data/local/tmp/test"); err == nil {
		t.Fatal("accepted outside source")
	}
}
