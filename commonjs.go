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
	"time"
)

var (
	errModuleMissingName = errors.New("module does not have a name")
	reFunCall            = regexp.MustCompile(`require\(['"](.+?)['"]\)`)
)

// A Module is a name, a function returning
type Module struct {
	Name         string
	Content      []byte
	LastModified *time.Time
	Require      []string // required module names
}

// A Provider provides Modules.
type Provider interface {
	Module(name string) (*Module, error)
}

// A CustomProvider allows providing dynamically generated modules.
type CustomProvider struct {
	modules map[string]*Module
}

// Define a module as a JSON data structure. This is useful to inject
// configuration data for example.
func NewJSONModule(name string, v interface{}) (*Module, error) {
	buf := new(bytes.Buffer)
	buf.WriteString("exports.module=")
	if err := json.NewEncoder(buf).Encode(v); err != nil {
		return nil, err
	}
	return &Module{
		Name:    name,
		Content: buf.Bytes(),
	}, nil
}

// Define a module where the content is pulled from a URL.
func NewURLModule(name string, url string) (*Module, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return &Module{
		Name:    name,
		Content: buf,
	}, nil
}

// Define a module where the content is pulled from a file.
func NewFileModule(name string, filename string) (*Module, error) {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return &Module{
		Name:    name,
		Content: buf,
	}, nil
}

// Parse modules from a directory.
func NewModulesFromDir(dirname string) (l []*Module, err error) {
	err = filepath.Walk(
		dirname,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() || filepath.Ext(path) != ".js" {
				return nil
			}
			m, err := NewFileModule(path[len(dirname)+1:len(path)-3], path)
			if err != nil {
				return err
			}
			l = append(l, m)
			return nil
		})
	if err != nil {
		return nil, err
	}
	return l, nil
}

// Find all required modules and populate Require.
func (m *Module) ParseRequire() error {
	calls := reFunCall.FindAllSubmatch(m.Content, -1)
	m.Require = make([]string, len(calls))
	for ix, dep := range calls {
		m.Require[ix] = string(dep[1])
	}
	return nil
}

// Add a Module to the provider.
func (p *CustomProvider) Add(m *Module) error {
	if p.modules == nil {
		p.modules = make(map[string]*Module)
	}
	if m.Name == "" {
		return errModuleMissingName
	}
	if _, exists := p.modules[m.Name]; exists {
		return fmt.Errorf("module %s already exists", m.Name)
	}
	p.modules[m.Name] = m
	return nil
}

func (p *CustomProvider) Module(name string) (*Module, error) {
	if m, ok := p.modules[name]; ok {
		return m, nil
	}
	return nil, fmt.Errorf("module %s was not found", name)
}
