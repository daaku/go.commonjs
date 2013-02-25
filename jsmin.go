package commonjs

import (
	"bitbucket.org/maxhauser/jsmin"
	"bytes"
)

// Provides a basic jsmin based transform.
var JSMin Transform = &jsminTransform{}

type jsminTransform struct{}

func (j *jsminTransform) Transform(m Module) (Module, error) {
	if m.Ext() != jsExt {
		return m, nil
	}

	content, err := m.Content()
	if err != nil {
		return nil, err
	}

	out := new(bytes.Buffer)
	jsmin.Run(bytes.NewBuffer(content), out)
	return NewScriptModule(m.Name(), out.Bytes()), nil
}
