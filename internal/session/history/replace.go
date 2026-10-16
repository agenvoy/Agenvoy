package history

import (
	"context"
	"encoding/json"
	"sync"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/runtime/torii"
	utilsmsg "github.com/pardnchiu/agenvoy/internal/utils/msg"
)

func Replace(sessionID string, messages []Record) error {
	if sessionID == "" {
		return utilsmsg.New(utilsmsg.Invalid, "session id", "required")
	}

	mu, _ := muMap.LoadOrStore(sessionID, &sync.Mutex{})
	lock := mu.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()

	historyPath := filesystem.HistoryPath(sessionID)

	raw, err := json.Marshal(messages)
	if err != nil {
		return utilsmsg.Wrap(utilsmsg.Internal, "json.Marshal", "", err)
	}
	if err := go_pkg_filesystem.WriteFile(historyPath, string(raw), 0644); err != nil {
		return utilsmsg.Wrap(utilsmsg.Internal, "go_pkg_filesystem.WriteFile", historyPath, err)
	}

	db := torii.DB(torii.DBSessionHist)
	keys := db.Keys(context.Background(), sessionID+":*")
	if len(keys) > 0 {
		db.Del(context.Background(), keys...)
	}

	return nil
}
