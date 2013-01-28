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
}

type jsonModule struct {
	name  string
	value interface{}
}

type urlModule struct {
	name    string
	url     string
	content []byte
}

type fileModule struct {
	name    string
	path    string
	content []byte
}

// A CustomProvider allows providing dynamically generated modules.
type CustomProvider struct {
	modules map[string]Module
}

// A Chain Provider proxies module requests to an underlying ordered list of
// Providers.
type ChainProvider struct {
	providers []Provider
}

// Provides modules from a directory.
type dirProvider struct {
	path string
}

type errModuleNotFound string

func (e errModuleNotFound) Error() string {
	return fmt.Sprintf("module %s was not found", string(e))
}

// Check if the error indicates the module was not found.
func IsNotFound(err error) bool {
	_, ok := err.(errModuleNotFound)
	return ok
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
	return requireFromModule(m)
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
	return requireFromModule(m)
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
	return requireFromModule(m)
}

// Provide modules from a directory.
func NewDirProvider(dirname string) Provider {
	return &dirProvider{path: dirname}
}

func (d *dirProvider) Module(name string) (Module, error) {
	filename := filepath.Join(d.path, name+".js")
	if stat, err := os.Stat(filename); os.IsNotExist(err) || stat.IsDir() {
		return nil, errModuleNotFound(name)
	}
	return NewFileModule(name, filename), nil
}

func (c *ChainProvider) Add(p Provider) {
	c.providers = append(c.providers, p)
}

func (c *ChainProvider) Module(name string) (m Module, err error) {
	for _, p := range c.providers {
		m, err = p.Module(name)
		if err == nil {
			return m, err
		}
		if IsNotFound(err) {
			continue
		}
		return nil, err
	}
	return nil, errModuleNotFound(name)
}

func requireFromModule(m Module) ([]string, error) {
	content, err := m.Content()
	if err != nil {
		return nil, err
	}
	return ParseRequire(content)
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
	return nil, errModuleNotFound(name)
}
