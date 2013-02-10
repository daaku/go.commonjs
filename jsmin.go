package commonjs

import (
	"bitbucket.org/maxhauser/jsmin"
	"bytes"
)

// Provides a basic jsmin based transform.
var JSMin Transform = &jsminTransform{}

type jsminTransform struct{}

func (j *jsminTransform) Transform(content []byte) ([]byte, error) {
	out := new(bytes.Buffer)
	jsmin.Run(bytes.NewBuffer(content), out)
	return out.Bytes(), nil
}
