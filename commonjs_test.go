package commonjs_test

import (
	"bytes"
	"errors"
	"github.com/daaku/go.commonjs"
	"github.com/daaku/go.pkgrsrc/pkgrsrc"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type providerWithError int

func (p providerWithError) Module(name string) (commonjs.Module, error) {
	return nil, errors.New("dummy error")
}

func TestApp(t *testing.T) {
	t.Parallel()
	var (
		name0    = "name0"
		js0      = []byte("js0")
		module0a = commonjs.NewScriptModule(name0, js0)
		module0b = commonjs.NewScriptModule(name0, js0)

		name1   = "name1"
		js1     = []byte("js1")
		module1 = commonjs.NewScriptModule(name1, js1)

		name2   = "name2"
		js2     = []byte("js2")
		module2 = commonjs.NewScriptModule(name2, js2)

		a0 = &commonjs.App{
			Modules: []commonjs.Module{module0a},
		}
		a1 = &commonjs.App{
			Modules: []commonjs.Module{module1},
		}
		a2 = &commonjs.App{
			Providers: []commonjs.Provider{a0, a1},
			Modules:   []commonjs.Module{module0b, module2},
		}

		// ensure it satisfies commonjs.Provider
		p commonjs.Provider = a2

		expected = map[string]commonjs.Module{
			name0: module0b,
			name1: module1,
			name2: module2,
		}
	)

	for en, em := range expected {
		gm, err := p.Module(en)
		if err != nil {
			t.Fatal(err)
		}
		if gm != em {
			t.Fatal("failed to find expected Module")
		}
	}
}

func TestAppModuleNotFound(t *testing.T) {
	t.Parallel()
	const name = "foo"
	a := &commonjs.App{}
	_, err := a.Module(name)
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

func TestAppOtherError(t *testing.T) {
	t.Parallel()
	const name = "foo"
	a := &commonjs.App{
		Providers: []commonjs.Provider{providerWithError(0)},
	}
	_, err := a.Module(name)
	if err == nil {
		t.Fatal("was expecting an error")
	}
	if commonjs.IsNotFound(err) {
		t.Fatal("was expecting an IsNotFound to be false")
	}
}

func TestLiteralModule(t *testing.T) {
	t.Parallel()
	const name = "foo"
	const content = "require('baz')"
	m := commonjs.NewScriptModule("foo", []byte(content))
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
	m := commonjs.NewScriptModule("bar", []byte(`require('foo')`))
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
	m := commonjs.NewScriptModule("bar", []byte(`require('foo') require("baz")`))
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

func TestFileSystemProvider(t *testing.T) {
	t.Parallel()
	const name = "b/baz"
	p := commonjs.NewFileSystemProvider(
		pkgrsrc.New("github.com/daaku/go.commonjs/_test"))
	m, err := p.Module(name)
	if err != nil {
		t.Fatal(err)
	}
	if m.Name() != name {
		t.Fatal("did not find expected name")
	}
}

func TestFileSystemProviderNotExistModule(t *testing.T) {
	t.Parallel()
	p := commonjs.NewFileSystemProvider(
		pkgrsrc.New("github.com/daaku/go.commonjs/_test"))
	if _, err := p.Module("xyz"); err == nil {
		t.Fatal("did not find expected error")
	}
}

func TestFileSystemProviderNotExistPackage(t *testing.T) {
	t.Parallel()
	p := commonjs.NewFileSystemProvider(pkgrsrc.New("foo"))
	if _, err := p.Module("xyz"); err == nil {
		t.Fatal("did not find expected error")
	}
}

func TestWrapModule(t *testing.T) {
	t.Parallel()
	const name = "foo"
	const content = "require('baz')"
	const prelude = "prelude"
	const postlude = "postlude"
	m := commonjs.NewScriptModule("foo", []byte(content))
	m = commonjs.NewWrapModule(m, []byte(prelude), []byte(postlude))
	c, err := m.Content()
	if err != nil {
		t.Fatal(err)
	}
	if string(c) != prelude+content+postlude {
		t.Fatalf("did not find expected content, found %s", c)
	}
}

func TestAppURLAndContent(t *testing.T) {
	t.Parallel()
	const expectedURL = "/r/a102771.js"
	const expectedContent = `define("a/foo","require('bar')\nrequire('b/baz')");
define("b/baz","require('bar')");
define("bar","bar");
`
	p := &commonjs.App{
		MountPath:    "r",
		Providers:    []commonjs.Provider{commonjs.NewDirProvider("_test")},
		ContentStore: commonjs.NewMemoryStore(),
	}
	actualURL, err := p.ModulesURL([]string{"a/foo", "b/baz"})
	if err != nil {
		t.Fatal(err)
	}
	if actualURL != expectedURL {
		t.Fatalf("did not find expected url, instead found %s", actualURL)
	}
	w := httptest.NewRecorder()
	p.ServeHTTP(w, &http.Request{URL: &url.URL{Path: actualURL}})
	content := w.Body.Bytes()
	if string(content) != expectedContent {
		println(string(content))
		t.Fatal("did not find expected content, instead found content above")
	}
}

func TestAppURLLengthError(t *testing.T) {
	t.Parallel()
	p := &commonjs.App{
		MountPath:    "r",
		Providers:    []commonjs.Provider{commonjs.NewDirProvider("_test")},
		ContentStore: commonjs.NewMemoryStore(),
	}
	w := httptest.NewRecorder()
	p.ServeHTTP(w, &http.Request{URL: &url.URL{Path: "/foo"}})
	if w.Code != 404 {
		t.Fatalf("was expecting a 404, got %s", w.Code)
	}
	if bytes.Compare(w.Body.Bytes(), []byte("invalid url\n")) != 0 {
		println(string(w.Body.Bytes()))
		t.Fatalf("did not find expected content")
	}
}

func TestAppURLPackageMissingError(t *testing.T) {
	t.Parallel()
	p := &commonjs.App{
		MountPath:    "r",
		Providers:    []commonjs.Provider{commonjs.NewDirProvider("_test")},
		ContentStore: commonjs.NewMemoryStore(),
	}
	w := httptest.NewRecorder()
	p.ServeHTTP(w, &http.Request{URL: &url.URL{Path: "/r/d613ea9.js"}})
	if w.Code != 404 {
		println(string(w.Body.Bytes()))
		t.Fatalf("was expecting a 500, got %s", w.Code)
	}

	expected := []byte("not found\n")
	if bytes.Compare(w.Body.Bytes(), expected) != 0 {
		println(string(w.Body.Bytes()))
		t.Fatalf("did not find expected content")
	}
}

type testTransform int

var testTransformContent = []byte("expected")

func (t testTransform) Transform(m commonjs.Module) (commonjs.Module, error) {
	return commonjs.NewScriptModule(m.Name(), testTransformContent), nil
}

func TestAppAppliesTransform(t *testing.T) {
	t.Parallel()
	var (
		name   = "name"
		module = commonjs.NewScriptModule(name, []byte("js"))
		app    = &commonjs.App{
			MountPath:    "r",
			ContentStore: commonjs.NewMemoryStore(),
			Modules:      []commonjs.Module{module},
			Transform:    testTransform(0),
		}
	)

	actualURL, err := app.ModulesURL([]string{name})
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	app.ServeHTTP(w, &http.Request{URL: &url.URL{Path: actualURL}})
	actual := w.Body.Bytes()
	if bytes.Compare([]byte("define(\"name\",\"expected\");\n"), actual) != 0 {
		println(string(actual))
		t.Fatal("failed to find expected content")
	}
}

func TestAppAppliesTransformToPrelude(t *testing.T) {
	t.Parallel()
	var app = &commonjs.App{
		MountPath: "r",
		Transform: testTransform(0),
	}

	actual, err := app.ScriptPrelude()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Compare([]byte(testTransformContent), actual) != 0 {
		println(string(actual))
		t.Fatal("failed to find expected content")
	}
}

func TestJSMin(t *testing.T) {
	t.Parallel()
	m, err := commonjs.JSMin.Transform(
		commonjs.NewScriptModule("foo", []byte("function foo ( ) { return 1 ; }")))
	if err != nil {
		t.Fatal(err)
	}
	actual, err := m.Content()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(actual, []byte("\nfunction foo(){return 1;}")) != 0 {
		println(string(actual))
		t.Fatal("did not find expected content")
	}
}
