var $ = require('jquery')
require('bootstrap')

exports.log = function(id) {
  $('#' + id).html("in module cjse")
}
