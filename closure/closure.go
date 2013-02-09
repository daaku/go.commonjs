package closure

import (
	"encoding/json"
	"net/http"
	"net/url"
)

type CompilationLevel string

const (
	Whitespace            CompilationLevel = "WHITESPACE_ONLY"
	SimpleOptimizations   CompilationLevel = "SIMPLE_OPTIMIZATIONS"
	AdvancedOptimizations CompilationLevel = "ADVANCED_OPTIMIZATIONS"
)

const defaultURL = "http://closure-compiler.appspot.com/compile"

type Closure struct {
	Level CompilationLevel
}

type closureResponse struct {
	CompiledCode string `json:"compiledCode"`
}

func (c *Closure) TransformContent(content []byte) ([]byte, error) {
	l := string(c.Level)
	if l == "" {
		l = string(SimpleOptimizations)
	}
	val := url.Values{}
	val.Add("js_code", string(content))
	val.Add("compilation_level", l)
	val.Add("output_format", "json")
	val.Add("output_info", "compiled_code")
	resp, err := http.PostForm(defaultURL, val)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	cr := new(closureResponse)
	if err = json.NewDecoder(resp.Body).Decode(cr); err != nil {
		return nil, err
	}
	return []byte(cr.CompiledCode), nil
}
