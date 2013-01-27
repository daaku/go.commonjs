package commonjs_test

import (
	"bytes"
	"github.com/daaku/go.commonjs"
	"github.com/daaku/go.subset"
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
	if string(m.Content) != "exports.module={\"answer\":42}\n" {
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

func TestURLBackedModule(t *testing.T) {
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

func TestURLBackedModuleInvalid(t *testing.T) {
	t.Parallel()
	if _, err := commonjs.NewURLModule("foo", "foo"); err == nil {
		t.Fatal("was expecting an error")
	}
}

func TestFileBackedModule(t *testing.T) {
	t.Parallel()
	const name = "foo"
	m, err := commonjs.NewFileModule(name, "commonjs_test.go")
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != name {
		t.Fatalf("unexpected name %s", m.Name)
	}
	if !bytes.Contains(m.Content, []byte("meta meta meta")) {
		t.Fatalf("did not find expected content")
	}
}

func TestFileBackedModuleInvalid(t *testing.T) {
	t.Parallel()
	if _, err := commonjs.NewFileModule("foo", "foo"); err == nil {
		t.Fatal("was expecting an error")
	}
}

func TestModuleDeps(t *testing.T) {
	t.Parallel()
	m := &commonjs.Module{
		Name:    "bar",
		Content: []byte(`require('foo')`),
	}
	if err := m.ParseRequire(); err != nil {
		t.Fatal(err)
	}
	if len(m.Require) != 1 {
		t.Fatalf("expecting 1 require, got %s", m.Require)
	}
	if m.Require[0] != "foo" {
		t.Fatalf("expecting 1 require foo, got %s", m.Require)
	}
}

func TestModuleDepsMultiple(t *testing.T) {
	t.Parallel()
	m := &commonjs.Module{
		Name:    "bar",
		Content: []byte(`require('foo') require("baz")`),
	}
	if err := m.ParseRequire(); err != nil {
		t.Fatal(err)
	}
	if len(m.Require) != 2 {
		t.Fatalf("expecting 2 requires, got %s", m.Require)
	}
	if m.Require[0] != "foo" {
		t.Fatalf("expecting 2 requires foo, got %s", m.Require)
	}
	if m.Require[1] != "baz" {
		t.Fatalf("expecting 2 requires baz, got %s", m.Require)
	}
}

func TestModulesFromDir(t *testing.T) {
	t.Parallel()
	l, err := commonjs.NewModulesFromDir("_test")
	if err != nil {
		t.Fatal(err)
	}
	subset.Assert(
		t,
		[]*commonjs.Module{
			&commonjs.Module{Name: "a/foo"},
			&commonjs.Module{Name: "b/baz"},
			&commonjs.Module{Name: "bar"},
		},
		l)
}
