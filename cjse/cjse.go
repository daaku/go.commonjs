package main

import (
	"github.com/daaku/go.commonjs"
	"github.com/daaku/go.commonjs/jslib"
	"github.com/daaku/go.h"
	"log"
	"net/http"
)

var (
	provider = &commonjs.AppProvider{
		Providers: []commonjs.Provider{
			commonjs.NewDirProvider("/home/naitik/usr/go/src/github.com/daaku/go.commonjs/cjse"),
		},
		Modules: []commonjs.Module{
			jslib.JQuery_1_8_2,
		},
	}
	jshandler = &commonjs.Handler{
		BaseURL: "/r/",
	}
	pkg = &commonjs.Package{
		Provider: provider,
		Modules:  []string{"cjse"},
		Handler:  jshandler,
	}
)

func main() {
	http.Handle(jshandler.BaseURL, jshandler)
	http.HandleFunc("/", handler)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	url, err := pkg.URL()
	if err != nil {
		log.Fatal(err)
	}
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
						&h.Script{Src: url},
					},
				},
			},
		})
}
