package routes

import (
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"github.com/pardnchiu/agenvoy/page"
)

func registPage(r *gin.Engine) gin.HandlerFunc {
	if dir := os.Getenv("AGENVOY_PAGE_DIR"); dir != "" {
		noStore := func(c *gin.Context) { c.Header("Cache-Control", "no-store") }
		path := filepath.Join(dir, "index.html")
		index := func(c *gin.Context) {
			noStore(c)
			c.File(path)
		}
		r.GET("/", index)
		r.Group("/public", noStore).Static("/", filepath.Join(dir, "public"))
		return index
	}

	public, err := fs.Sub(page.FS, "public")
	if err != nil {
		panic(err)
	}
	r.StaticFS("/public", http.FS(public))

	raw, err := page.FS.ReadFile("index.html")
	if err != nil {
		panic(err)
	}
	index := func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", raw)
	}
	r.GET("/", index)
	return index
}
