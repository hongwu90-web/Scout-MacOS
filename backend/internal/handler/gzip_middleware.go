package handler

import (
	"compress/gzip"
	"io"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

type gzipWriter struct {
	gin.ResponseWriter
	writer *gzip.Writer
}

func (g *gzipWriter) Write(data []byte) (int, error) {
	return g.writer.Write(data)
}

func (g *gzipWriter) WriteString(s string) (int, error) {
	return g.writer.Write([]byte(s))
}

var gzipPool = sync.Pool{
	New: func() any {
		w, _ := gzip.NewWriterLevel(io.Discard, gzip.DefaultCompression)
		return w
	},
}

// gzipMiddleware compresses HTTP responses when client sends Accept-Encoding: gzip.
// It skips small payloads (< 1KB) or already compressed formats like images.
func gzipMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") ||
			strings.Contains(c.GetHeader("Connection"), "Upgrade") ||
			strings.Contains(c.GetHeader("Accept"), "text/event-stream") {
			c.Next()
			return
		}

		// Skip paths that are unlikely to benefit (favicons, binary downloads)
		path := c.Request.URL.Path
		if strings.HasSuffix(path, ".png") ||
			strings.HasSuffix(path, ".jpg") ||
			strings.HasSuffix(path, ".jpeg") ||
			strings.HasSuffix(path, ".webp") ||
			strings.HasSuffix(path, ".ico") ||
			strings.HasSuffix(path, ".woff2") {
			c.Next()
			return
		}

		gz := gzipPool.Get().(*gzip.Writer)
		gz.Reset(c.Writer)
		defer func() {
			_ = gz.Close()
			gzipPool.Put(gz)
		}()

		c.Header("Content-Encoding", "gzip")
		c.Header("Vary", "Accept-Encoding")

		c.Writer = &gzipWriter{
			ResponseWriter: c.Writer,
			writer:         gz,
		}

		c.Next()
	}
}
