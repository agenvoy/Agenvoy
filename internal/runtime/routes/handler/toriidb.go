package handler

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	toriidb "github.com/pardnchiu/toriidb/core/store"

	"github.com/pardnchiu/agenvoy/internal/runtime/torii"
)

type toriiEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func ReadTorii() gin.HandlerFunc {
	return func(c *gin.Context) {
		idx, err := strconv.Atoi(c.Param("db"))
		if err != nil || idx < 0 || idx > 15 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "db must be 0-15"})
			return
		}
		key := strings.TrimPrefix(c.Param("key"), "/")
		if key == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "key is required"})
			return
		}
		db := torii.DB(idx)

		if text := strings.TrimSpace(c.Query("search")); text != "" {
			limit, err := strconv.Atoi(c.Query("limit"))
			if err != nil || limit <= 0 {
				limit = 8
			}
			keys, err := db.VSearch(c.Request.Context(), text, key, limit)
			if err != nil {
				c.JSON(http.StatusOK, gin.H{"keys": []string{}})
				return
			}
			c.JSON(http.StatusOK, gin.H{"keys": keys})
			return
		}

		if !strings.ContainsAny(key, "*?[") {
			entry, found := db.Get(key)
			if !found {
				c.JSON(http.StatusOK, gin.H{"entry": nil})
				return
			}
			c.JSON(http.StatusOK, gin.H{"entry": toriiEntry{Key: key, Value: entry.Value()}})
			return
		}

		keys := db.Keys(key)
		if c.Query("keys") != "" {
			c.JSON(http.StatusOK, gin.H{"keys": keys})
			return
		}

		contains := strings.ToLower(c.Query("contains"))
		after, _ := strconv.ParseInt(c.Query("after"), 10, 64)
		limit, _ := strconv.Atoi(c.Query("limit"))

		list := make([]toriiEntry, 0, len(keys))
		for _, one := range keys {
			entry, found := db.Get(one)
			if !found {
				continue
			}
			if after > 0 && entryUnix(entry) < after {
				continue
			}
			if contains != "" && !strings.Contains(strings.ToLower(entry.Value()), contains) {
				continue
			}
			list = append(list, toriiEntry{Key: one, Value: entry.Value()})
		}
		if limit > 0 && len(list) > limit {
			list = list[len(list)-limit:]
		}
		c.JSON(http.StatusOK, gin.H{"entries": list})
	}
}

func WriteTorii() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			DB       int      `json:"db"`
			Key      string   `json:"key"`
			Keys     []string `json:"keys"`
			Value    *string  `json:"value"`
			ExpireAt *int64   `json:"expire_at"`
			TTL      *int64   `json:"ttl"`
			Vector   bool     `json:"vector"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if body.DB < 0 || body.DB > 15 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "db must be 0-15"})
			return
		}
		db := torii.DB(body.DB)

		if body.TTL != nil {
			if len(body.Keys) == 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "keys is required when ttl is set"})
				return
			}
			for _, one := range body.Keys {
				if err := db.Expire(one, *body.TTL); err != nil {
					slog.Debug("toriidb Expire",
						slog.String("key", one),
						slog.String("error", err.Error()))
				}
			}
			c.JSON(http.StatusOK, gin.H{"ok": true})
			return
		}

		if strings.TrimSpace(body.Key) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "key is required"})
			return
		}

		if body.Value == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "value is required unless ttl is set"})
			return
		}

		if body.Vector {
			if err := db.SetVector(c.Request.Context(), body.Key, *body.Value, torii.SetDefault, body.ExpireAt); err == nil {
				c.JSON(http.StatusOK, gin.H{"ok": true})
				return
			}
		}
		if err := db.Set(body.Key, *body.Value, torii.SetDefault, body.ExpireAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func RemoveTorii() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			DB   int      `json:"db"`
			Keys []string `json:"keys"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if body.DB < 0 || body.DB > 15 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "db must be 0-15"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"removed": torii.DB(body.DB).Del(body.Keys...)})
	}
}

func entryUnix(e *toriidb.Entry) int64 {
	if e.UpdatedAt != nil {
		return *e.UpdatedAt
	}
	return e.CreatedAt
}
