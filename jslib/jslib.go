// Package jslib provides often used third party JS libraries.
package jslib

import (
	"github.com/daaku/go.commonjs"
)

var JQuery_1_8_2 = commonjs.NewWrapModule(
	commonjs.NewURLModule(
		"jquery",
		"http://code.jquery.com/jquery-1.8.2.min.js"),
	nil,
	[]byte("module.exports = jQuery.noConflict()"))

var Bootstrap_2_2_2 = commonjs.NewURLModule(
	"bootstrap",
	"https://cdnjs.cloudflare.com/ajax/libs/twitter-bootstrap/2.2.2/bootstrap.min.js")
