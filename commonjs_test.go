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
	m := commonjs.NewModule(name, nil)
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
	m := commonjs.NewModule(name, nil)
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
	m := commonjs.NewModule("", nil)
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
	m := commonjs.NewJSONModule("foo", map[string]int{"answer": 42})
	if m.Name() != name {
		t.Fatal("did not find expected name")
	}
	content, err := m.Content()
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "exports.module={\"answer\":42}\n" {
		t.Fatalf(`did not find expected content, found "%s"`, content)
	}
}

func TestJSONModuleError(t *testing.T) {
	t.Parallel()
	const name = "foo"
	if _, err := commonjs.NewJSONModule("foo", math.NaN()).Content(); err == nil {
		t.Fatal("was expecting an error")
	}
}

func TestURLBackedModule(t *testing.T) {
	t.Parallel()
	const name = "jquery"
	m := commonjs.NewURLModule(
		name,
		"http://ajax.googleapis.com/ajax/libs/jquery/1.9.0/jquery.js")
	if m.Name() != name {
		t.Fatalf("unexpected name %s", m.Name())
	}
	content, err := m.Content()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(content, []byte("jQuery JavaScript Library v1.9.0")) {
		t.Fatalf("did not find expected content")
	}
}

func TestURLBackedModuleInvalid(t *testing.T) {
	t.Parallel()
	if _, err := commonjs.NewURLModule("foo", "foo").Content(); err == nil {
		t.Fatal("was expecting an error")
	}
}

func TestFileBackedModule(t *testing.T) {
	t.Parallel()
	const name = "foo"
	m := commonjs.NewFileModule(name, "_test/a/foo.js")
	if m.Name() != name {
		t.Fatalf("unexpected name %s", m.Name())
	}
	content, err := m.Content()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(content, []byte("require")) {
		t.Fatalf("did not find expected content")
	}
	r, err := m.Require()
	if err != nil {
		t.Fatal(err)
	}
	if len(r) != 2 || r[0] != "bar" || r[1] != "b/baz" {
		t.Fatal("did not find expected require")
	}
}

func TestFileBackedModuleInvalid(t *testing.T) {
	t.Parallel()
	if _, err := commonjs.NewFileModule("foo", "foo").Content(); err == nil {
		t.Fatal("was expecting an error")
	}
}

func TestModuleDeps(t *testing.T) {
	t.Parallel()
	m := commonjs.NewModule("bar", []byte(`require('foo')`))
	require, err := m.Require()
	if err != nil {
		t.Fatal(err)
	}
	if len(require) != 1 {
		t.Fatalf("expecting 1 require, got %s", require)
	}
	if require[0] != "foo" {
		t.Fatalf("expecting 1 require foo, got %s", require)
	}
}

func TestModuleDepsMultiple(t *testing.T) {
	t.Parallel()
	m := commonjs.NewModule("bar", []byte(`require('foo') require("baz")`))
	require, err := m.Require()
	if err != nil {
		t.Fatal(err)
	}
	if len(require) != 2 {
		t.Fatalf("expecting 2 requires, got %s", require)
	}
	if require[0] != "foo" {
		t.Fatalf("expecting 2 requires foo, got %s", require)
	}
	if require[1] != "baz" {
		t.Fatalf("expecting 2 requires baz, got %s", require)
	}
}

func TestModulesFromDir(t *testing.T) {
	t.Parallel()
	l, err := commonjs.NewModulesFromDir("_test")
	if err != nil {
		t.Fatal(err)
	}
	if len(l) != 3 {
		t.Fatal("was expecting 3 modules")
	}
	if l[0].Name() != "a/foo" || l[1].Name() != "b/baz" || l[2].Name() != "bar" {
		t.Fatal("did not find expected modules")
	}
}
