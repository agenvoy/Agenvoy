package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/pardnchiu/agenvoy/internal/note"
)

type noteBody struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

func ListNotes() gin.HandlerFunc {
	return func(c *gin.Context) {
		records := note.List()
		list := make([]gin.H, 0, len(records))
		for _, record := range records {
			list = append(list, gin.H{
				"name":       record.Name,
				"size":       len(record.Content),
				"updated_at": record.UpdatedAt,
			})
		}
		c.JSON(http.StatusOK, gin.H{"notes": list})
	}
}

func GetNote() gin.HandlerFunc {
	return func(c *gin.Context) {
		name, err := note.Key(strings.TrimPrefix(c.Param("name"), "/"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		record, ok := note.Read(name)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "note not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"name": record.Name, "content": record.Content, "updated_at": record.UpdatedAt})
	}
}

func CreateNote() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body noteBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		name, err := note.Create(body.Name, body.Content)
		if err != nil {
			c.JSON(noteStatus(err), gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"name": name})
	}
}

func UpdateNote() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			noteBody
			Rename string `json:"rename"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		target, err := note.Update(body.Name, body.Rename, body.Content)
		if err != nil {
			c.JSON(noteStatus(err), gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"name": target})
	}
}

func noteStatus(err error) int {
	switch {
	case errors.Is(err, note.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, note.ErrExists):
		return http.StatusConflict
	case errors.Is(err, note.ErrWrite):
		return http.StatusInternalServerError
	}
	return http.StatusBadRequest
}

func DeleteNote() gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Query("name")
		if name == "" {
			var body noteBody
			if err := c.ShouldBindJSON(&body); err == nil {
				name = body.Name
			}
		}

		key, err := note.Key(name)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if _, exists := note.Read(key); !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "note not found"})
			return
		}
		if !note.Delete(key) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}
