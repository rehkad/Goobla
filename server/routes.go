package server

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"io"
	"io/fs"
	"log/slog"
	"math"
	"net"
	"net/http"
	_ "net/http/pprof"
	"net/netip"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"golang.org/x/image/webp"
	"golang.org/x/sync/errgroup"

	"github.com/goobla/goobla/api"
	"github.com/goobla/goobla/discover"
	"github.com/goobla/goobla/envconfig"
	"github.com/goobla/goobla/fs/ggml"
	"github.com/goobla/goobla/llm"
	"github.com/goobla/goobla/logutil"
	openaimid "github.com/goobla/goobla/openai/middleware"
	"github.com/goobla/goobla/server/internal/client/goobla"
	"github.com/goobla/goobla/server/internal/registry"
	"github.com/goobla/goobla/template"
	"github.com/goobla/goobla/thinking"
	"github.com/goobla/goobla/tools"
	"github.com/goobla/goobla/types/errtypes"
	"github.com/goobla/goobla/types/model"
	"github.com/goobla/goobla/version"
)

func experimentEnabled(name string) bool {
	return slices.Contains(strings.Split(os.Getenv("GOOBLA_EXPERIMENT"), ","), name)
}

var useClient2 = experimentEnabled("client2")

func newHTTPServer(h http.Handler) *http.Server {
	return &http.Server{
		Handler:      h,
		ReadTimeout:  envconfig.ReadTimeout(),
		WriteTimeout: envconfig.WriteTimeout(),
		IdleTimeout:  envconfig.IdleTimeout(),
	}
}

var mode string = gin.DebugMode

type Server struct {
	addr  net.Addr
	sched *Scheduler
}

func init() {
	switch mode {
	case gin.DebugMode:
	case gin.ReleaseMode:
	case gin.TestMode:
	default:
		mode = gin.DebugMode
	}

	gin.SetMode(mode)
}

var (
	errRequired    = errors.New("is required")
	errBadTemplate = errors.New("template error")
)

func modelOptions(model *Model, requestOpts map[string]any) (api.Options, error) {
	opts := api.DefaultOptions()
	if err := opts.FromMap(model.Options); err != nil {
		return api.Options{}, err
	}

	if err := opts.FromMap(requestOpts); err != nil {
		return api.Options{}, err
	}

	return opts, nil
}

// scheduleRunner schedules a runner after validating inputs such as capabilities and model options.
// It returns the allocated runner, model instance, and consolidated options if successful and error otherwise.
func (s *Server) scheduleRunner(ctx context.Context, name string, caps []model.Capability, requestOpts map[string]any, keepAlive *api.Duration) (llm.LlamaServer, *Model, *api.Options, error) {
	if name == "" {
		return nil, nil, nil, fmt.Errorf("model %w", errRequired)
	}

	model, err := GetModel(name)
	if err != nil {
		return nil, nil, nil, err
	}

	if slices.Contains(model.Config.ModelFamilies, "mllama") && len(model.ProjectorPaths) > 0 {
		return nil, nil, nil, fmt.Errorf("'llama3.2-vision' is no longer compatible with your version of Goobla and has been replaced by a newer version. To re-download, run 'goobla pull llama3.2-vision'")
	}

	if err := model.CheckCapabilities(caps...); err != nil {
		return nil, nil, nil, fmt.Errorf("%s %w", name, err)
	}

	opts, err := modelOptions(model, requestOpts)
	if err != nil {
		return nil, nil, nil, err
	}

	runnerCh, errCh := s.sched.GetRunner(ctx, model, opts, keepAlive)
	var runner *runnerRef
	select {
	case runner = <-runnerCh:
	case err = <-errCh:
		return nil, nil, nil, err
	}

	return runner.llama, model, &opts, nil
}

func (s *Server) GenerateHandler(c *gin.Context) {
	checkpointStart := time.Now()
	var req api.GenerateRequest
	if err := c.ShouldBindJSON(&req); errors.Is(err, io.EOF) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "missing request body"})
		return
	} else if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	name := model.ParseName(req.Model)
	if !name.IsValid() {
		// Ideally this is "invalid model name" but we're keeping with
		// what the API currently returns until we can change it.
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("model '%s' not found", req.Model)})
		return
	}

	// We cannot currently consolidate this into GetModel because all we'll
	// induce infinite recursion given the current code structure.
	name, err := getExistingName(name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("model '%s' not found", req.Model)})
		return
	}

	m, err := GetModel(name.String())
	if err != nil {
		switch {
		case errors.Is(err, fs.ErrNotExist):
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("model '%s' not found", req.Model)})
		case err.Error() == errtypes.InvalidModelNameErrMsg:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// expire the runner
	if req.Prompt == "" && req.KeepAlive != nil && int(req.KeepAlive.Seconds()) == 0 {
		s.sched.expireRunner(m)

		c.JSON(http.StatusOK, api.GenerateResponse{
			Model:      req.Model,
			CreatedAt:  time.Now().UTC(),
			Response:   "",
			Done:       true,
			DoneReason: "unload",
		})
		return
	}

	if req.Raw && (req.Template != "" || req.System != "" || len(req.Context) > 0) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "raw mode does not support template, system, or context"})
		return
	}

	caps := []model.Capability{model.CapabilityCompletion}
	if req.Suffix != "" {
		caps = append(caps, model.CapabilityInsert)
	}
	if req.Think != nil && *req.Think {
		caps = append(caps, model.CapabilityThinking)
		// TODO(drifkin): consider adding a warning if it's false and the model
		// doesn't support thinking. It's not strictly required, but it can be a
		// hint that the user is on an older qwen3/r1 model that doesn't have an
		// updated template supporting thinking
	}

	r, m, opts, err := s.scheduleRunner(c.Request.Context(), name.String(), caps, req.Options, req.KeepAlive)
	if errors.Is(err, errCapabilityCompletion) {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%q does not support generate", req.Model)})
		return
	} else if err != nil {
		handleScheduleError(c, req.Model, err)
		return
	}

	if req.Think != nil && !*req.Think && !slices.Contains(m.Capabilities(), model.CapabilityThinking) {
		msg := "model does not support thinking output"
		if m.Config.ModelFamily == "qwen3" || model.ParseName(m.Name).Model == "deepseek-r1" {
			msg += "; pull the model again to get the latest version with full thinking support"
		}
		slog.Warn(msg, "model", req.Model)
	}

	checkpointLoaded := time.Now()

	// load the model
	if req.Prompt == "" {
		c.JSON(http.StatusOK, api.GenerateResponse{
			Model:      req.Model,
			CreatedAt:  time.Now().UTC(),
			Done:       true,
			DoneReason: "load",
		})
		return
	}

	if slices.Contains(m.Config.ModelFamilies, "mllama") && len(req.Images) > 1 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "this model only supports one image while more than one image requested"})
		return
	}

	images := make([]llm.ImageData, len(req.Images))
	for i := range req.Images {
		images[i] = llm.ImageData{ID: i, Data: req.Images[i]}
	}

	prompt := req.Prompt
	if !req.Raw {
		tmpl := m.Template
		if req.Template != "" {
			tmpl, err = template.Parse(req.Template)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}

		var values template.Values
		if req.Suffix != "" {
			values.Prompt = prompt
			values.Suffix = req.Suffix
		} else {
			var msgs []api.Message
			if req.System != "" {
				msgs = append(msgs, api.Message{Role: "system", Content: req.System})
			} else if m.System != "" {
				msgs = append(msgs, api.Message{Role: "system", Content: m.System})
			}

			if req.Context == nil {
				msgs = append(msgs, m.Messages...)
			}

			for _, i := range images {
				imgPrompt := ""
				msgs = append(msgs, api.Message{Role: "user", Content: fmt.Sprintf("[img-%d]"+imgPrompt, i.ID)})
			}

			values.Messages = append(msgs, api.Message{Role: "user", Content: req.Prompt})
		}

		values.Think = req.Think != nil && *req.Think
		values.IsThinkSet = req.Think != nil

		var b bytes.Buffer
		if req.Context != nil {
			slog.Warn("the context field is deprecated and will be removed in a future version of Goobla")
			s, err := r.Detokenize(c.Request.Context(), req.Context)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			b.WriteString(s)
		}

		if err := tmpl.Execute(&b, values); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		prompt = b.String()
	}

	var thinkingState *thinking.Parser
	openingTag, closingTag := thinking.InferTags(m.Template.Template)
	if req.Think != nil && *req.Think && openingTag != "" && closingTag != "" {
		thinkingState = &thinking.Parser{
			OpeningTag: openingTag,
			ClosingTag: closingTag,
		}
	}

	var sbRaw strings.Builder
	var sbThinking strings.Builder
	var sbContent strings.Builder

	ch := make(chan any)
	go func() {
		defer close(ch)
		if err := r.Completion(c.Request.Context(), llm.CompletionRequest{
			Prompt:  prompt,
			Images:  images,
			Format:  req.Format,
			Options: opts,
		}, func(cr llm.CompletionResponse) {
			res := api.GenerateResponse{
				Model:     req.Model,
				CreatedAt: time.Now().UTC(),
				Response:  cr.Content,
				Done:      cr.Done,
				Metrics: api.Metrics{
					PromptEvalCount:    cr.PromptEvalCount,
					PromptEvalDuration: cr.PromptEvalDuration,
					EvalCount:          cr.EvalCount,
					EvalDuration:       cr.EvalDuration,
				},
			}

			if thinkingState != nil {
				thinking, content := thinkingState.AddContent(cr.Content)
				res.Thinking = thinking
				res.Response = content
				sbThinking.WriteString(thinking)
				sbContent.WriteString(content)
			} else {
				sbContent.WriteString(cr.Content)
			}

			if _, err := sbRaw.WriteString(cr.Content); err != nil {
				ch <- gin.H{"error": err.Error()}
			}

			if cr.Done {
				res.DoneReason = cr.DoneReason.String()
				res.TotalDuration = time.Since(checkpointStart)
				res.LoadDuration = checkpointLoaded.Sub(checkpointStart)

				if !req.Raw {
					tokens, err := r.Tokenize(c.Request.Context(), prompt+sbRaw.String())
					if err != nil {
						ch <- gin.H{"error": err.Error()}
						return
					}
					res.Context = tokens
				}
			}

			ch <- res
		}); err != nil {
			ch <- gin.H{"error": err.Error()}
		}
	}()

	if req.Stream != nil && !*req.Stream {
		var r api.GenerateResponse
		for rr := range ch {
			switch t := rr.(type) {
			case api.GenerateResponse:
				r = t
			case gin.H:
				msg, ok := t["error"].(string)
				if !ok {
					msg = "unexpected error format in response"
				}

				c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
				return
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "unexpected response"})
				return
			}
		}

		r.Thinking = sbThinking.String()
		r.Response = sbContent.String()

		c.JSON(http.StatusOK, r)
		return
	}

	streamResponse(c, ch)
}

func (s *Server) EmbedHandler(c *gin.Context) {
	checkpointStart := time.Now()
	var req api.EmbedRequest
	err := c.ShouldBindJSON(&req)
	switch {
	case errors.Is(err, io.EOF):
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "missing request body"})
		return
	case err != nil:
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	truncate := true

	if req.Truncate != nil && !*req.Truncate {
		truncate = false
	}

	var input []string

	switch i := req.Input.(type) {
	case string:
		if len(i) > 0 {
			input = append(input, i)
		}
	case []any:
		for _, v := range i {
			if _, ok := v.(string); !ok {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid input type"})
				return
			}
			input = append(input, v.(string))
		}
	default:
		if req.Input != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid input type"})
			return
		}
	}

	name, err := getExistingName(model.ParseName(req.Model))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("model '%s' not found", req.Model)})
		return
	}

	r, m, opts, err := s.scheduleRunner(c.Request.Context(), name.String(), []model.Capability{}, req.Options, req.KeepAlive)
	if err != nil {
		handleScheduleError(c, req.Model, err)
		return
	}

	checkpointLoaded := time.Now()

	if len(input) == 0 {
		c.JSON(http.StatusOK, api.EmbedResponse{Model: req.Model, Embeddings: [][]float32{}})
		return
	}

	kvData, _, err := getModelData(m.ModelPath, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var count int
	for i, s := range input {
		tokens, err := r.Tokenize(c.Request.Context(), s)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		ctxLen := min(opts.NumCtx, int(kvData.ContextLength()))
		if len(tokens) > ctxLen {
			if !truncate {
				c.JSON(http.StatusBadRequest, gin.H{"error": "input length exceeds maximum context length"})
				return
			}

			tokens = tokens[:ctxLen]
			s, err = r.Detokenize(c.Request.Context(), tokens)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}

		count += len(tokens)

		input[i] = s
	}

	var g errgroup.Group
	embeddings := make([][]float32, len(input))
	for i, text := range input {
		g.Go(func() error {
			embedding, err := r.Embedding(c.Request.Context(), text)
			if err != nil {
				return err
			}
			embeddings[i] = normalize(embedding)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": strings.TrimSpace(err.Error())})
		return
	}

	resp := api.EmbedResponse{
		Model:           req.Model,
		Embeddings:      embeddings,
		TotalDuration:   time.Since(checkpointStart),
		LoadDuration:    checkpointLoaded.Sub(checkpointStart),
		PromptEvalCount: count,
	}
	c.JSON(http.StatusOK, resp)
}

func normalize(vec []float32) []float32 {
	var sum float32
	for _, v := range vec {
		sum += v * v
	}

	norm := float32(0.0)
	if sum > 0 {
		norm = float32(1.0 / math.Sqrt(float64(sum)))
	}

	for i := range vec {
		vec[i] *= norm
	}
	return vec
}

func (s *Server) EmbeddingsHandler(c *gin.Context) {
	var req api.EmbeddingRequest
	if err := c.ShouldBindJSON(&req); errors.Is(err, io.EOF) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "missing request body"})
		return
	} else if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	name := model.ParseName(req.Model)
	if !name.IsValid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model is required"})
		return
	}

	r, _, _, err := s.scheduleRunner(c.Request.Context(), name.String(), []model.Capability{}, req.Options, req.KeepAlive)
	if err != nil {
		handleScheduleError(c, req.Model, err)
		return
	}

	// an empty request loads the model
	if req.Prompt == "" {
		c.JSON(http.StatusOK, api.EmbeddingResponse{Embedding: []float64{}})
		return
	}

	embedding, err := r.Embedding(c.Request.Context(), req.Prompt)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": strings.TrimSpace(err.Error())})
		return
	}

	var e []float64
	for _, v := range embedding {
		e = append(e, float64(v))
	}

	resp := api.EmbeddingResponse{
		Embedding: e,
	}
	c.JSON(http.StatusOK, resp)
}

func (s *Server) PullHandler(c *gin.Context) {
	var req api.PullRequest
	err := c.ShouldBindJSON(&req)
	switch {
	case errors.Is(err, io.EOF):
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "missing request body"})
		return
	case err != nil:
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	name := model.ParseName(cmp.Or(req.Model, req.Name))
	if !name.IsValid() {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": errtypes.InvalidModelNameErrMsg})
		return
	}

	name, err = getExistingName(name)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ch := make(chan any)
	go func() {
		defer close(ch)
		fn := func(r api.ProgressResponse) {
			ch <- r
		}

		regOpts := &registryOptions{
			Insecure: req.Insecure,
			Timeout:  defaultHTTPTimeout,
		}

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		if err := PullModel(ctx, name.DisplayShortest(), regOpts, fn); err != nil {
			ch <- gin.H{"error": err.Error()}
		}
	}()

	if req.Stream != nil && !*req.Stream {
		waitForStream(c, ch)
		return
	}

	streamResponse(c, ch)
}

func (s *Server) PushHandler(c *gin.Context) {
	var req api.PushRequest
	err := c.ShouldBindJSON(&req)
	switch {
	case errors.Is(err, io.EOF):
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "missing request body"})
		return
	case err != nil:
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var mname string
	if req.Model != "" {
		mname = req.Model
	} else if req.Name != "" {
		mname = req.Name
	} else {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "model is required"})
		return
	}

	ch := make(chan any)
	go func() {
		defer close(ch)
		fn := func(r api.ProgressResponse) {
			ch <- r
		}

		regOpts := &registryOptions{
			Insecure: req.Insecure,
			Timeout:  defaultHTTPTimeout,
		}

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		name, err := getExistingName(model.ParseName(mname))
		if err != nil {
			ch <- gin.H{"error": err.Error()}
			return
		}

		if err := PushModel(ctx, name.DisplayShortest(), regOpts, fn); err != nil {
			ch <- gin.H{"error": err.Error()}
		}
	}()

	if req.Stream != nil && !*req.Stream {
		waitForStream(c, ch)
		return
	}

	streamResponse(c, ch)
}

// getExistingName searches the models directory for the longest prefix match of
// the input name and returns the input name with all existing parts replaced
// with each part found. If no parts are found, the input name is returned as
// is.
func getExistingName(n model.Name) (model.Name, error) {
	var zero model.Name
	existing, err := Manifests(true)
	if err != nil {
		return zero, err
	}
	var set model.Name // tracks parts already canonicalized
	for e := range existing {
		if set.Host == "" && strings.EqualFold(e.Host, n.Host) {
			n.Host = e.Host
		}
		if set.Namespace == "" && strings.EqualFold(e.Namespace, n.Namespace) {
			n.Namespace = e.Namespace
		}
		if set.Model == "" && strings.EqualFold(e.Model, n.Model) {
			n.Model = e.Model
		}
		if set.Tag == "" && strings.EqualFold(e.Tag, n.Tag) {
			n.Tag = e.Tag
		}
	}
	return n, nil
}

func (s *Server) DeleteHandler(c *gin.Context) {
	var r api.DeleteRequest
	if err := c.ShouldBindJSON(&r); errors.Is(err, io.EOF) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "missing request body"})
		return
	} else if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	n := model.ParseName(cmp.Or(r.Model, r.Name))
	if !n.IsValid() {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("name %q is invalid", cmp.Or(r.Model, r.Name))})
		return
	}

	n, err := getExistingName(n)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("model '%s' not found", cmp.Or(r.Model, r.Name))})
		return
	}

	m, err := ParseNamedManifest(n)
	if err != nil {
		switch {
		case os.IsNotExist(err):
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("model '%s' not found", cmp.Or(r.Model, r.Name))})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	if err := m.Remove(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := m.RemoveLayers(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
}

func (s *Server) ShowHandler(c *gin.Context) {
	var req api.ShowRequest
	err := c.ShouldBindJSON(&req)
	switch {
	case errors.Is(err, io.EOF):
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "missing request body"})
		return
	case err != nil:
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Model != "" {
		// noop
	} else if req.Name != "" {
		req.Model = req.Name
	} else {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "model is required"})
		return
	}

	resp, err := GetModelInfo(req)
	if err != nil {
		switch {
		case os.IsNotExist(err):
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("model '%s' not found", req.Model)})
		case err.Error() == errtypes.InvalidModelNameErrMsg:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, resp)
}

func GetModelInfo(req api.ShowRequest) (*api.ShowResponse, error) {
	name := model.ParseName(req.Model)
	if !name.IsValid() {
		return nil, ErrModelPathInvalid
	}
	name, err := getExistingName(name)
	if err != nil {
		return nil, err
	}

	m, err := GetModel(name.String())
	if err != nil {
		return nil, err
	}

	modelDetails := api.ModelDetails{
		ParentModel:       m.ParentModel,
		Format:            m.Config.ModelFormat,
		Family:            m.Config.ModelFamily,
		Families:          m.Config.ModelFamilies,
		ParameterSize:     m.Config.ModelType,
		QuantizationLevel: m.Config.FileType,
	}

	if req.System != "" {
		m.System = req.System
	}

	msgs := make([]api.Message, len(m.Messages))
	for i, msg := range m.Messages {
		msgs[i] = api.Message{Role: msg.Role, Content: msg.Content}
	}

	manifest, err := ParseNamedManifest(name)
	if err != nil {
		return nil, err
	}

	resp := &api.ShowResponse{
		License:      strings.Join(m.License, "\n"),
		System:       m.System,
		Template:     m.Template.String(),
		Details:      modelDetails,
		Messages:     msgs,
		Capabilities: m.Capabilities(),
		ModifiedAt:   manifest.fi.ModTime(),
	}

	var params []string
	cs := 30
	for k, v := range m.Options {
		switch val := v.(type) {
		case []any:
			for _, nv := range val {
				params = append(params, fmt.Sprintf("%-*s %#v", cs, k, nv))
			}
		default:
			params = append(params, fmt.Sprintf("%-*s %#v", cs, k, v))
		}
	}
	resp.Parameters = strings.Join(params, "\n")

	for k, v := range req.Options {
		if _, ok := req.Options[k]; ok {
			m.Options[k] = v
		}
	}

	var sb strings.Builder
	fmt.Fprintln(&sb, "# Modelfile generated by \"goobla show\"")
	fmt.Fprintln(&sb, "# To build a new Modelfile based on this, replace FROM with:")
	fmt.Fprintf(&sb, "# FROM %s\n\n", m.ShortName)
	fmt.Fprint(&sb, m.String())
	resp.Modelfile = sb.String()

	kvData, tensors, err := getModelData(m.ModelPath, req.Verbose)
	if err != nil {
		return nil, err
	}

	delete(kvData, "general.name")
	delete(kvData, "tokenizer.chat_template")
	resp.ModelInfo = kvData

	tensorData := make([]api.Tensor, len(tensors.Items()))
	for cnt, t := range tensors.Items() {
		tensorData[cnt] = api.Tensor{Name: t.Name, Type: t.Type(), Shape: t.Shape}
	}
	resp.Tensors = tensorData

	if len(m.ProjectorPaths) > 0 {
		projectorData, _, err := getModelData(m.ProjectorPaths[0], req.Verbose)
		if err != nil {
			return nil, err
		}
		resp.ProjectorInfo = projectorData
	}

	return resp, nil
}

func getModelData(digest string, verbose bool) (ggml.KV, ggml.Tensors, error) {
	maxArraySize := 0
	if verbose {
		maxArraySize = -1
	}
	data, err := llm.LoadModel(digest, maxArraySize)
	if err != nil {
		return nil, ggml.Tensors{}, err
	}

	kv := data.KV()

	if !verbose {
		for k := range kv {
			if t, ok := kv[k].([]any); len(t) > 5 && ok {
				kv[k] = []any{}
			}
		}
	}

	return kv, data.Tensors(), nil
}

func (s *Server) ListHandler(c *gin.Context) {
	ms, err := Manifests(true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	models := []api.ListModelResponse{}
	for n, m := range ms {
		var cf ConfigV2

		if m.Config.Digest != "" {
			f, err := m.Config.Open()
			if err != nil {
				slog.Warn("bad manifest filepath", "name", n, "error", err)
				continue
			}
			defer f.Close()

			if err := json.NewDecoder(f).Decode(&cf); err != nil {
				slog.Warn("bad manifest config", "name", n, "error", err)
				continue
			}
		}

		// tag should never be masked
		models = append(models, api.ListModelResponse{
			Model:      n.DisplayShortest(),
			Name:       n.DisplayShortest(),
			Size:       m.Size(),
			Digest:     m.digest,
			ModifiedAt: m.fi.ModTime(),
			Details: api.ModelDetails{
				Format:            cf.ModelFormat,
				Family:            cf.ModelFamily,
				Families:          cf.ModelFamilies,
				ParameterSize:     cf.ModelType,
				QuantizationLevel: cf.FileType,
			},
		})
	}

	slices.SortStableFunc(models, func(i, j api.ListModelResponse) int {
		// most recently modified first
		return cmp.Compare(j.ModifiedAt.Unix(), i.ModifiedAt.Unix())
	})

	c.JSON(http.StatusOK, api.ListResponse{Models: models})
}

func (s *Server) CopyHandler(c *gin.Context) {
	var r api.CopyRequest
	if err := c.ShouldBindJSON(&r); errors.Is(err, io.EOF) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "missing request body"})
		return
	} else if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	src := model.ParseName(r.Source)
	if !src.IsValid() {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("source %q is invalid", r.Source)})
		return
	}
	src, err := getExistingName(src)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dst := model.ParseName(r.Destination)
	if !dst.IsValid() {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("destination %q is invalid", r.Destination)})
		return
	}
	dst, err = getExistingName(dst)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := CopyModel(src, dst); errors.Is(err, os.ErrNotExist) {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("model %q not found", r.Source)})
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

func (s *Server) HeadBlobHandler(c *gin.Context) {
	path, err := GetBlobsPath(c.Param("digest"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if _, err := os.Stat(path); err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("blob %q not found", c.Param("digest"))})
		return
	}

	c.Status(http.StatusOK)
}

func (s *Server) CreateBlobHandler(c *gin.Context) {

	path, err := GetBlobsPath(c.Param("digest"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err = os.Stat(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		// noop
	case err != nil:
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	default:
		c.Status(http.StatusOK)
		return
	}

	layer, err := NewLayer(c.Request.Body, "")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if layer.Digest != c.Param("digest") {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("digest mismatch, expected %q, got %q", c.Param("digest"), layer.Digest)})
		return
	}

	c.Status(http.StatusCreated)
}

func isLocalIP(ip netip.Addr) bool {
	if interfaces, err := net.Interfaces(); err == nil {
		for _, iface := range interfaces {
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}

			for _, a := range addrs {
				if parsed, _, err := net.ParseCIDR(a.String()); err == nil {
					if parsed.String() == ip.String() {
						return true
					}
				}
			}
		}
	}

	return false
}

func allowedHost(host string) bool {
	host = strings.ToLower(host)

	if host == "" || host == "localhost" {
		return true
	}

	if hostname, err := os.Hostname(); err == nil && host == strings.ToLower(hostname) {
		return true
	}

	tlds := []string{
		"localhost",
		"local",
		"internal",
	}

	// check if the host is a local TLD
	for _, tld := range tlds {
		if strings.HasSuffix(host, "."+tld) {
			return true
		}
	}

	return false
}

func allowedHostsMiddleware(addr net.Addr) gin.HandlerFunc {
	return func(c *gin.Context) {
		if addr == nil {
			c.Next()
			return
		}

		if addr, err := netip.ParseAddrPort(addr.String()); err == nil && !addr.Addr().IsLoopback() {
			c.Next()
			return
		}

		host, _, err := net.SplitHostPort(c.Request.Host)
		if err != nil {
			host = c.Request.Host
		}

		if addr, err := netip.ParseAddr(host); err == nil {
			if addr.IsLoopback() || addr.IsPrivate() || addr.IsUnspecified() || isLocalIP(addr) {
				c.Next()
				return
			}
		}

		if allowedHost(host) {
			if c.Request.Method == http.MethodOptions {
				c.AbortWithStatus(http.StatusNoContent)
				return
			}

			c.Next()
			return
		}

		c.AbortWithStatus(http.StatusForbidden)
	}
}

func (s *Server) GenerateRoutes(logger *slog.Logger, rc *goobla.Registry) (http.Handler, error) {
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowWildcard = true
	corsConfig.AllowBrowserExtensions = true
	corsConfig.AllowHeaders = []string{
		"Authorization",
		"Content-Type",
		"User-Agent",
		"Accept",
		"X-Requested-With",

		// OpenAI compatibility headers
		"OpenAI-Beta",
		"x-stainless-arch",
		"x-stainless-async",
		"x-stainless-custom-poll-interval",
		"x-stainless-helper-method",
		"x-stainless-lang",
		"x-stainless-os",
		"x-stainless-package-version",
		"x-stainless-poll-helper",
		"x-stainless-retry-count",
		"x-stainless-runtime",
		"x-stainless-runtime-version",
		"x-stainless-timeout",
	}
	corsConfig.AllowOrigins = envconfig.AllowedOrigins()

	r := gin.Default()
	r.HandleMethodNotAllowed = true
	r.Use(
		cors.New(corsConfig),
		allowedHostsMiddleware(s.addr),
	)

	// General
	r.HEAD("/", func(c *gin.Context) { c.String(http.StatusOK, "Goobla is running") })
	r.GET("/", func(c *gin.Context) { c.String(http.StatusOK, "Goobla is running") })
	r.HEAD("/api/version", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"version": version.Version}) })
	r.GET("/api/version", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"version": version.Version}) })

	// Local model cache management (new implementation is at end of function)
	r.POST("/api/pull", s.PullHandler)
	r.POST("/api/push", s.PushHandler)
	r.HEAD("/api/tags", s.ListHandler)
	r.GET("/api/tags", s.ListHandler)
	r.POST("/api/show", s.ShowHandler)
	r.DELETE("/api/delete", s.DeleteHandler)

	// Create
	r.POST("/api/create", s.CreateHandler)
	r.POST("/api/blobs/:digest", s.CreateBlobHandler)
	r.HEAD("/api/blobs/:digest", s.HeadBlobHandler)
	r.POST("/api/copy", s.CopyHandler)

	// Inference
	r.GET("/api/ps", s.PsHandler)
	r.POST("/api/generate", s.GenerateHandler)
	r.POST("/api/chat", s.ChatHandler)
	r.POST("/api/embed", s.EmbedHandler)
	r.POST("/api/embeddings", s.EmbeddingsHandler)

	// Inference (OpenAI compatibility)
	r.POST("/v1/chat/completions", openaimid.ChatMiddleware(), s.ChatHandler)
	r.POST("/v1/completions", openaimid.CompletionsMiddleware(), s.GenerateHandler)
	r.POST("/v1/embeddings", openaimid.EmbeddingsMiddleware(), s.EmbedHandler)
	r.GET("/v1/models", openaimid.ListMiddleware(), s.ListHandler)
	r.GET("/v1/models/:model", openaimid.RetrieveMiddleware(), s.ShowHandler)

	if rc != nil {
		// wrap old with new
		rs := &registry.Local{
			Client:   rc,
			Logger:   logger,
			Fallback: r,

			Prune: PruneLayers,
		}
		return rs, nil
	}

	return r, nil
}

func Serve(ln net.Listener) error {
	if err := envconfig.Validate(); err != nil {
		return err
	}
	slog.SetDefault(logutil.NewLogger(os.Stderr, envconfig.LogLevel()))
	slog.Info("server config", "env", envconfig.Values())

	blobsDir, err := GetBlobsPath("")
	if err != nil {
		return err
	}
	if err := fixBlobs(blobsDir); err != nil {
		return err
	}

	if !envconfig.NoPrune() {
		if _, err := Manifests(false); err != nil {
			slog.Warn("corrupt manifests detected, skipping prune operation.  Re-pull or delete to clear", "error", err)
		} else {
			// clean up unused layers and manifests
			if err := PruneLayers(); err != nil {
				return err
			}

			manifestsPath, err := GetManifestPath()
			if err != nil {
				return err
			}

			if err := PruneDirectory(manifestsPath); err != nil {
				return err
			}
		}
	}

	s := &Server{addr: ln.Addr()}

	var rc *goobla.Registry
	if useClient2 {
		var err error
		rc, err = goobla.DefaultRegistry()
		if err != nil {
			return err
		}
	}

	h, err := s.GenerateRoutes(slog.Default(), rc)
	if err != nil {
		return err
	}

	pprofAddr := strings.ToLower(envconfig.PprofAddr())
	var srvr *http.Server
	switch pprofAddr {
	case "on":
		// Use DefaultServeMux so we get net/http/pprof handlers on the
		// main server.
		http.Handle("/", h)
		srvr = newHTTPServer(nil)
	case "", "off", "false", "0":
		srvr = newHTTPServer(h)
	default:
		// Serve application routes on the main server and start pprof on
		// a separate port using the default mux.
		srvr = newHTTPServer(h)
		go func() {
			slog.Info("pprof listening", "addr", pprofAddr)
			if err := http.ListenAndServe(pprofAddr, nil); err != nil && !errors.Is(err, http.ErrServerClosed) {
				slog.Error("pprof server error", "error", err)
			}
		}()
	}

	ctx, done := context.WithCancel(context.Background())
	schedCtx, schedDone := context.WithCancel(ctx)
	sched := InitScheduler(schedCtx)
	s.sched = sched

	proto := "http"
	if envconfig.TLSCert() != "" {
		proto = "https"
	}
	slog.Info(fmt.Sprintf("Listening on %s (%s, version %s)", ln.Addr(), proto, version.Version))

	// listen for a ctrl+c and stop any loaded llm
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signals
		ctxShutdown, cancel := context.WithTimeout(context.Background(), envconfig.ShutdownTimeout())
		defer cancel()
		if err := srvr.Shutdown(ctxShutdown); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server shutdown error", "error", err)
		}
		schedDone()
		sched.unloadAllRunners()
		done()
	}()

	s.sched.Run(schedCtx)

	// register the experimental webp decoder
	// so webp images can be used in multimodal inputs
	image.RegisterFormat("webp", "RIFF????WEBP", webp.Decode, webp.DecodeConfig)

	// At startup we retrieve GPU information so we can get log messages before loading a model
	// This will log warnings to the log in case we have problems with detected GPUs
	gpus := discover.GetGPUInfo()
	gpus.LogDetails()

	if envconfig.TLSCert() != "" {
		err = srvr.ServeTLS(ln, envconfig.TLSCert(), envconfig.TLSKey())
	} else {
		err = srvr.Serve(ln)
	}
	// If server is closed from the signal handler, wait for the ctx to be done
	// otherwise error out quickly
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	<-ctx.Done()
	return nil
}

func waitForStream(c *gin.Context, ch chan any) {
	c.Header("Content-Type", "application/json")
	var latest api.ProgressResponse
	for resp := range ch {
		switch r := resp.(type) {
		case api.ProgressResponse:
			latest = r
		case gin.H:
			status, ok := r["status"].(int)
			if !ok {
				status = http.StatusInternalServerError
			}
			errorMsg, ok := r["error"].(string)
			if !ok {
				errorMsg = "unknown error"
			}
			c.JSON(status, gin.H{"error": errorMsg})
			return
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "unknown message type"})
			return
		}
	}

	c.JSON(http.StatusOK, latest)
}

func streamResponse(c *gin.Context, ch chan any) {
	c.Header("Content-Type", "application/x-ndjson")
	c.Stream(func(w io.Writer) bool {
		val, ok := <-ch
		if !ok {
			return false
		}

		bts, err := json.Marshal(val)
		if err != nil {
			slog.Info(fmt.Sprintf("streamResponse: json.Marshal failed with %s", err))
			return false
		}

		// Delineate chunks with new-line delimiter
		bts = append(bts, '\n')
		if _, err := w.Write(bts); err != nil {
			slog.Info(fmt.Sprintf("streamResponse: w.Write failed with %s", err))
			return false
		}

		return true
	})
}

func (s *Server) PsHandler(c *gin.Context) {
	models := []api.ProcessModelResponse{}

	for _, v := range s.sched.loaded {
		model := v.model
		modelDetails := api.ModelDetails{
			Format:            model.Config.ModelFormat,
			Family:            model.Config.ModelFamily,
			Families:          model.Config.ModelFamilies,
			ParameterSize:     model.Config.ModelType,
			QuantizationLevel: model.Config.FileType,
		}

		mr := api.ProcessModelResponse{
			Model:     model.ShortName,
			Name:      model.ShortName,
			Size:      int64(v.estimatedTotal),
			SizeVRAM:  int64(v.estimatedVRAM),
			Digest:    model.Digest,
			Details:   modelDetails,
			ExpiresAt: v.expiresAt,
		}
		// The scheduler waits to set expiresAt, so if a model is loading it's
		// possible that it will be set to the unix epoch. For those cases, just
		// calculate the time w/ the sessionDuration instead.
		var epoch time.Time
		if v.expiresAt == epoch {
			mr.ExpiresAt = time.Now().Add(v.sessionDuration)
		}

		models = append(models, mr)
	}

	slices.SortStableFunc(models, func(i, j api.ProcessModelResponse) int {
		// longest duration remaining listed first
		return cmp.Compare(j.ExpiresAt.Unix(), i.ExpiresAt.Unix())
	})

	c.JSON(http.StatusOK, api.ProcessResponse{Models: models})
}

func (s *Server) ChatHandler(c *gin.Context) {
	checkpointStart := time.Now()

	var req api.ChatRequest
	if err := c.ShouldBindJSON(&req); errors.Is(err, io.EOF) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "missing request body"})
		return
	} else if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// expire the runner
	if len(req.Messages) == 0 && req.KeepAlive != nil && int(req.KeepAlive.Seconds()) == 0 {
		model, err := GetModel(req.Model)
		if err != nil {
			switch {
			case os.IsNotExist(err):
				c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("model '%s' not found", req.Model)})
			case err.Error() == errtypes.InvalidModelNameErrMsg:
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}
		s.sched.expireRunner(model)

		c.JSON(http.StatusOK, api.ChatResponse{
			Model:      req.Model,
			CreatedAt:  time.Now().UTC(),
			Message:    api.Message{Role: "assistant"},
			Done:       true,
			DoneReason: "unload",
		})
		return
	}

	caps := []model.Capability{model.CapabilityCompletion}
	if len(req.Tools) > 0 {
		caps = append(caps, model.CapabilityTools)
	}
	if req.Think != nil && *req.Think {
		caps = append(caps, model.CapabilityThinking)
	}

	name := model.ParseName(req.Model)
	if !name.IsValid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model is required"})
		return
	}
	name, err := getExistingName(name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model is required"})
		return
	}

	r, m, opts, err := s.scheduleRunner(c.Request.Context(), name.String(), caps, req.Options, req.KeepAlive)
	if errors.Is(err, errCapabilityCompletion) {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%q does not support chat", req.Model)})
		return
	} else if err != nil {
		handleScheduleError(c, req.Model, err)
		return
	}

	checkpointLoaded := time.Now()

	if len(req.Messages) == 0 {
		c.JSON(http.StatusOK, api.ChatResponse{
			Model:      req.Model,
			CreatedAt:  time.Now().UTC(),
			Message:    api.Message{Role: "assistant"},
			Done:       true,
			DoneReason: "load",
		})
		return
	}

	msgs := append(m.Messages, req.Messages...)
	if req.Messages[0].Role != "system" && m.System != "" {
		msgs = append([]api.Message{{Role: "system", Content: m.System}}, msgs...)
	}
	msgs = filterThinkTags(msgs, m)

	prompt, images, err := chatPrompt(c.Request.Context(), m, r.Tokenize, opts, msgs, req.Tools, req.Think)
	if err != nil {
		slog.Error("chat prompt error", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var thinkingState *thinking.Parser
	openingTag, closingTag := thinking.InferTags(m.Template.Template)
	if req.Think != nil && *req.Think && openingTag != "" && closingTag != "" {
		thinkingState = &thinking.Parser{
			OpeningTag: openingTag,
			ClosingTag: closingTag,
		}
	}

	var toolParser *tools.Parser
	if len(req.Tools) > 0 {
		toolParser = tools.NewParser(m.Template.Template, req.Tools)
	}

	ch := make(chan any)
	go func() {
		defer close(ch)

		if err := r.Completion(c.Request.Context(), llm.CompletionRequest{
			Prompt:  prompt,
			Images:  images,
			Format:  req.Format,
			Options: opts,
		}, func(r llm.CompletionResponse) {
			res := api.ChatResponse{
				Model:     req.Model,
				CreatedAt: time.Now().UTC(),
				Message:   api.Message{Role: "assistant", Content: r.Content},
				Done:      r.Done,
				Metrics: api.Metrics{
					PromptEvalCount:    r.PromptEvalCount,
					PromptEvalDuration: r.PromptEvalDuration,
					EvalCount:          r.EvalCount,
					EvalDuration:       r.EvalDuration,
				},
			}

			if thinkingState != nil {
				thinkingContent, remainingContent := thinkingState.AddContent(res.Message.Content)
				if thinkingContent == "" && remainingContent == "" && !r.Done {
					// need to accumulate more to decide what to send
					return
				}
				res.Message.Content = remainingContent
				res.Message.Thinking = thinkingContent
			}

			if r.Done {
				res.DoneReason = r.DoneReason.String()
				res.TotalDuration = time.Since(checkpointStart)
				res.LoadDuration = checkpointLoaded.Sub(checkpointStart)
			}

			if len(req.Tools) > 0 {
				toolCalls, content := toolParser.Add(res.Message.Content)
				if len(content) > 0 {
					res.Message.Content = content
				} else if len(toolCalls) > 0 {
					res.Message.ToolCalls = toolCalls
					res.Message.Content = ""
				} else if res.Message.Thinking != "" {
					// don't return
				} else {
					if r.Done {
						res.Message.Content = toolParser.Content()
						ch <- res
					}
					return
				}
			}

			ch <- res
		}); err != nil {
			ch <- gin.H{"error": err.Error()}
		}
	}()

	if req.Stream != nil && !*req.Stream {
		var resp api.ChatResponse
		var toolCalls []api.ToolCall
		var sbThinking strings.Builder
		var sbContent strings.Builder
		for rr := range ch {
			switch t := rr.(type) {
			case api.ChatResponse:
				sbThinking.WriteString(t.Message.Thinking)
				sbContent.WriteString(t.Message.Content)
				resp = t
				if len(req.Tools) > 0 {
					toolCalls = append(toolCalls, t.Message.ToolCalls...)
				}
			case gin.H:
				msg, ok := t["error"].(string)
				if !ok {
					msg = "unexpected error format in response"
				}

				c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
				return
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "unexpected response"})
				return
			}
		}

		resp.Message.Content = sbContent.String()
		resp.Message.Thinking = sbThinking.String()

		if len(toolCalls) > 0 {
			resp.Message.ToolCalls = toolCalls
		}

		c.JSON(http.StatusOK, resp)
		return
	}

	streamResponse(c, ch)
}

func handleScheduleError(c *gin.Context, name string, err error) {
	switch {
	case errors.Is(err, errCapabilities), errors.Is(err, errRequired):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, context.Canceled):
		c.JSON(499, gin.H{"error": "request canceled"})
	case errors.Is(err, ErrMaxQueue):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
	case errors.Is(err, os.ErrNotExist):
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("model %q not found, try pulling it first", name)})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

func filterThinkTags(msgs []api.Message, m *Model) []api.Message {
	openingTag, closingTag := thinking.InferTags(m.Template.Template)
	if openingTag == "" || closingTag == "" {
		return msgs
	}

	finalUserIndex := -1
	for i, msg := range msgs {
		if msg.Role == "user" {
			finalUserIndex = i
		}
	}

	for i, msg := range msgs {
		if msg.Role == "assistant" && i < finalUserIndex {
			msgs[i].Content = stripThinking(msg.Content, openingTag, closingTag)
		}
	}

	return msgs
}

// stripThinking removes thinking traces from a single message using the tags
// inferred from the template. It is called for all previous assistant messages
// regardless of whether the current request has thinking enabled.
func stripThinking(content, openingTag, closingTag string) string {
	parser := &thinking.Parser{OpeningTag: openingTag, ClosingTag: closingTag}
	_, out := parser.AddContent(content)
	return out
}
