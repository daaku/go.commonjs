package jsh_test

import (
	"github.com/daaku/go.commonjs"
	"github.com/daaku/go.commonjs/jsh"
	"github.com/daaku/go.h"
	"strings"
	"testing"
)

func TestSanity(t *testing.T) {
	t.Parallel()
	var (
		functionName = "fname"
		moduleName   = "mname"
		module       = commonjs.NewScriptModule(moduleName, []byte("js"))
		app          = &commonjs.App{
			MountPath:    "r",
			ContentStore: commonjs.NewMemoryStore(),
			Modules:      []commonjs.Module{module},
		}
		appScripts = &jsh.AppScripts{
			App: app,
			Calls: []jsh.Call{
				jsh.Call{
					Module:   moduleName,
					Function: functionName,
					Args:     []interface{}{1, true, "foo"},
				},
			},
		}
		expectedThings = []string{
			"exports.define = define",
			"execute({\"module\":\"mname\",\"fn\":\"fname\",\"args\":[1,true,\"foo",
			"r/56cc634.js",
		}
		actualHTML, err = h.Render(appScripts)
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range expectedThings {
		if !strings.Contains(actualHTML, e) {
			println(actualHTML)
			t.Fatalf("did not find %s", e)
		}
	}
}
