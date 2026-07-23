package allowCmd

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
)

func List() []string {
	return filesystem.WhiteList
}

func Append(name string) (bool, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return false, fmt.Errorf("empty command name")
	}
	if slices.Contains(filesystem.WhiteList, name) {
		return false, nil
	}

	dic := map[string]json.RawMessage{}
	if go_pkg_filesystem_reader.Exists(filesystem.ConfigPath) {
		loaded, err := go_pkg_filesystem.ReadJSON[map[string]json.RawMessage](filesystem.ConfigPath)
		if err != nil {
			return false, fmt.Errorf("read config: %w", err)
		}
		dic = loaded
	}
	var current []string
	if data, ok := dic["white_list"]; ok && len(data) > 0 {
		if err := json.Unmarshal(data, &current); err != nil {
			return false, fmt.Errorf("parse white_list: %w", err)
		}
	}
	if slices.Contains(current, name) {
		return false, nil
	}
	current = append(current, name)

	raw, err := json.Marshal(current)
	if err != nil {
		return false, fmt.Errorf("marshal white_list: %w", err)
	}
	dic["white_list"] = raw
	if err := go_pkg_filesystem.CheckDir(filesystem.AgenvoyDir, true); err != nil {
		return false, fmt.Errorf("mkdir agenvoy: %w", err)
	}
	if err := go_pkg_filesystem.WriteJSON(filesystem.ConfigPath, dic, false); err != nil {
		return false, fmt.Errorf("write config: %w", err)
	}
	return true, nil
}
