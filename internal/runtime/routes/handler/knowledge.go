package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/pardnchiu/agenvoy/internal/knowledge"
)

type knowledgeBody struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

func ListKnowledges() gin.HandlerFunc {
	return func(c *gin.Context) {
		records := knowledge.List()
		list := make([]gin.H, 0, len(records))
		for _, record := range records {
			list = append(list, gin.H{
				"name":       record.Name,
				"size":       len(record.Content),
				"updated_at": record.UpdatedAt,
			})
		}
		c.JSON(http.StatusOK, gin.H{"knowledges": list})
	}
}

func GetKnowledge() gin.HandlerFunc {
	return func(c *gin.Context) {
		name, err := knowledge.Key(strings.TrimPrefix(c.Param("name"), "/"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		record, ok := knowledge.Read(name)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "knowledge not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"name": record.Name, "content": record.Content, "updated_at": record.UpdatedAt})
	}
}

func CreateKnowledge() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body knowledgeBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		name, err := knowledge.Name(body.Name, body.Content)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if _, exists := knowledge.Read(name); exists {
			c.JSON(http.StatusConflict, gin.H{"error": "knowledge already exists"})
			return
		}
		if err := knowledge.Write(name, body.Content); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"name": name})
	}
}

func UpdateKnowledge() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			knowledgeBody
			Rename string `json:"rename"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		name, err := knowledge.Key(body.Name)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if _, exists := knowledge.Read(name); !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "knowledge not found"})
			return
		}

		target := name
		if strings.TrimSpace(body.Rename) != "" {
			target, err = knowledge.Name(body.Rename, body.Content)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if target != name {
				if _, exists := knowledge.Read(target); exists {
					c.JSON(http.StatusConflict, gin.H{"error": "knowledge already exists"})
					return
				}
			}
		}

		if err := knowledge.Write(target, body.Content); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if target != name {
			knowledge.Delete(name)
		}
		c.JSON(http.StatusOK, gin.H{"name": target})
	}
}

func DeleteKnowledge() gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Query("name")
		if name == "" {
			var body knowledgeBody
			if err := c.ShouldBindJSON(&body); err == nil {
				name = body.Name
			}
		}

		key, err := knowledge.Key(name)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if _, exists := knowledge.Read(key); !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "knowledge not found"})
			return
		}
		if !knowledge.Delete(key) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}
