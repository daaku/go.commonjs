// Package commonjs provides a Common JS based build system.
package commonjs

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var errModuleMissingName = errors.New("module does not have a name")

// A Module is a name, a function returning
type Module struct {
	Name         string
	Content      []byte
	LastModified *time.Time
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
	buf.WriteString("return ")
	if err := json.NewEncoder(buf).Encode(v); err != nil {
		return nil, err
	}
	return &Module{
		Name:    name,
		Content: buf.Bytes(),
	}, nil
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
