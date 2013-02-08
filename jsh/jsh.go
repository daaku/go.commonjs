// Package jsh provides go.h compatible optimized script tags.
package jsh

import (
	"bytes"
	"encoding/json"
	"github.com/daaku/go.commonjs"
	"github.com/daaku/go.h"
	"strings"
)

// A single JavaScript Function call.
type Call struct {
	Module   string        `json:"name"`
	Function string        `json:"fn"`
	Args     []interface{} `json:"args"`
}

// A minimal set of script blocks and efficient loading of an external package
// file.
type AppScripts struct {
	Provider commonjs.Provider
	Handler  commonjs.Handler
	URLStore commonjs.ByteStore
	Modules  []commonjs.Module // this should be used for dynamically generated modules
	Calls    []Call
}

func (a *AppScripts) HTML() (h.HTML, error) {
	buf := new(bytes.Buffer)
	var tmp []byte
	var err error
	modules := make([]string, len(a.Calls))
	for ix, call := range a.Calls {
		modules[ix] = call.Module
		buf.WriteString("execute(")
		tmp, err = json.Marshal(call)
		if err != nil {
			return nil, err
		}
		buf.Write(tmp)
		buf.WriteString(");")
	}

	src, err := a.url(modules)
	if err != nil {
		return nil, err
	}

	return &h.Frag{
		&h.Script{
			Inner: &h.Frag{
				h.Unsafe(commonjs.Prelude()),
				h.UnsafeBytes(buf.Bytes()),
			},
		},
		&h.Script{
			Src:   src,
			Async: true,
		},
	}, nil
}

func (a *AppScripts) url(modules []string) (string, error) {
	key := strings.Join(modules, "")
	raw, err := a.URLStore.Get(key)
	if err != nil {
		return "", err
	}
	if raw != nil {
		return string(raw), nil
	}
	pkg := &commonjs.Package{
		Provider: a,
		Handler:  a.Handler,
		Modules:  modules,
	}
	src, err := pkg.URL()
	if err != nil {
		return "", err
	}
	err = a.URLStore.Store(key, []byte(src))
	if err != nil {
		return "", err
	}
	return src, err
}

func (a *AppScripts) Module(name string) (commonjs.Module, error) {
	for _, m := range a.Modules {
		if m.Name() == name {
			return m, nil
		}
	}
	return a.Provider.Module(name)
}
