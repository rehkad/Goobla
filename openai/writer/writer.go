package writer

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/goobla/goobla/api"
	opentypes "github.com/goobla/goobla/openai/types"
)

type BaseWriter struct {
	gin.ResponseWriter
}

type ChatWriter struct {
	Stream        bool
	StreamOptions *opentypes.StreamOptions
	ID            string
	ToolCallSent  bool
	BaseWriter
}

type CompleteWriter struct {
	Stream        bool
	StreamOptions *opentypes.StreamOptions
	ID            string
	BaseWriter
}

type ListWriter struct {
	BaseWriter
}

type RetrieveWriter struct {
	BaseWriter
	Model string
}

type EmbedWriter struct {
	BaseWriter
	Model string
}

func (w *BaseWriter) writeError(data []byte) (int, error) {
	var serr api.StatusError
	if err := json.Unmarshal(data, &serr); err != nil {
		return 0, err
	}
	w.ResponseWriter.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w.ResponseWriter).Encode(opentypes.NewError(http.StatusInternalServerError, serr.Error())); err != nil {
		return 0, err
	}
	return len(data), nil
}

func (w *ChatWriter) writeResponse(data []byte) (int, error) {
	var r api.ChatResponse
	if err := json.Unmarshal(data, &r); err != nil {
		return 0, err
	}
	if w.Stream {
		c := opentypes.ToChunk(w.ID, r, w.ToolCallSent)
		d, err := json.Marshal(c)
		if err != nil {
			return 0, err
		}
		if !w.ToolCallSent && len(c.Choices) > 0 && len(c.Choices[0].Delta.ToolCalls) > 0 {
			w.ToolCallSent = true
		}
		w.ResponseWriter.Header().Set("Content-Type", "text/event-stream")
		if _, err = w.ResponseWriter.Write([]byte(fmt.Sprintf("data: %s\n\n", d))); err != nil {
			return 0, err
		}
		if r.Done {
			if w.StreamOptions != nil && w.StreamOptions.IncludeUsage {
				u := opentypes.ToUsage(r)
				c.Usage = &u
				c.Choices = []opentypes.ChunkChoice{}
				d, err := json.Marshal(c)
				if err != nil {
					return 0, err
				}
				if _, err = w.ResponseWriter.Write([]byte(fmt.Sprintf("data: %s\n\n", d))); err != nil {
					return 0, err
				}
			}
			if _, err = w.ResponseWriter.Write([]byte("data: [DONE]\n\n")); err != nil {
				return 0, err
			}
		}
		return len(data), nil
	}
	w.ResponseWriter.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w.ResponseWriter).Encode(opentypes.ToChatCompletion(w.ID, r)); err != nil {
		return 0, err
	}
	return len(data), nil
}

func (w *ChatWriter) Write(data []byte) (int, error) {
	if w.ResponseWriter.Status() != http.StatusOK {
		return w.writeError(data)
	}
	return w.writeResponse(data)
}

func (w *CompleteWriter) writeResponse(data []byte) (int, error) {
	var r api.GenerateResponse
	if err := json.Unmarshal(data, &r); err != nil {
		return 0, err
	}
	if w.Stream {
		c := opentypes.ToCompleteChunk(w.ID, r)
		if w.StreamOptions != nil && w.StreamOptions.IncludeUsage {
			c.Usage = &opentypes.Usage{}
		}
		d, err := json.Marshal(c)
		if err != nil {
			return 0, err
		}
		w.ResponseWriter.Header().Set("Content-Type", "text/event-stream")
		if _, err = w.ResponseWriter.Write([]byte(fmt.Sprintf("data: %s\n\n", d))); err != nil {
			return 0, err
		}
		if r.Done {
			if w.StreamOptions != nil && w.StreamOptions.IncludeUsage {
				u := opentypes.ToUsageGenerate(r)
				c.Usage = &u
				c.Choices = []opentypes.CompleteChunkChoice{}
				d, err := json.Marshal(c)
				if err != nil {
					return 0, err
				}
				if _, err = w.ResponseWriter.Write([]byte(fmt.Sprintf("data: %s\n\n", d))); err != nil {
					return 0, err
				}
			}
			if _, err = w.ResponseWriter.Write([]byte("data: [DONE]\n\n")); err != nil {
				return 0, err
			}
		}
		return len(data), nil
	}
	w.ResponseWriter.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w.ResponseWriter).Encode(opentypes.ToCompletion(w.ID, r)); err != nil {
		return 0, err
	}
	return len(data), nil
}

func (w *CompleteWriter) Write(data []byte) (int, error) {
	if w.ResponseWriter.Status() != http.StatusOK {
		return w.writeError(data)
	}
	return w.writeResponse(data)
}

func (w *ListWriter) writeResponse(data []byte) (int, error) {
	var r api.ListResponse
	if err := json.Unmarshal(data, &r); err != nil {
		return 0, err
	}
	w.ResponseWriter.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w.ResponseWriter).Encode(opentypes.ToListCompletion(r)); err != nil {
		return 0, err
	}
	return len(data), nil
}

func (w *ListWriter) Write(data []byte) (int, error) {
	if w.ResponseWriter.Status() != http.StatusOK {
		return w.writeError(data)
	}
	return w.writeResponse(data)
}

func (w *RetrieveWriter) writeResponse(data []byte) (int, error) {
	var r api.ShowResponse
	if err := json.Unmarshal(data, &r); err != nil {
		return 0, err
	}
	w.ResponseWriter.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w.ResponseWriter).Encode(opentypes.ToModel(r, w.Model)); err != nil {
		return 0, err
	}
	return len(data), nil
}

func (w *RetrieveWriter) Write(data []byte) (int, error) {
	if w.ResponseWriter.Status() != http.StatusOK {
		return w.writeError(data)
	}
	return w.writeResponse(data)
}

func (w *EmbedWriter) writeResponse(data []byte) (int, error) {
	var r api.EmbedResponse
	if err := json.Unmarshal(data, &r); err != nil {
		return 0, err
	}
	w.ResponseWriter.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w.ResponseWriter).Encode(opentypes.ToEmbeddingList(w.Model, r)); err != nil {
		return 0, err
	}
	return len(data), nil
}

func (w *EmbedWriter) Write(data []byte) (int, error) {
	if w.ResponseWriter.Status() != http.StatusOK {
		return w.writeError(data)
	}
	return w.writeResponse(data)
}
