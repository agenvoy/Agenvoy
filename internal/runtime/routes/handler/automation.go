package handler

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pardnchiu/agenvoy/internal/filesystem/skill"
	"github.com/pardnchiu/agenvoy/internal/runtime"
	historyStore "github.com/pardnchiu/agenvoy/internal/runtime/history"
	sessionManager "github.com/pardnchiu/agenvoy/internal/session"
	schedulerTool "github.com/pardnchiu/agenvoy/internal/tools/scheduler"
)

var scheduleName = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

func GetScheduleSkill() gin.HandlerFunc {
	return func(c *gin.Context) {
		name := strings.TrimPrefix(c.Param("skill"), "/")
		name = strings.TrimSpace(name)
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "skill is required"})
			return
		}
		one, err := skill.LoadSchedule(name)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"skill":       one.Name,
			"name":        one.Name,
			"description": one.Description,
			"body":        one.Body,
		})
	}
}

type scheduleBody struct {
	Target      string   `json:"target"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Content     string   `json:"content"`
	SessionID   string   `json:"session_id"`
	Expressions []string `json:"expressions"`
}

func (b scheduleBody) values() []string {
	list := make([]string, 0, len(b.Expressions))
	for _, one := range b.Expressions {
		one = strings.TrimSpace(one)
		if one == "" {
			continue
		}
		list = append(list, one)
	}
	return list
}

func scheduleTimes(list []string) ([]time.Time, error) {
	out := make([]time.Time, 0, len(list))
	for _, one := range list {
		at, err := schedulerTool.ParseTime(one)
		if err != nil {
			return nil, err
		}
		if !at.After(time.Now()) {
			return nil, fmt.Errorf("already gone: %s", one)
		}
		out = append(out, at)
	}
	return out, nil
}

func scheduleExpressions(list []string) error {
	for _, one := range list {
		if len(strings.Fields(one)) != 5 {
			return fmt.Errorf("expression must be 5 fields '{min} {hour} {dom} {mon} {dow}' (got %q)", one)
		}
	}
	return nil
}

func scheduleSession(target, name, given string) (string, error) {
	if given = strings.TrimSpace(given); given != "" {
		return given, nil
	}
	if target == "cron" {
		crons, err := runtime.LoadCrons()
		if err != nil {
			return "", err
		}
		for _, one := range crons {
			if one.Skill == name && strings.TrimSpace(one.SessionID) != "" {
				return one.SessionID, nil
			}
		}
	} else {
		tasks, err := runtime.LoadTasks()
		if err != nil {
			return "", err
		}
		for _, one := range tasks {
			if one.Skill == name && strings.TrimSpace(one.SessionID) != "" {
				return one.SessionID, nil
			}
		}
	}
	return sessionManager.New("chat-")
}

func prepareSchedule(target string, list []string) (func(name, sessionID string) error, error) {
	if target == "cron" {
		if err := scheduleExpressions(list); err != nil {
			return nil, err
		}
		return func(name, sessionID string) error { return runtime.SetCrons(name, sessionID, list) }, nil
	}
	times, err := scheduleTimes(list)
	if err != nil {
		return nil, err
	}
	return func(name, sessionID string) error { return runtime.SetTasks(name, sessionID, times) }, nil
}

func readScheduleBody(c *gin.Context) (scheduleBody, []string, bool) {
	var body scheduleBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return body, nil, false
	}

	body.Target = strings.ToLower(strings.TrimSpace(body.Target))
	if body.Target != "cron" && body.Target != "task" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target must be 'cron' or 'task'"})
		return body, nil, false
	}

	body.Name = strings.TrimSpace(body.Name)
	if !scheduleName.MatchString(body.Name) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name allows only A-Z a-z 0-9 _ - and must be 1-64 characters, no spaces"})
		return body, nil, false
	}

	if strings.TrimSpace(body.Content) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content is required"})
		return body, nil, false
	}

	list := body.values()
	if len(list) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one " + body.Target + " entry is required"})
		return body, nil, false
	}
	return body, list, true
}

func CreateSchedule() gin.HandlerFunc {
	return func(c *gin.Context) {
		body, list, ok := readScheduleBody(c)
		if !ok {
			return
		}
		if skill.HasSchedule(body.Name) {
			c.JSON(http.StatusConflict, gin.H{"error": "name already taken: " + body.Name})
			return
		}

		commit, err := prepareSchedule(body.Target, list)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		sessionID, err := scheduleSession(body.Target, body.Name, body.SessionID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if err := skill.WriteSchedule(body.Name, body.Description, body.Content); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if err := commit(body.Name, sessionID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"name": body.Name, "session_id": sessionID})
	}
}

func UpdateSchedule() gin.HandlerFunc {
	return func(c *gin.Context) {
		body, list, ok := readScheduleBody(c)
		if !ok {
			return
		}
		if !skill.HasSchedule(body.Name) {
			c.JSON(http.StatusNotFound, gin.H{"error": "schedule skill not found: " + body.Name})
			return
		}

		commit, err := prepareSchedule(body.Target, list)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		sessionID, err := scheduleSession(body.Target, body.Name, body.SessionID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if err := skill.WriteSchedule(body.Name, body.Description, body.Content); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if err := commit(body.Name, sessionID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"name": body.Name, "session_id": sessionID})
	}
}

func ListCrons() gin.HandlerFunc {
	return func(c *gin.Context) {
		crons, err := runtime.LoadCrons()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"crons": crons})
	}
}

func DeleteCron() gin.HandlerFunc {
	return func(c *gin.Context) {
		deleteSchedule(c, "cron")
	}
}

func RunCron() gin.HandlerFunc {
	return func(c *gin.Context) {
		runSchedule(c)
	}
}

func ListTasks() gin.HandlerFunc {
	return func(c *gin.Context) {
		tasks, err := runtime.LoadTasks()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"tasks": tasks})
	}
}

func DeleteTask() gin.HandlerFunc {
	return func(c *gin.Context) {
		deleteSchedule(c, "task")
	}
}

func RunTask() gin.HandlerFunc {
	return func(c *gin.Context) {
		runSchedule(c)
	}
}

func deleteSchedule(c *gin.Context, target string) {
	var body struct {
		Skill string `json:"skill"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	name := strings.TrimSpace(body.Skill)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "skill is required"})
		return
	}

	sessionID, _ := scheduleSessionOf(target, name)

	var removed int
	var err error
	if target == "cron" {
		removed, err = runtime.RemoveCron(name)
	} else {
		removed, err = runtime.RemoveTask(name)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if removed == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": target + " not found"})
		return
	}

	bound, err := scheduleBound(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	trashed := false
	if !bound {
		if err := skill.TrashSchedule(c.Request.Context(), name, historyStore.Meta{SessionID: sessionID}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		trashed = true
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "removed": removed, "trashed": trashed})
}

func scheduleSessionOf(target, name string) (string, error) {
	if target == "cron" {
		crons, err := runtime.LoadCrons()
		if err != nil {
			return "", err
		}
		for _, one := range crons {
			if one.Skill == name {
				return one.SessionID, nil
			}
		}
		return "", nil
	}
	tasks, err := runtime.LoadTasks()
	if err != nil {
		return "", err
	}
	for _, one := range tasks {
		if one.Skill == name {
			return one.SessionID, nil
		}
	}
	return "", nil
}

func scheduleBound(name string) (bool, error) {
	crons, err := runtime.LoadCrons()
	if err != nil {
		return false, err
	}
	for _, one := range crons {
		if one.Skill == name {
			return true, nil
		}
	}
	tasks, err := runtime.LoadTasks()
	if err != nil {
		return false, err
	}
	for _, one := range tasks {
		if one.Skill == name {
			return true, nil
		}
	}
	return false, nil
}

func runSchedule(c *gin.Context) {
	var body struct {
		SessionID string `json:"session_id"`
		Skill     string `json:"skill"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sessionID := strings.TrimSpace(body.SessionID)
	name := strings.TrimSpace(body.Skill)
	if sessionID == "" || name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id and skill are required"})
		return
	}
	go runtime.Fire(sessionID, name)
	c.JSON(http.StatusAccepted, gin.H{"ok": true, "started": true})
}
