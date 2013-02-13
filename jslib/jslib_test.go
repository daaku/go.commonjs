package jslib_test

import (
	"github.com/daaku/go.commonjs/jslib"
	"testing"
)

// really just want to compile the source as a sanity check
func TestSanity(t *testing.T) {
	t.Parallel()
	if jslib.Bootstrap_2_2_2.Name() != "bootstrap" {
		t.Fatal("did not find expected name")
	}
}
