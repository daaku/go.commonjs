// Package jsh provides go.h compatible optimized script tags.
package jsh

import (
	"bytes"
	"encoding/json"
	"github.com/daaku/go.commonjs"
	"github.com/daaku/go.h"
)

// A single JavaScript Function call.
type Call struct {
	Module   string        `json:"module"`
	Function string        `json:"fn"`
	Args     []interface{} `json:"args"`
}

// A minimal set of script blocks and efficient loading of an external package
// file.
type AppScripts struct {
	App   *commonjs.App
	Calls []Call
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

	prelude, err := a.App.Prelude()
	if err != nil {
		return nil, err
	}

	src, err := a.App.ModulesURL(modules)
	if err != nil {
		return nil, err
	}

	return &h.Frag{
		&h.Script{
			Inner: &h.Frag{
				h.UnsafeBytes(prelude),
				h.UnsafeBytes(buf.Bytes()),
			},
		},
		&h.Script{
			Src:   src,
			Async: true,
		},
	}, nil
}
