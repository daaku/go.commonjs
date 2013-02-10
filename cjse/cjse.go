// Command cjse provides an example of an application built using go.commonjs
// and go.h.
package main

import (
	"github.com/daaku/go.commonjs"
	"github.com/daaku/go.commonjs/jsh"
	"github.com/daaku/go.commonjs/jslib"
	"github.com/daaku/go.h"
	"log"
	"net/http"
)

var (
	jsApp = &commonjs.App{
		MountPath:    "/r/",
		ContentStore: commonjs.NewMemoryStore(),
		Transform:    commonjs.JSMin,
		Providers: []commonjs.Provider{
			commonjs.NewPackageProvider("github.com/daaku/go.commonjs/cjse"),
		},
		Modules: []commonjs.Module{
			jslib.JQuery_1_8_2,
			jslib.Bootstrap_2_2_2,
		},
	}

	elementID = "cjse-log"
	document  = h.Compile(&h.Document{
		Inner: &h.Frag{
			&h.Head{
				Inner: &h.Frag{
					&h.Meta{Charset: "utf-8"},
					&h.Title{h.String("CommonJS Example")},
				},
			},
			&h.Body{
				Inner: &h.Frag{
					&h.H1{ID: elementID},
					&jsh.AppScripts{
						App: jsApp,
						Calls: []jsh.Call{
							jsh.Call{
								Module:   "cjse",
								Function: "log",
								Args:     []interface{}{elementID},
							},
						},
					},
				},
			},
		},
	})
)

func main() {
	http.Handle(jsApp.MountPath, jsApp)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if _, err := h.Write(w, document); err != nil {
			log.Fatal(err)
		}
	})
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
