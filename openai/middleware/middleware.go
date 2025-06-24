package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/goobla/goobla/api"
	opentypes "github.com/goobla/goobla/openai/types"
	"github.com/goobla/goobla/openai/writer"
)

func ListMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		w := &writer.ListWriter{BaseWriter: writer.BaseWriter{ResponseWriter: c.Writer}}
		c.Writer = w
		c.Next()
	}
}

func RetrieveMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var b bytes.Buffer
		if err := json.NewEncoder(&b).Encode(api.ShowRequest{Name: c.Param("model")}); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, opentypes.NewError(http.StatusInternalServerError, err.Error()))
			return
		}
		c.Request.Body = io.NopCloser(&b)
		w := &writer.RetrieveWriter{BaseWriter: writer.BaseWriter{ResponseWriter: c.Writer}, Model: c.Param("model")}
		c.Writer = w
		c.Next()
	}
}

func CompletionsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req opentypes.CompletionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, opentypes.NewError(http.StatusBadRequest, err.Error()))
			return
		}
		var b bytes.Buffer
		genReq, err := opentypes.FromCompleteRequest(req)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, opentypes.NewError(http.StatusBadRequest, err.Error()))
			return
		}
		if err := json.NewEncoder(&b).Encode(genReq); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, opentypes.NewError(http.StatusInternalServerError, err.Error()))
			return
		}
		c.Request.Body = io.NopCloser(&b)
		w := &writer.CompleteWriter{
			BaseWriter:    writer.BaseWriter{ResponseWriter: c.Writer},
			Stream:        req.Stream,
			ID:            fmt.Sprintf("cmpl-%d", rand.Intn(999)),
			StreamOptions: req.StreamOptions,
		}
		c.Writer = w
		c.Next()
	}
}

func EmbeddingsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req opentypes.EmbedRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, opentypes.NewError(http.StatusBadRequest, err.Error()))
			return
		}
		if req.Input == "" {
			req.Input = []string{""}
		}
		if req.Input == nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, opentypes.NewError(http.StatusBadRequest, "invalid input"))
			return
		}
		if v, ok := req.Input.([]any); ok && len(v) == 0 {
			c.AbortWithStatusJSON(http.StatusBadRequest, opentypes.NewError(http.StatusBadRequest, "invalid input"))
			return
		}
		var b bytes.Buffer
		if err := json.NewEncoder(&b).Encode(api.EmbedRequest{Model: req.Model, Input: req.Input}); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, opentypes.NewError(http.StatusInternalServerError, err.Error()))
			return
		}
		c.Request.Body = io.NopCloser(&b)
		w := &writer.EmbedWriter{BaseWriter: writer.BaseWriter{ResponseWriter: c.Writer}, Model: req.Model}
		c.Writer = w
		c.Next()
	}
}

func ChatMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req opentypes.ChatCompletionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, opentypes.NewError(http.StatusBadRequest, err.Error()))
			return
		}
		if len(req.Messages) == 0 {
			c.AbortWithStatusJSON(http.StatusBadRequest, opentypes.NewError(http.StatusBadRequest, "[] is too short - 'messages'"))
			return
		}
		var b bytes.Buffer
		chatReq, err := opentypes.FromChatRequest(req)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, opentypes.NewError(http.StatusBadRequest, err.Error()))
			return
		}
		if err := json.NewEncoder(&b).Encode(chatReq); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, opentypes.NewError(http.StatusInternalServerError, err.Error()))
			return
		}
		c.Request.Body = io.NopCloser(&b)
		w := &writer.ChatWriter{
			BaseWriter:    writer.BaseWriter{ResponseWriter: c.Writer},
			Stream:        req.Stream,
			ID:            fmt.Sprintf("chatcmpl-%d", rand.Intn(999)),
			StreamOptions: req.StreamOptions,
		}
		c.Writer = w
		c.Next()
	}
}
