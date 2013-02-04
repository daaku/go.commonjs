// Package commonjs provides a CommonJS based build system.
package commonjs

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
)

var (
	errModuleMissingName = errors.New("module does not have a name")
	reFunCall            = regexp.MustCompile(`require\(['"](.+?)['"]\)`)
	hashLen              = 7
	ext                  = ".js"
	extLen               = len(ext)
)

// A Module provides some JavaScript.
type Module interface {
	// The name of the module.
	Name() string

	// The script content of the module.
	Content() ([]byte, error)

	// Names of modules required by this module.
	Require() ([]string, error)
}

// A Provider provides Modules.
type Provider interface {
	// Find a named module.
	Module(name string) (Module, error)
}

type literalModule struct {
	name    string
	content []byte
}

type jsonModule struct {
	name  string
	value interface{}
}

type urlModule struct {
	name    string
	url     string
	content []byte
}

type fileModule struct {
	name    string
	path    string
	content []byte
}

type wrapModule struct {
	Module
	prelude  []byte
	postlude []byte
}

// Provides modules from a directory.
type dirProvider struct {
	path string
}

// An AppProvider provides zero or more Modules and zero or more fallback
// Providers. The preference order is Modules then first Providers with module.
type AppProvider struct {
	Modules   []Module
	Providers []Provider
}

// A Package delivers a set of requested modules and it's dependencies.
type Package struct {
	Provider Provider // The Provider to pull Modules from.
	Modules  []string // The Modules to include in the Package.
	Handler  Handler  // The Handler to store content, generate & serve URLs.
	Prelude  bool     // If true will include the Prelude.
	url      string
}

// A http handler with the ability to add content to be served.
type Handler interface {
	http.Handler

	// Make some content available via this handler. It returns the URL for the
	// content.
	Add(content []byte) string
}

type memoryHandler struct {
	baseURL string
	cache   map[string][]byte
}

type errModuleNotFound string

func (e errModuleNotFound) Error() string {
	return fmt.Sprintf("module %s was not found", string(e))
}

// Check if the error indicates the module was not found.
func IsNotFound(err error) bool {
	_, ok := err.(errModuleNotFound)
	return ok
}

// Define a module with the given content.
func NewModule(name string, content []byte) Module {
	return &literalModule{
		name:    name,
		content: content,
	}
}

func (m *literalModule) Name() string {
	return m.name
}

func (m *literalModule) Content() ([]byte, error) {
	return m.content, nil
}

func (m *literalModule) Require() ([]string, error) {
	return requireFromModule(m)
}

// Define a module as a JSON data structure. This is useful to inject
// configuration data for example.
func NewJSONModule(name string, v interface{}) Module {
	return &jsonModule{
		name:  name,
		value: v,
	}
}

func (m *jsonModule) Name() string {
	return m.name
}

func (m *jsonModule) Content() ([]byte, error) {
	buf := new(bytes.Buffer)
	buf.WriteString("exports.module=")
	if err := json.NewEncoder(buf).Encode(m.value); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (m *jsonModule) Require() ([]string, error) {
	return nil, nil
}

// Define a module where the content is pulled from a URL.
func NewURLModule(name string, url string) Module {
	return &urlModule{
		name: name,
		url:  url,
	}
}

func (m *urlModule) Name() string {
	return m.name
}

func (m *urlModule) Content() ([]byte, error) {
	if m.content == nil {
		resp, err := http.Get(m.url)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		m.content, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
	}
	return m.content, nil
}

func (m *urlModule) Require() ([]string, error) {
	return requireFromModule(m)
}

// Define a module where the content is pulled from a file.
func NewFileModule(name string, filename string) Module {
	return &fileModule{
		name: name,
		path: filename,
	}
}

func (m *fileModule) Name() string {
	return m.name
}

func (m *fileModule) Content() ([]byte, error) {
	return ioutil.ReadFile(m.path)
}

func (m *fileModule) Require() ([]string, error) {
	return requireFromModule(m)
}

// Provide modules from a directory.
func NewDirProvider(dirname string) Provider {
	return &dirProvider{path: dirname}
}

func (d *dirProvider) Module(name string) (Module, error) {
	filename := filepath.Join(d.path, name+".js")
	if stat, err := os.Stat(filename); os.IsNotExist(err) || stat.IsDir() {
		return nil, errModuleNotFound(name)
	}
	return NewFileModule(name, filename), nil
}

func requireFromModule(m Module) ([]string, error) {
	content, err := m.Content()
	if err != nil {
		return nil, err
	}
	return ParseRequire(content)
}

// Find all required modules.
func ParseRequire(content []byte) ([]string, error) {
	calls := reFunCall.FindAllSubmatch(content, -1)
	l := make([]string, len(calls))
	for ix, dep := range calls {
		l[ix] = string(dep[1])
	}
	return l, nil
}

func (a *AppProvider) Module(name string) (m Module, err error) {
	for _, m = range a.Modules {
		if m.Name() == name {
			return m, nil
		}
	}

	for _, p := range a.Providers {
		m, err = p.Module(name)
		if err == nil {
			return m, err
		}
		if IsNotFound(err) {
			continue
		}
		return nil, err
	}
	return nil, errModuleNotFound(name)
}

// Wraps another module and provides the ability to supply a prelude and
// postlude. This is useful to wrap non CommonJS modules.
func NewWrapModule(m Module, prelude, postlude []byte) Module {
	return &wrapModule{
		Module:   m,
		prelude:  prelude,
		postlude: postlude,
	}
}

func (w *wrapModule) Content() ([]byte, error) {
	c, err := w.Module.Content()
	if err != nil {
		return nil, err
	}
	return bytes.Join([][]byte{w.prelude, c, w.postlude}, nil), nil
}

// Returns the content (including the Prelude if configured).
func (p *Package) Content() ([]byte, error) {
	set := make(map[string]bool)
	if err := p.buildDeps(p.Modules, set); err != nil {
		return nil, err
	}

	// write a sorted list of modules for predictable output
	var names []string
	for name, _ := range set {
		names = append(names, name)
	}
	sort.Strings(names)

	out := new(bytes.Buffer)

	if p.Prelude {
		out.WriteString(Prelude())
	}

	var tmp []byte
	for _, name := range names {
		m, err := p.Provider.Module(name)
		if err != nil {
			return nil, err
		}
		content, err := m.Content()
		if err != nil {
			return nil, err
		}

		out.WriteString("define(")
		if tmp, err = json.Marshal(m.Name()); err != nil {
			return nil, err
		}
		out.Write(tmp)
		out.WriteString(",")
		if tmp, err = json.Marshal(string(bytes.TrimSpace(content))); err != nil {
			return nil, err
		}
		out.Write(tmp)
		out.WriteString(");\n")
	}
	return out.Bytes(), nil
}

func (p *Package) buildDeps(require []string, set map[string]bool) error {
	for _, name := range require {
		if set[name] {
			continue
		}
		set[name] = true
		m, err := p.Provider.Module(name)
		if err != nil {
			return err
		}
		d, err := m.Require()
		if err != nil {
			return err
		}
		p.buildDeps(d, set)
	}
	return nil
}

// Provides a URL for this Package.
func (p *Package) URL() (string, error) {
	if p.url == "" {
		content, err := p.Content()
		if err != nil {
			return "", err
		}
		p.url = p.Handler.Add(content)
	}
	return p.url, nil
}

// Create a new handler that caches content in memory with the given base URL.
func NewMemoryHandler(url string) Handler {
	return &memoryHandler{
		baseURL: url,
		cache:   make(map[string][]byte),
	}
}

func (h *memoryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name := path.Base(r.URL.Path)
	nameLen := len(name)
	if nameLen != hashLen+extLen {
		w.WriteHeader(404)
		w.Write([]byte("invalid url\n"))
		return
	}
	content, found := h.cache[name[:nameLen-extLen]]
	if !found {
		w.WriteHeader(404)
		w.Write([]byte("not found\n"))
		return
	}
	w.Header().Add("Content-Type", "text/javascript")
	w.WriteHeader(200)
	w.Write(content)
}

func (h *memoryHandler) Add(content []byte) string {
	s := sha256.New()
	s.Write(content)
	name := fmt.Sprintf("%x", s.Sum(nil))[:hashLen]
	h.cache[name] = content
	return path.Join("/", h.baseURL, name+ext)
}

// Returns the CommonJS/npm style prelude that provides define &
// require functions.
func Prelude() string {
	return prelude
}
