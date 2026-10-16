package apiAdapter

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	utilsmsg "github.com/pardnchiu/agenvoy/internal/utils/msg"
)

type Adapter struct {
	prefix string
	apis   map[string]*Document
	client *http.Client
}

func New(prefix string) *Adapter {
	return &Adapter{
		prefix: prefix,
		apis:   make(map[string]*Document),
		client: &http.Client{Transport: http.DefaultTransport.(*http.Transport).Clone()},
	}
}

func (a *Adapter) Builtin(fsys fs.FS, dir string) error {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return utilsmsg.Wrap(utilsmsg.Internal, "fs: ReadDir", "", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		data, err := fs.ReadFile(fsys, fmt.Sprintf("%s/%s", dir, entry.Name()))
		if err != nil {
			slog.Warn("fs: ReadFile",
				slog.String("error", err.Error()))
			continue
		}

		var doc Document
		if err := json.Unmarshal(data, &doc); err != nil {
			slog.Warn("json: Unmarshal",
				slog.String("error", err.Error()))
			continue
		}

		if err := a.check(&doc); err != nil {
			slog.Warn("Adapter: check",
				slog.String("error", err.Error()))
			continue
		}

		a.apis[doc.Name] = &doc
	}
	return nil
}

func (a *Adapter) Load(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return utilsmsg.Wrap(utilsmsg.Internal, "os: ReadDir", "", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		apiPath := filepath.Join(path, entry.Name())
		doc, err := a.load(apiPath)
		if err != nil {
			slog.Warn("failed to load API",
				slog.String("path", apiPath),
				slog.String("error", err.Error()))
			continue
		}
		if doc == nil {
			continue
		}

		a.apis[doc.Name] = doc
	}
	return nil
}

func (a *Adapter) load(path string) (*Document, error) {
	if !go_pkg_filesystem_reader.Exists(path) {
		return nil, nil
	}

	doc, err := go_pkg_filesystem.ReadJSON[Document](path)
	if err != nil {
		return nil, utilsmsg.Wrap(utilsmsg.Invalid, "go_pkg_filesystem.ReadJSON", "", err)
	}

	if err := a.check(&doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

func (a *Adapter) check(doc *Document) error {
	if doc.Name == "" {
		return utilsmsg.New(utilsmsg.Invalid, "name", "required")
	}

	if doc.Description == "" {
		return utilsmsg.New(utilsmsg.Invalid, "description", "required")
	}

	if doc.Endpoint.URL == "" {
		return utilsmsg.New(utilsmsg.Invalid, "endpoint[url]", "required")
	}

	if doc.Endpoint.Method == "" {
		return utilsmsg.New(utilsmsg.Invalid, "endpoint[method]", "required")
	}

	doc.Endpoint.Method = strings.ToUpper(doc.Endpoint.Method)
	switch doc.Endpoint.Method {
	case "GET", "POST", "PUT", "DELETE", "PATCH":
	default:
		return utilsmsg.New(utilsmsg.Invalid, "endpoint[method]", "unsupported method")
	}

	if doc.Endpoint.ContentType == "" {
		doc.Endpoint.ContentType = "json"
	}
	return nil
}

func (a *Adapter) IsExist(name string) bool {
	key := strings.TrimPrefix(name, a.prefix)
	_, ok := a.apis[key]
	return ok
}

func (a *Adapter) GetTools() []map[string]any {
	tools := make([]map[string]any, 0, len(a.apis))
	for _, key := range slices.Sorted(maps.Keys(a.apis)) {
		tools = append(tools, a.apis[key].translate(a.prefix))
	}
	return tools
}

func (a *Adapter) AlwaysAllowNames() []string {
	names := make([]string, 0, len(a.apis))
	for _, key := range slices.Sorted(maps.Keys(a.apis)) {
		if api := a.apis[key]; api.AlwaysAllow {
			names = append(names, a.prefix+api.Name)
		}
	}
	return names
}

func (a *Adapter) ConcurrentNames() []string {
	names := make([]string, 0, len(a.apis))
	for _, key := range slices.Sorted(maps.Keys(a.apis)) {
		if api := a.apis[key]; api.Concurrent {
			names = append(names, a.prefix+api.Name)
		}
	}
	return names
}

func (a *Adapter) Timeouts() map[string]int {
	out := make(map[string]int, len(a.apis))
	for _, key := range slices.Sorted(maps.Keys(a.apis)) {
		if api := a.apis[key]; api.Endpoint.Timeout > 0 {
			out[a.prefix+api.Name] = api.Endpoint.Timeout
		}
	}
	return out
}
