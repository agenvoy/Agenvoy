package routes

import (
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
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
		r.GET("/sw.js", func(c *gin.Context) {
			c.Header("Cache-Control", "no-cache")
			c.Data(http.StatusOK, "text/javascript; charset=utf-8", []byte(swTeardown))
		})
		r.Group("/public", noStore).Static("/", filepath.Join(dir, "public"))
		r.Static("/vendor", filesystem.VendorDir)
		return index
	}

	public, err := fs.Sub(page.FS, "public")
	if err != nil {
		panic(err)
	}
	r.StaticFS("/public", http.FS(public))
	r.Static("/vendor", filesystem.VendorDir)

	raw, err := page.FS.ReadFile("index.html")
	if err != nil {
		panic(err)
	}
	index := func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", raw)
	}
	r.GET("/", index)

	sw, err := swWorker()
	if err != nil {
		panic(err)
	}
	r.GET("/sw.js", func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache")
		c.Data(http.StatusOK, "text/javascript; charset=utf-8", []byte(sw))
	})
	return index
}

const swTeardown = `self.addEventListener("install", function () {
  self.skipWaiting();
});

self.addEventListener("activate", function (event) {
  event.waitUntil(
    caches
      .keys()
      .then(function (keys) {
        return Promise.all(keys.map(function (key) { return caches.delete(key); }));
      })
      .then(function () { return self.registration.unregister(); })
      .then(function () { return self.clients.matchAll({ type: "window" }); })
      .then(function (list) {
        list.forEach(function (one) { one.navigate(one.url); });
      })
  );
});
`
