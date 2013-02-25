package commonjs

var prelude = []byte(`
(function(exports) {
  var _payloads = {},
      _modules = {},
      _execute = [],
      _schedule = null;

  function key(name) {
    return '_n_' + name;
  }

  function run() {
    var current = _execute;
    _execute = [];
    for (var i=0, l=current.length; i<l; i++) {
      var c = current[i],
          k = key(c.module);
      if (_modules[k] || _payloads[k]) {
        require(c.module)[c.fn].apply(null, c.args);
      } else {
        execute(c);
      }
    }
  }

  function schedule() {
    if (!_schedule) {
      _schedule = window.setTimeout(
        function() {
          _schedule = null;
          run();
        },
        0
      );
    }
  }

  function execute(c) {
    _execute.push(c);
    schedule();
  }

  function require(name) {
    var k = key(name),
        m = _modules[k];

    if (m) {
      return m.exports;
    }

    var fn = _payloads[k];
    if (!fn) {
      throw 'module ' + name + ' not found';
    }
    delete _payloads[k];
    fn = new Function('require', 'exports', 'module', fn);
    _modules[k] = m = { name: name, exports: {} };
    fn.call(exports, require, m.exports, m);
    return m.exports;
  }

  function define(name, payload) {
    var k = key(name);
    if (k in _payloads || k in _modules) {
      throw 'module ' + name + ' already defined';
    }
    _payloads[k] = payload;
    schedule();
  }

  exports.define = define;
  exports.require = require;
  exports.execute = execute;
})(this);
`)

// Returns the CommonJS/npm style prelude that provides define, require &
// execute functions.
func Prelude() Module {
	return NewScriptModule("prelude", prelude)
}
