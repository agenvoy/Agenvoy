package session

import (
	"context"
	"fmt"
	"os"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	historyStore "github.com/pardnchiu/agenvoy/internal/runtime/history"
	"github.com/pardnchiu/agenvoy/internal/runtime/torii"
	sessionHistory "github.com/pardnchiu/agenvoy/internal/session/history"
)

func Reset(sessionID string) (int, error) {
	if sessionID == "" {
		return 0, fmt.Errorf("sessionID is required")
	}
	if err := os.Remove(filesystem.HistoryPath(sessionID)); err != nil && !os.IsNotExist(err) {
		return 0, fmt.Errorf("os.Remove [%s]: %w", filesystem.HistoryPath(sessionID), err)
	}

	if err := historyStore.ClearAction(context.Background(), sessionID); err != nil {
		return 0, fmt.Errorf("historyStore.ClearAction: %w", err)
	}

	os.RemoveAll(filesystem.PendingDir(sessionID))

	if err := os.Remove(filesystem.ActionLogPath(sessionID)); err != nil && !os.IsNotExist(err) {
		return 0, fmt.Errorf("os.Remove [%s]: %w", filesystem.ActionLogPath(sessionID), err)
	}

	historyStore.Clear(sessionID)
	sessionHistory.ClearMutex(sessionID)

	db := torii.Remote(torii.DBSessionHist)
	keys := db.Keys(context.Background(), sessionID+":*")
	if len(keys) == 0 {
		return 0, nil
	}
	return db.Del(context.Background(), keys...), nil
}

func ResetAll(sessionID string) (int, error) {
	keys, err := Reset(sessionID)
	if err != nil {
		return keys, err
	}
	if err := os.Remove(filesystem.SummaryPath(sessionID)); err != nil && !os.IsNotExist(err) {
		return keys, fmt.Errorf("os.Remove [%s]: %w", filesystem.SummaryPath(sessionID), err)
	}
	if err := os.Remove(filesystem.SummaryCursorPath(sessionID)); err != nil && !os.IsNotExist(err) {
		return keys, fmt.Errorf("os.Remove [%s]: %w", filesystem.SummaryCursorPath(sessionID), err)
	}
	if err := os.Remove(filesystem.LegacySummaryMetaPath(sessionID)); err != nil && !os.IsNotExist(err) {
		return keys, fmt.Errorf("os.Remove [%s]: %w", filesystem.LegacySummaryMetaPath(sessionID), err)
	}
	return keys, nil
}
