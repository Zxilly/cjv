package clictx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractPlusToolchain(t *testing.T) {
	tests := []struct {
		args       []string
		wantTC     string
		wantRemain []string
	}{
		{[]string{"+nightly", "./binary", "arg"}, "nightly", []string{"./binary", "arg"}},
		{[]string{"./binary", "arg"}, "", []string{"./binary", "arg"}},
		{[]string{"+lts-1.0.5", "cmd"}, "lts-1.0.5", []string{"cmd"}},
	}

	for _, tt := range tests {
		tc, remain := ExtractPlusToolchain(tt.args)
		assert.Equal(t, tt.wantTC, tc)
		assert.Equal(t, tt.wantRemain, remain)
	}
}
