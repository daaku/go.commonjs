package commonjs_test

import (
	"bytes"
	"github.com/daaku/go.commonjs"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
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
	_, err := c.Module(name)
	if err == nil {
		t.Fatal("was expecting an error")
	}
	if !commonjs.IsNotFound(err) {
		t.Fatal("was expecting an IsNotFound to be true")
	}
	if !strings.Contains(err.Error(), name) {
		t.Fatal("was expecting error to contain name")
	}
}

func TestLiteralModule(t *testing.T) {
	t.Parallel()
	const name = "foo"
	const content = "require('baz')"
	m := commonjs.NewModule("foo", []byte(content))
	if m.Name() != name {
		t.Fatal("did not find expected name")
	}
	c, err := m.Content()
	if err != nil {
		t.Fatal(err)
	}
	if string(c) != content {
		t.Fatalf(`did not find expected content, found "%s"`, c)
	}
	r, err := m.Require()
	if err != nil {
		t.Fatal(err)
	}
	if len(r) != 1 || r[0] != "baz" {
		t.Fatal("did not find expected require")
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
	r, err := m.Require()
	if r != nil || err != nil {
		t.Fatal("did not find expected require")
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
	js := []byte("require('foo')")
	s := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write(js)
		}))
	defer s.Close()
	const name = "foo"
	m := commonjs.NewURLModule(name, s.URL+"/")
	if m.Name() != name {
		t.Fatalf("unexpected name %s", m.Name())
	}
	content, err := m.Content()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(content, js) {
		t.Fatalf("did not find expected content")
	}
	r, err := m.Require()
	if err != nil {
		t.Fatal(err)
	}
	if len(r) != 1 || r[0] != "foo" {
		t.Fatal("did not find expected require")
	}
}

func TestURLBackedModuleInvalid(t *testing.T) {
	t.Parallel()
	if _, err := commonjs.NewURLModule("foo", "foo").Content(); err == nil {
		t.Fatal("was expecting an error")
	}
}

func TestURLBackedModuleInvalidRequire(t *testing.T) {
	t.Parallel()
	if _, err := commonjs.NewURLModule("foo", "foo").Require(); err == nil {
		t.Fatalf("did not find expected exception")
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

func TestDirProvider(t *testing.T) {
	t.Parallel()
	const name = "b/baz"
	p := commonjs.NewDirProvider("_test")
	m, err := p.Module(name)
	if err != nil {
		t.Fatal(err)
	}
	if m.Name() != name {
		t.Fatal("did not find expected name")
	}
}

func TestDirProviderNotExist(t *testing.T) {
	t.Parallel()
	p := commonjs.NewDirProvider("_test")
	if _, err := p.Module("xyz"); err == nil {
		t.Fatal("did not find expected error")
	}
}

func TestChainProvider(t *testing.T) {
	t.Parallel()
	const name = "foo"
	c := &commonjs.ChainProvider{}
	c.Add(&commonjs.CustomProvider{})
	p := &commonjs.CustomProvider{}
	c.Add(p)
	m := commonjs.NewModule(name, nil)
	if err := p.Add(m); err != nil {
		t.Fatal(err)
	}

	// ensure it satisfies commonjs.Provider
	var pi commonjs.Provider = c
	m2, err := pi.Module(name)
	if err != nil {
		t.Fatal(err)
	}
	if m2 != m {
		t.Fatal("did not find expected module")
	}
}

func TestChainProviderNotFound(t *testing.T) {
	t.Parallel()
	const name = "foo"
	c := &commonjs.ChainProvider{}
	_, err := c.Module(name)
	if err == nil {
		t.Fatal("was expecting an error")
	}
	if !commonjs.IsNotFound(err) {
		t.Fatal("was expecting an IsNotFound to be true")
	}
}

func TestWrapModule(t *testing.T) {
	t.Parallel()
	const name = "foo"
	const content = "require('baz')"
	const prelude = "prelude"
	const postlude = "postlude"
	m := commonjs.NewModule("foo", []byte(content))
	m = commonjs.WrapModule(m, []byte(prelude), []byte(postlude))
	c, err := m.Content()
	if err != nil {
		t.Fatal(err)
	}
	if string(c) != prelude+content+postlude {
		t.Fatalf("did not find expected content, found %s", c)
	}
}

func TestPackage(t *testing.T) {
	t.Parallel()
	const expected = "require('bar')\nrequire('b/baz')\nrequire('bar')\nbar\n"
	p := commonjs.Package{
		Provider: commonjs.NewDirProvider("_test"),
		Module:   []string{"a/foo", "b/baz"},
	}
	content, err := p.Content()
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != expected {
		t.Fatalf("did not find expected content, instead found %s", content)
	}
}
