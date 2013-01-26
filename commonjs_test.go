package commonjs_test

import (
	"bytes"
	"github.com/daaku/go.commonjs"
	"math"
	"testing"
)

func TestCustomProvider(t *testing.T) {
	t.Parallel()
	const name = "foo"
	c := &commonjs.CustomProvider{}
	m := &commonjs.Module{Name: name}
	if err := c.Add(m); err != nil {
		t.Fatal(err)
	}

	// ensure it satisfies commonjs.Provider
	var p commonjs.Provider = c
	m2, err := p.Module(name)
	if err != nil {
		t.Fatal(err)
	}
	if m2 != m {
		t.Fatal("did not find expected module")
	}
}

func TestCustomProviderRepeatModule(t *testing.T) {
	t.Parallel()
	const name = "foo"
	c := &commonjs.CustomProvider{}
	m := &commonjs.Module{Name: name}
	if err := c.Add(m); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(m); err == nil {
		t.Fatal("was expecting a error")
	}
}

func TestCustomProviderMissingName(t *testing.T) {
	t.Parallel()
	c := &commonjs.CustomProvider{}
	m := &commonjs.Module{}
	if err := c.Add(m); err == nil {
		t.Fatal("was expecting a error")
	}
}

func TestCustomProviderModuleNotFound(t *testing.T) {
	t.Parallel()
	const name = "foo"
	c := &commonjs.CustomProvider{}
	if _, err := c.Module(name); err == nil {
		t.Fatal("was expecting an error")
	}
}

func TestJSONModule(t *testing.T) {
	t.Parallel()
	const name = "foo"
	m, err := commonjs.NewJSONModule("foo", map[string]int{"answer": 42})
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != name {
		t.Fatal("did not find expected name")
	}
	if string(m.Content) != "return {\"answer\":42}\n" {
		t.Fatalf(`did not find expected content, found "%s"`, m.Content)
	}
}

func TestJSONModuleError(t *testing.T) {
	t.Parallel()
	const name = "foo"
	if _, err := commonjs.NewJSONModule("foo", math.NaN()); err == nil {
		t.Fatal("was expecting an error")
	}
}

func TestURLBacked(t *testing.T) {
	t.Parallel()
	const name = "jquery"
	m, err := commonjs.NewURLModule(
		name,
		"http://ajax.googleapis.com/ajax/libs/jquery/1.9.0/jquery.js")
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != name {
		t.Fatalf("unexpected name %s", m.Name)
	}
	if !bytes.Contains(m.Content, []byte("jQuery JavaScript Library v1.9.0")) {
		t.Fatalf("did not find expected content")
	}
}

func TestURLBackedInvalid(t *testing.T) {
	t.Parallel()
	if _, err := commonjs.NewURLModule("foo", "foo"); err == nil {
		t.Fatal("was expecting an error")
	}
}
