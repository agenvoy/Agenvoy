package handler

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/pardnchiu/agenvoy/internal/runtime/webui"
)

func WebuiProxy() gin.HandlerFunc {
	return func(c *gin.Context) {
		target, err := url.Parse(webui.URL)
		if err != nil {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		proxy := httputil.NewSingleHostReverseProxy(target)
		proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, _ error) {
			w.WriteHeader(http.StatusNotFound)
		}

		req := c.Request.Clone(c.Request.Context())
		if trimmed, ok := strings.CutPrefix(req.URL.Path, "/webui"); ok {
			if trimmed == "" {
				trimmed = "/"
			}
			req.URL.Path = trimmed
		}
		proxy.ServeHTTP(c.Writer, req)
	}
}
