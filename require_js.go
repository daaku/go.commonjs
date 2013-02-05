package commonjs

import (
	"bitbucket.org/maxhauser/jsmin"
	"bytes"
	"strings"
)

var prelude string

func init() {
	raw := `
(function(exports) {
  var _payloads = {},
      _modules = {}

  function key(name) {
    return '_n_' + name
  }

  function require(name) {
    var k = key(name),
        m = _modules[k]

    if (m) return m.exports

    var fn = _payloads[k]
    if (!fn) throw 'module ' + name + ' not found'
    ;delete _payloads[k]
    fn = new Function('require', 'exports', 'module', fn)
    _modules[k] = m = { name: name, exports: {} }
    fn.call(exports, require, m.exports, m)
    return m.exports
  }

  function define(name, payload) {
    var k = key(name)
    if (k in _payloads || k in _modules)
      throw 'module ' + name + ' already defined'
    _payloads[k] = payload
  }

  exports.define = define
  exports.require = require
})(this)
`
	out := new(bytes.Buffer)
	jsmin.Run(strings.NewReader(raw), out)
	prelude = out.String()
}
