package session

import (
	"context"
	"os"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	runtimestore "github.com/pardnchiu/agenvoy/internal/runtime/store"
	"github.com/pardnchiu/agenvoy/internal/runtime/torii"
	sessionhistory "github.com/pardnchiu/agenvoy/internal/session/history"
	utilsmsg "github.com/pardnchiu/agenvoy/internal/utils/msg"
)

func Reset(sessionID string, withSummary bool) (int, error) {
	if sessionID == "" {
		return 0, utilsmsg.New(utilsmsg.Invalid, "session id", "required")
	}
	if err := os.Remove(filesystem.HistoryPath(sessionID)); err != nil && !os.IsNotExist(err) {
		return 0, utilsmsg.Wrap(utilsmsg.Internal, "os.Remove", filesystem.HistoryPath(sessionID), err)
	}

	if err := runtimestore.ClearAction(context.Background(), sessionID); err != nil {
		return 0, utilsmsg.Trace("runtimestore.ClearAction", "", err)
	}

	os.RemoveAll(filesystem.PendingDir(sessionID))

	if err := os.Remove(filesystem.ActionLogPath(sessionID)); err != nil && !os.IsNotExist(err) {
		return 0, utilsmsg.Wrap(utilsmsg.Internal, "os.Remove", filesystem.ActionLogPath(sessionID), err)
	}

	runtimestore.Clear(sessionID)
	sessionhistory.ClearMutex(sessionID)

	if !withSummary {
		if err := os.Remove(filesystem.SummaryPath(sessionID)); err != nil && !os.IsNotExist(err) {
			return 0, utilsmsg.Wrap(utilsmsg.Internal, "os.Remove", filesystem.SummaryPath(sessionID), err)
		}
		if err := os.Remove(filesystem.SummaryCursorPath(sessionID)); err != nil && !os.IsNotExist(err) {
			return 0, utilsmsg.Wrap(utilsmsg.Internal, "os.Remove", filesystem.SummaryCursorPath(sessionID), err)
		}
	}

	db := torii.DB(torii.DBSessionHist)
	keys := db.Keys(context.Background(), sessionID+":*")
	if len(keys) == 0 {
		return 0, nil
	}
	return db.Del(context.Background(), keys...), nil
}
