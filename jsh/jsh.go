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
	Module   string
	Function string
	Args     []interface{}
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
		buf.WriteString("require(")
		tmp, err = json.Marshal(call.Module)
		if err != nil {
			return nil, err
		}
		buf.Write(tmp)
		buf.WriteString(").")
		buf.WriteString(call.Function)
		buf.WriteString("(")
		last := len(call.Args) - 1
		for iy, arg := range call.Args {
			tmp, err = json.Marshal(arg)
			if err != nil {
				return nil, err
			}
			buf.Write(tmp)
			if iy != last {
				buf.WriteString(",")
			}
		}
		buf.WriteString(");")
	}

	src, err := a.url(modules)
	if err != nil {
		return nil, err
	}

	return &h.Frag{
		&h.Script{Inner: h.Unsafe(commonjs.Prelude())},
		&h.Script{Src: src},
		&h.Script{Inner: h.UnsafeBytes(buf.Bytes())},
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
