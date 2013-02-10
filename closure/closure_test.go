package closure_test

import (
	"bytes"
	"github.com/daaku/go.commonjs/closure"
	"testing"
)

func TestSimple(t *testing.T) {
	t.Parallel()
	in := []byte("function foo() { return 1; }")
	expected := []byte("function foo(){return 1};")
	c := &closure.Closure{}
	actual, err := c.Transform(in)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(expected, actual) != 0 {
		t.Fatalf("did not get expected output, got: %s", actual)
	}
}
