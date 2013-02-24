// Package commonjs provides a CommonJS based build system.
package commonjs

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	hashLen = 7
	ext     = ".js"
	extLen  = len(ext)
)

var (
	errModuleMissingName = errors.New("module does not have a name")
	reFunCall            = regexp.MustCompile(`require\(['"](.+?)['"]\)`)
)

type FileSystem interface {
	Open(path string) (io.ReadCloser, error)
	IsNotExist(err error) bool
}

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

type ByteStore interface {
	// Store a value with the given key.
	Store(key string, value []byte) error

	// Get a stored value. A missing value will return nil, nil.
	Get(key string) ([]byte, error)
}

// Package content may be transformed. This is useful for minification for
// example.
type Transform interface {
	Transform(content []byte) ([]byte, error)
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

type literalModule struct {
	name    string
	content []byte
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

type jsonModule struct {
	name  string
	value interface{}
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

type urlModule struct {
	name    string
	url     string
	content []byte
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

type fileModule struct {
	name    string
	path    string
	content []byte
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

type wrapModule struct {
	Module
	prelude  []byte
	postlude []byte
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

// Provides modules from a directory.
type dirProvider struct {
	path string
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

type fsProvider struct {
	fs FileSystem
}

// Provides a FileSystem backed Provider.
func NewFileSystemProvider(z FileSystem) Provider {
	return &fsProvider{fs: z}
}

func (p *fsProvider) Module(name string) (Module, error) {
	reader, err := p.fs.Open(name + ".js")
	if err != nil {
		if p.fs.IsNotExist(err) {
			return nil, errModuleNotFound(name)
		}
		return nil, err
	}
	defer reader.Close()
	content, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return NewModule(name, content), nil
}

func requireFromModule(m Module) ([]string, error) {
	content, err := m.Content()
	if err != nil {
		return nil, err
	}
	return ParseRequire(content)
}

// Find all required modules in the given content. This essentially looks for
// all require() calls with a string literal as the only argument.
func ParseRequire(content []byte) ([]string, error) {
	calls := reFunCall.FindAllSubmatch(content, -1)
	l := make([]string, len(calls))
	for ix, dep := range calls {
		l[ix] = string(dep[1])
	}
	return l, nil
}

// An App provides a way to source modules, transform code and serves as a
// http.Handler.
type App struct {
	MountPath    string     // URL the http.Handler is serving on
	ContentStore ByteStore  // ByteStore used for storing Content to be served
	Transform    Transform  // optional Transform applied to the code
	Modules      []Module   // optional Modules directly provided by the App
	Providers    []Provider // optional fallback Providers
	prelude      []byte
	packageURLs  map[string]string
}

// Returns a URL for a given set of modules. This caches URLs for a requested
// set of modules.
func (a *App) ModulesURL(modules []string) (string, error) {
	key := strings.Join(modules, "")
	url := a.packageURLs[key]
	if url != "" {
		return url, nil
	}

	content, err := a.content(modules)
	if err != nil {
		return "", err
	}

	sha := sha256.New()
	sha.Write(content)
	hash := fmt.Sprintf("%x", sha.Sum(nil))[:hashLen]
	err = a.ContentStore.Store(hash, content)
	if err != nil {
		return "", err
	}

	url = path.Join("/", a.MountPath, hash+ext)

	if a.packageURLs == nil {
		a.packageURLs = make(map[string]string)
	}
	a.packageURLs[key] = url

	return url, nil
}

// Retrive a Module by name.
func (a *App) Module(name string) (m Module, err error) {
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

// Serves HTTP requests for resources.
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name := path.Base(r.URL.Path)
	nameLen := len(name)
	if nameLen != hashLen+extLen {
		w.WriteHeader(404)
		w.Write([]byte("invalid url\n"))
		return
	}
	content, err := a.ContentStore.Get(name[:nameLen-extLen])
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("error retriving package from store\n"))
		log.Printf("error retriving package from store: %s", err)
	}
	if content == nil {
		w.WriteHeader(404)
		w.Write([]byte("not found\n"))
		return
	}
	w.Header().Add("Content-Type", "text/javascript")
	w.WriteHeader(200)
	w.Write(content)
}

func (a *App) content(modules []string) ([]byte, error) {
	set := make(map[string]bool)
	if err := a.buildDeps(modules, set); err != nil {
		return nil, err
	}

	// write a sorted list of modules for predictable output
	var names []string
	for name, _ := range set {
		names = append(names, name)
	}
	sort.Strings(names)
	out := new(bytes.Buffer)

	var tmp []byte
	for _, name := range names {
		m, err := a.Module(name)
		if err != nil {
			return nil, err
		}
		content, err := m.Content()
		if err != nil {
			return nil, err
		}
		if a.Transform != nil {
			if content, err = a.Transform.Transform(content); err != nil {
				return nil, err
			}
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

func (a *App) buildDeps(require []string, set map[string]bool) error {
	for _, name := range require {
		if set[name] {
			continue
		}
		set[name] = true
		m, err := a.Module(name)
		if err != nil {
			return err
		}
		d, err := m.Require()
		if err != nil {
			return err
		}
		a.buildDeps(d, set)
	}
	return nil
}

// Provides the Prelude, with Transform applied. The result is cached so you
// don't have to.
func (a *App) Prelude() ([]byte, error) {
	if a.prelude == nil {
		var err error
		content := []byte(Prelude())
		if a.Transform != nil {
			if content, err = a.Transform.Transform(content); err != nil {
				return nil, err
			}
		}
		a.prelude = content
	}
	return a.prelude, nil
}

type memoryStore struct {
	data map[string][]byte
}

// Provides a simple in-memory byte store.
func NewMemoryStore() ByteStore {
	return &memoryStore{data: make(map[string][]byte)}
}

func (s *memoryStore) Store(key string, value []byte) error {
	s.data[key] = value
	return nil
}

func (s *memoryStore) Get(key string) ([]byte, error) {
	return s.data[key], nil
}
