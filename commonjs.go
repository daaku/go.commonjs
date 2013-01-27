// Package commonjs provides a Common JS based build system.
package commonjs

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
)

var (
	errModuleMissingName = errors.New("module does not have a name")
	reFunCall            = regexp.MustCompile(`require\(['"](.+?)['"]\)`)
)

// A Module provides exposes some JavaScript.
type Module interface {
	// The name of the module.
	Name() string

	// The script content of the module.
	Content() ([]byte, error)

	// Names of modules required by this module.
	Require() ([]string, error)
}

// A Provider provides Modules.
type Provider interface {
	// Find a named module.
	Module(name string) (Module, error)
}

type literalModule struct {
	name    string
	content []byte
	require []string
}

type jsonModule struct {
	name  string
	value interface{}
}

type urlModule struct {
	name    string
	url     string
	content []byte
	require []string
}

type fileModule struct {
	name    string
	path    string
	content []byte
	require []string
}

// A CustomProvider allows providing dynamically generated modules.
type CustomProvider struct {
	modules map[string]Module
}

// Define a module with the given content.
func NewModule(name string, content []byte) Module {
	return &literalModule{
		name:    name,
		content: content,
	}
}

func (m *literalModule) Name() string {
	return m.name
}

func (m *literalModule) Content() ([]byte, error) {
	return m.content, nil
}

func (m *literalModule) Require() ([]string, error) {
	if m.require == nil {
		var err error
		m.require, err = ParseRequire(m.content)
		if err != nil {
			return nil, err
		}
	}
	return m.require, nil
}

// Define a module as a JSON data structure. This is useful to inject
// configuration data for example.
func NewJSONModule(name string, v interface{}) Module {
	return &jsonModule{
		name:  name,
		value: v,
	}
}

func (m *jsonModule) Name() string {
	return m.name
}

func (m *jsonModule) Content() ([]byte, error) {
	buf := new(bytes.Buffer)
	buf.WriteString("exports.module=")
	if err := json.NewEncoder(buf).Encode(m.value); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (m *jsonModule) Require() ([]string, error) {
	return nil, nil
}

// Define a module where the content is pulled from a URL.
func NewURLModule(name string, url string) Module {
	return &urlModule{
		name: name,
		url:  url,
	}
}

func (m *urlModule) Name() string {
	return m.name
}

func (m *urlModule) Content() ([]byte, error) {
	if m.content == nil {
		resp, err := http.Get(m.url)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		m.content, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
	}
	return m.content, nil
}

func (m *urlModule) Require() ([]string, error) {
	if m.require == nil {
		content, err := m.Content()
		if err != nil {
			return nil, err
		}
		m.require, err = ParseRequire(content)
		if err != nil {
			return nil, err
		}
	}
	return m.require, nil
}

// Define a module where the content is pulled from a file.
func NewFileModule(name string, filename string) Module {
	return &fileModule{
		name: name,
		path: filename,
	}
}

func (m *fileModule) Name() string {
	return m.name
}

func (m *fileModule) Content() ([]byte, error) {
	return ioutil.ReadFile(m.path)
}

func (m *fileModule) Require() ([]string, error) {
	if m.require == nil {
		content, err := m.Content()
		if err != nil {
			return nil, err
		}
		m.require, err = ParseRequire(content)
		if err != nil {
			return nil, err
		}
	}
	return m.require, nil
}

// Parse modules from a directory.
func NewModulesFromDir(dirname string) (l []Module, err error) {
	err = filepath.Walk(
		dirname,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() || filepath.Ext(path) != ".js" {
				return nil
			}
			m := NewFileModule(path[len(dirname)+1:len(path)-3], path)
			l = append(l, m)
			return nil
		})
	if err != nil {
		return nil, err
	}
	return l, nil
}

// Find all required modules.
func ParseRequire(content []byte) ([]string, error) {
	calls := reFunCall.FindAllSubmatch(content, -1)
	l := make([]string, len(calls))
	for ix, dep := range calls {
		l[ix] = string(dep[1])
	}
	return l, nil
}

// Add a Module to the provider.
func (p *CustomProvider) Add(m Module) error {
	if p.modules == nil {
		p.modules = make(map[string]Module)
	}
	if m.Name() == "" {
		return errModuleMissingName
	}
	if _, exists := p.modules[m.Name()]; exists {
		return fmt.Errorf("module %s already exists", m.Name())
	}
	p.modules[m.Name()] = m
	return nil
}

func (p *CustomProvider) Module(name string) (Module, error) {
	if m, ok := p.modules[name]; ok {
		return m, nil
	}
	return nil, fmt.Errorf("module %s was not found", name)
}
