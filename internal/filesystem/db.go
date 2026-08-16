package filesystem

import (
	"fmt"
	"path/filepath"
	"sync"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_sqlkit "github.com/pardnchiu/go-sqlkit"
	go_sqlkit_core "github.com/pardnchiu/go-sqlkit/core"
)

var (
	dbMu sync.Mutex
	db   *go_sqlkit_core.Connector
)

func OpenDB() error {
	dbMu.Lock()
	defer dbMu.Unlock()

	if db != nil {
		return nil
	}
	if HistoryDBPath == "" {
		return fmt.Errorf("filesystem.Init has not run: no database path")
	}
	if err := go_pkg_filesystem.CheckDir(filepath.Dir(HistoryDBPath), true); err != nil {
		return fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem CheckDir: %w", err)
	}

	c, err := go_sqlkit.New(go_sqlkit_core.Config{Target: go_sqlkit_core.SQLite, Path: HistoryDBPath})
	if err != nil {
		return fmt.Errorf("github.com/pardnchiu/go-sqlkit New: %w", err)
	}

	db = c
	return nil
}

func DB() *go_sqlkit_core.Connector {
	dbMu.Lock()
	defer dbMu.Unlock()
	return db
}

func CloseDB() {
	dbMu.Lock()
	defer dbMu.Unlock()

	if db == nil {
		return
	}
	db.Close()
	db = nil
}
