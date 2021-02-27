package http

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Context ...
type Context struct {
	*gin.Context
}

// NewContext return a new Context
func NewContext(c *gin.Context) *Context {
	ctx := &Context{
		Context: c,
	}
	return ctx
}

// HandlerFunc ...
type HandlerFunc func(c *Context)

// Handle return gin.HandlerFunc
func Handle(h HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		ct := NewContext(c)
		h(ct)
	}
}

// Jsonify returns json data
func (c *Context) Jsonify(data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"code": http.StatusOK,
		"data": data,
	})
}

// GinLogFormatter log formatter function
func GinLogFormatter(param gin.LogFormatterParams) string {
	return fmt.Sprintf("[%s] %d %s %s %s (%s) %s %s\n",
		param.TimeStamp.Format("20060102 15:04:05"),
		param.StatusCode,
		param.Method,
		param.Path,
		param.Request.Proto,
		param.ClientIP,
		param.Latency,
		param.ErrorMessage,
	)
}
