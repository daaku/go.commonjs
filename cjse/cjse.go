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
	provider = &commonjs.AppProvider{
		Providers: []commonjs.Provider{
			commonjs.NewPackageProvider("github.com/daaku/go.commonjs/cjse"),
		},
		Modules: []commonjs.Module{
			jslib.JQuery_1_8_2,
			jslib.Bootstrap_2_2_2,
		},
	}
	jsURL     = "/r/"
	jsStore   = commonjs.NewMemoryStore()
	jsHandler = commonjs.NewHandler(jsURL, jsStore)
)

func main() {
	http.Handle(jsURL, jsHandler)
	http.HandleFunc("/", handler)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	const id = "cjse-log"
	h.Write(
		w,
		&h.Document{
			Inner: &h.Frag{
				&h.Head{
					Inner: &h.Frag{
						&h.Meta{Charset: "utf-8"},
						&h.Title{h.String("CommonJS Example")},
					},
				},
				&h.Body{
					Inner: &h.Frag{
						&h.H1{ID: id},
						&jsh.AppScripts{
							Provider: provider,
							Handler:  jsHandler,
							Store:    jsStore,
							Calls: []*jsh.Call{
								&jsh.Call{
									Module:   "cjse",
									Function: "log",
									Args:     []interface{}{id},
								},
							},
						},
					},
				},
			},
		})
}
