package writer

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/goobla/goobla/api"
	opentypes "github.com/goobla/goobla/openai/types"
)

func newTestWriter(status int) (gin.ResponseWriter, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Status(status)
	return c.Writer, rec
}

func TestChatWriter(t *testing.T) {
	resp := api.ChatResponse{
		Model:     "test-model",
		CreatedAt: time.Unix(1, 0).UTC(),
		Message:   api.Message{Role: "assistant", Content: "hello"},
		Done:      true,
		Metrics:   api.Metrics{PromptEvalCount: 1, EvalCount: 2},
	}
	data, _ := json.Marshal(resp)

	w, rec := newTestWriter(http.StatusOK)
	cw := &ChatWriter{ID: "chatcmpl-1", BaseWriter: BaseWriter{ResponseWriter: w}}
	if _, err := cw.Write(data); err != nil {
		t.Fatal(err)
	}
	var got opentypes.ChatCompletion
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	want := opentypes.ToChatCompletion("chatcmpl-1", resp)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected response: %#v", got)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("unexpected content type %s", ct)
	}

	serr := api.StatusError{StatusCode: 500, Status: "500", ErrorMessage: "boom"}
	data, _ = json.Marshal(serr)
	w, rec = newTestWriter(http.StatusInternalServerError)
	cw = &ChatWriter{ID: "chatcmpl-1", BaseWriter: BaseWriter{ResponseWriter: w}}
	if _, err := cw.Write(data); err != nil {
		t.Fatal(err)
	}
	var errResp opentypes.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatal(err)
	}
	wantErr := opentypes.NewError(http.StatusInternalServerError, serr.Error())
	if !reflect.DeepEqual(errResp, wantErr) {
		t.Fatalf("unexpected error response: %#v", errResp)
	}
}

func TestCompleteWriter(t *testing.T) {
	resp := api.GenerateResponse{
		Model:     "test-model",
		CreatedAt: time.Unix(2, 0).UTC(),
		Response:  "hello",
		Done:      true,
		Metrics:   api.Metrics{PromptEvalCount: 1, EvalCount: 2},
	}
	data, _ := json.Marshal(resp)

	w, rec := newTestWriter(http.StatusOK)
	cw := &CompleteWriter{ID: "cmpl-1", BaseWriter: BaseWriter{ResponseWriter: w}}
	if _, err := cw.Write(data); err != nil {
		t.Fatal(err)
	}
	var got opentypes.Completion
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	want := opentypes.ToCompletion("cmpl-1", resp)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected response: %#v", got)
	}

	serr := api.StatusError{StatusCode: 400, Status: "400", ErrorMessage: "bad"}
	data, _ = json.Marshal(serr)
	w, rec = newTestWriter(http.StatusBadRequest)
	cw = &CompleteWriter{ID: "cmpl-1", BaseWriter: BaseWriter{ResponseWriter: w}}
	if _, err := cw.Write(data); err != nil {
		t.Fatal(err)
	}
	var errResp opentypes.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatal(err)
	}
	wantErr := opentypes.NewError(http.StatusInternalServerError, serr.Error())
	if !reflect.DeepEqual(errResp, wantErr) {
		t.Fatalf("unexpected error response: %#v", errResp)
	}
}

func TestListWriter(t *testing.T) {
	resp := api.ListResponse{Models: []api.ListModelResponse{{Name: "test-model", ModifiedAt: time.Unix(3, 0).UTC()}}}
	data, _ := json.Marshal(resp)

	w, rec := newTestWriter(http.StatusOK)
	lw := &ListWriter{BaseWriter: BaseWriter{ResponseWriter: w}}
	if _, err := lw.Write(data); err != nil {
		t.Fatal(err)
	}
	var got opentypes.ListCompletion
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	want := opentypes.ToListCompletion(resp)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected response: %#v", got)
	}

	serr := api.StatusError{StatusCode: 404, Status: "404", ErrorMessage: "no"}
	data, _ = json.Marshal(serr)
	w, rec = newTestWriter(http.StatusNotFound)
	lw = &ListWriter{BaseWriter: BaseWriter{ResponseWriter: w}}
	if _, err := lw.Write(data); err != nil {
		t.Fatal(err)
	}
	var errResp opentypes.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatal(err)
	}
	wantErr := opentypes.NewError(http.StatusInternalServerError, serr.Error())
	if !reflect.DeepEqual(errResp, wantErr) {
		t.Fatalf("unexpected error response: %#v", errResp)
	}
}

func TestRetrieveWriter(t *testing.T) {
	resp := api.ShowResponse{ModifiedAt: time.Unix(4, 0).UTC()}
	data, _ := json.Marshal(resp)

	w, rec := newTestWriter(http.StatusOK)
	rw := &RetrieveWriter{Model: "test-model", BaseWriter: BaseWriter{ResponseWriter: w}}
	if _, err := rw.Write(data); err != nil {
		t.Fatal(err)
	}
	var got opentypes.Model
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	want := opentypes.ToModel(resp, "test-model")
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected response: %#v", got)
	}

	serr := api.StatusError{StatusCode: 500, Status: "500", ErrorMessage: "err"}
	data, _ = json.Marshal(serr)
	w, rec = newTestWriter(http.StatusInternalServerError)
	rw = &RetrieveWriter{Model: "test-model", BaseWriter: BaseWriter{ResponseWriter: w}}
	if _, err := rw.Write(data); err != nil {
		t.Fatal(err)
	}
	var errResp opentypes.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatal(err)
	}
	wantErr := opentypes.NewError(http.StatusInternalServerError, serr.Error())
	if !reflect.DeepEqual(errResp, wantErr) {
		t.Fatalf("unexpected error response: %#v", errResp)
	}
}

func TestEmbedWriter(t *testing.T) {
	resp := api.EmbedResponse{Model: "test-model", Embeddings: [][]float32{{0.1}}, PromptEvalCount: 1}
	data, _ := json.Marshal(resp)

	w, rec := newTestWriter(http.StatusOK)
	ew := &EmbedWriter{Model: "test-model", BaseWriter: BaseWriter{ResponseWriter: w}}
	if _, err := ew.Write(data); err != nil {
		t.Fatal(err)
	}
	var got opentypes.EmbeddingList
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	want := opentypes.ToEmbeddingList("test-model", resp)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected response: %#v", got)
	}

	serr := api.StatusError{StatusCode: 500, Status: "500", ErrorMessage: "err"}
	data, _ = json.Marshal(serr)
	w, rec = newTestWriter(http.StatusInternalServerError)
	ew = &EmbedWriter{Model: "test-model", BaseWriter: BaseWriter{ResponseWriter: w}}
	if _, err := ew.Write(data); err != nil {
		t.Fatal(err)
	}
	var errResp opentypes.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatal(err)
	}
	wantErr := opentypes.NewError(http.StatusInternalServerError, serr.Error())
	if !reflect.DeepEqual(errResp, wantErr) {
		t.Fatalf("unexpected error response: %#v", errResp)
	}
}
