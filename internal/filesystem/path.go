package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	"github.com/pardnchiu/go-pkg/filesystem/keychain"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"
)

var (
	filesystemOnce          sync.Once
	AgenvoyDir              string
	ConfigPath              string
	DaemonLogPath           string
	McpPath                 string
	StoreDir                string
	HistoryDBPath           string
	StoreTempDir            string
	SessionsDir             string
	ToolsDir                string
	APIToolsDir             string
	ScriptToolsDir          string
	SystemToolsDir          string
	ExtensionAPIToolsDir    string
	ExtensionScriptToolsDir string
	ScriptToolTrashDir      string
	ErrorsDir               string
	TasksPath               string
	CronsPath               string
	TelegramAuthPath        string
	DiscordAuthPath         string
	SkillsDir               string
	SystemSkillsDir         string
	ScheduleSkillsDir       string
	ScheduleSkillTrashDir   string
	SkillTrashDir           string
	DownloadDir             string
	DownloadTrashDir        string
	SessionsTrashDir        string
	AllowSkillGlobalPath    string
	AllowToolGlobalPath     string
	PromptsDir              string
	KnowledgeDir            string

	WorkAgenvoyDir     string
	WorkAPIToolsDir    string
	WorkScriptToolsDir string
	WorkSkillsDir      string

	// DELETE will deprecated
	LegacyAPIToolsDir        string
	LegacyScriptToolsDir     string
	LegacyWorkAPIToolsDir    string
	LegacyWorkScriptToolsDir string
)

const (
	projectName      = "agenvoy"
	TrashStampLayout = "20060102_150405.000"
)

func Init() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("os.UserHomeDir: %w", err)
	}

	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("os.Getwd: %w", err)
	}

	filesystemOnce.Do(func() {
		AgenvoyDir = filepath.Join(homeDir, ".config", projectName)
		ConfigPath = filepath.Join(AgenvoyDir, "config.json")
		DaemonLogPath = filepath.Join(AgenvoyDir, "daemon.log")
		McpPath = filepath.Join(AgenvoyDir, "mcp.json")

		StoreDir = filepath.Join(AgenvoyDir, ".store")
		HistoryDBPath = filepath.Join(StoreDir, "history.db")
		StoreTempDir = filepath.Join(StoreDir, "temp")
		SessionsDir = filepath.Join(AgenvoyDir, "sessions")
		ToolsDir = filepath.Join(AgenvoyDir, "tools")
		APIToolsDir = filepath.Join(ToolsDir, "api")
		ScriptToolsDir = filepath.Join(ToolsDir, "script")
		SystemToolsDir = filepath.Join(ToolsDir, ".system")
		ExtensionAPIToolsDir = filepath.Join(ToolsDir, ".extension", "api")
		ExtensionScriptToolsDir = filepath.Join(ToolsDir, ".extension", "script")
		ScriptToolTrashDir = filepath.Join(ScriptToolsDir, ".Trash")
		ErrorsDir = filepath.Join(AgenvoyDir, "errors")
		TasksPath = filepath.Join(AgenvoyDir, "tasks.json")
		CronsPath = filepath.Join(AgenvoyDir, "crons.json")
		TelegramAuthPath = filepath.Join(AgenvoyDir, ".telegram")
		DiscordAuthPath = filepath.Join(AgenvoyDir, ".discord")

		SkillsDir = filepath.Join(AgenvoyDir, "skills")
		SystemSkillsDir = filepath.Join(SkillsDir, ".system")
		ScheduleSkillsDir = filepath.Join(SkillsDir, "scheduler")
		ScheduleSkillTrashDir = filepath.Join(ScheduleSkillsDir, ".Trash")
		SkillTrashDir = filepath.Join(SkillsDir, ".Trash")

		LegacyAPIToolsDir = filepath.Join(AgenvoyDir, "api_tools")
		LegacyScriptToolsDir = filepath.Join(AgenvoyDir, "script_tools")

		DownloadDir = filepath.Join(AgenvoyDir, "download")
		DownloadTrashDir = filepath.Join(DownloadDir, ".Trash")
		SessionsTrashDir = filepath.Join(SessionsDir, ".Trash")
		AllowSkillGlobalPath = filepath.Join(AgenvoyDir, "allow_skill")
		AllowToolGlobalPath = filepath.Join(AgenvoyDir, "allow_tool")
		PromptsDir = filepath.Join(AgenvoyDir, "prompts")
		KnowledgeDir = filepath.Join(AgenvoyDir, "knowledge")

		WorkAgenvoyDir = filepath.Join(workDir, ".config", projectName)
		WorkAPIToolsDir = filepath.Join(WorkAgenvoyDir, "tools", "api")
		WorkScriptToolsDir = filepath.Join(WorkAgenvoyDir, "tools", "script")
		WorkSkillsDir = filepath.Join(WorkAgenvoyDir, "skills")

		LegacyWorkAPIToolsDir = filepath.Join(WorkAgenvoyDir, "api_tools")
		LegacyWorkScriptToolsDir = filepath.Join(WorkAgenvoyDir, "script_tools")
	})

	for _, dir := range []string{
		AgenvoyDir,
		DownloadDir,
		DownloadTrashDir,
		SessionsTrashDir,
		ExtensionAPIToolsDir,
		ExtensionScriptToolsDir,
	} {
		if err = go_pkg_filesystem.CheckDir(dir, true); err != nil {
			return fmt.Errorf("go_pkg_filesystem.CheckDir: %w", err)
		}
	}

	keychain.Init(projectName, AgenvoyDir)

	return nil
}

func SessionDir(sessionID string) string {
	return filepath.Join(SessionsDir, sessionID)
}

func SessionConfigPath(sessionID string) string {
	return filepath.Join(SessionDir(sessionID), "config.json")
}

func StatusPath(sessionID string) string {
	return filepath.Join(SessionDir(sessionID), "status.json")
}

func BotPath(sessionID string) string {
	return filepath.Join(SessionDir(sessionID), "bot.json")
}

// DELETE will deprecate, use bot.json replacement
func LegacyBotPath(sessionID string) string {
	return filepath.Join(SessionDir(sessionID), "bot.md")
}

func ActionLogPath(sessionID string) string {
	return filepath.Join(SessionDir(sessionID), "action.log")
}

func UsageLogPath(sessionID string) string {
	return filepath.Join(SessionDir(sessionID), "usage.log")
}

func HistoryPath(sessionID string) string {
	return filepath.Join(SessionDir(sessionID), "history.json")
}

func SummaryPath(sessionID string) string {
	return filepath.Join(SessionDir(sessionID), "summary.json")
}

func SummaryCursorPath(sessionID string) string {
	return filepath.Join(SessionDir(sessionID), ".summary_cursor")
}

func LegacySummaryMetaPath(sessionID string) string {
	return filepath.Join(SessionDir(sessionID), "summary.meta.json")
}

func InputHistoryPath(sessionID string) string {
	return filepath.Join(SessionDir(sessionID), ".cmd_history")
}

func PendingDir(sessionID string) string {
	return filepath.Join(SessionDir(sessionID), "pending")
}

func PendingMetaPath(sessionID, taskHash string) string {
	return filepath.Join(PendingDir(sessionID), taskHash+".json")
}

func TaskHistoryDir(sessionID string) string {
	return filepath.Join(SessionDir(sessionID), "history")
}

func AllowSkillProjectPath(workDir string) string {
	return filepath.Join(workDir, "."+projectName, "allow_skill")
}

func AllowToolPath(workDir string) string {
	return filepath.Join(workDir, "."+projectName, "allow_list")
}

func ScheduleSkillDir(name string) string {
	return filepath.Join(ScheduleSkillsDir, name)
}

func ScheduleSkillPath(name string) string {
	return filepath.Join(ScheduleSkillDir(name), "SKILL.md")
}

func CopyToStoreTemp(src string) (string, error) {
	dst, err := storeTempPath(src)
	if err != nil {
		return "", err
	}
	if err := CopyPath(src, dst); err != nil {
		return "", fmt.Errorf("CopyPath [%s → %s]: %w", src, dst, err)
	}
	return dst, nil
}

func MoveToStoreTemp(src string) (string, error) {
	dst, err := storeTempPath(src)
	if err != nil {
		return "", err
	}
	if err := go_pkg_filesystem.Move(src, dst); err != nil {
		return "", fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem Move [%s → %s]: %w", src, dst, err)
	}
	return dst, nil
}

func storeTempPath(src string) (string, error) {
	if StoreTempDir == "" {
		return "", fmt.Errorf("filesystem.Init has not run: no trash directory to move %s into", src)
	}

	mirrored := filepath.Join(StoreTempDir, strings.TrimPrefix(src, string(filepath.Separator)))
	ext := filepath.Ext(mirrored)
	base := strings.TrimSuffix(mirrored, ext)

	stamp := time.Now().Format(TrashStampLayout)
	dst := fmt.Sprintf("%s_%s%s", base, stamp, ext)
	if !go_pkg_filesystem_reader.Exists(dst) {
		return dst, nil
	}
	return fmt.Sprintf("%s_%s_%d%s", base, stamp, time.Now().UnixNano(), ext), nil
}

func CopyPath(src, dst string) error {
	if !go_pkg_filesystem_reader.IsDir(src) {
		return go_pkg_filesystem.Copy(src, dst)
	}

	return filepath.WalkDir(src, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return fmt.Errorf("filepath.Rel: %w", err)
		}
		target := filepath.Join(dst, rel)

		if entry.IsDir() {
			if err := go_pkg_filesystem.CheckDir(target, true); err != nil {
				return fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem CheckDir [%s]: %w", target, err)
			}
			return nil
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		if err := go_pkg_filesystem.Copy(path, target); err != nil {
			return fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem Copy [%s → %s]: %w", path, target, err)
		}
		return nil
	})
}

func TrashDir(src, trashBase, name string) (string, error) {
	if err := go_pkg_filesystem.CheckDir(trashBase, true); err != nil {
		return "", fmt.Errorf("go_pkg_filesystem.CheckDir [%s]: %w", trashBase, err)
	}
	dst := filepath.Join(trashBase, name)
	if go_pkg_filesystem_reader.Exists(dst) {
		dst = filepath.Join(trashBase, fmt.Sprintf("%s-%d", name, time.Now().Unix()))
	}
	if err := os.Rename(src, dst); err != nil {
		return "", fmt.Errorf("os.Rename [%s → %s]: %w", src, dst, err)
	}
	return dst, nil
}
