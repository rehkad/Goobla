package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/goobla/goobla/api"
	"github.com/goobla/goobla/fs/ggml"
	"github.com/goobla/goobla/llm"
	"github.com/goobla/goobla/template"
)

type tokenizeFunc func(context.Context, string) ([]int, error)

// chatPrompt accepts a list of messages and returns the prompt and images that should be used for the next chat turn.
// chatPrompt truncates any messages that exceed the context window of the model, making sure to always include 1) the
// latest message and 2) system messages
func chatPrompt(ctx context.Context, m *Model, tokenize tokenizeFunc, opts *api.Options, msgs []api.Message, tools []api.Tool, think *bool) (prompt string, images []llm.ImageData, _ error) {
	var system []api.Message

	imageNumTokens := 768
	if len(m.ProjectorPaths) > 0 {
		if tokens := projectorImageTokens(m.ProjectorPaths[0]); tokens > 0 {
			imageNumTokens = tokens
		}
	}

	n := len(msgs) - 1
	// in reverse, find all messages that fit into context window
	for i := n; i >= 0; i-- {
		// always include the last message
		if i == n {
			continue
		}

		system = make([]api.Message, 0)
		for j := range i {
			if msgs[j].Role == "system" {
				system = append(system, msgs[j])
			}
		}

		thinkVal := false
		if think != nil {
			thinkVal = *think
		}
		var b bytes.Buffer
		if err := m.Template.Execute(&b, template.Values{Messages: append(system, msgs[i:]...), Tools: tools, Think: thinkVal, IsThinkSet: think != nil}); err != nil {
			return "", nil, err
		}

		s, err := tokenize(ctx, b.String())
		if err != nil {
			return "", nil, err
		}

		ctxLen := len(s)
		if m.ProjectorPaths != nil {
			for _, m := range msgs[i:] {
				ctxLen += imageNumTokens * len(m.Images)
			}
		}

		if ctxLen > opts.NumCtx {
			slog.Debug("truncating input messages which exceed context length", "truncated", len(msgs[i:]))
			break
		} else {
			n = i
		}
	}

	currMsgIdx := n

	for cnt, msg := range msgs[currMsgIdx:] {
		if slices.Contains(m.Config.ModelFamilies, "mllama") && len(msg.Images) > 1 {
			return "", nil, errors.New("this model only supports one image while more than one image requested")
		}

		var prefix string
		prompt := msg.Content

		for _, i := range msg.Images {
			imgData := llm.ImageData{
				ID:   len(images),
				Data: i,
			}

			imgTag := fmt.Sprintf("[img-%d]", imgData.ID)
			if !strings.Contains(prompt, "[img]") {
				prefix += imgTag
			} else {
				prompt = strings.Replace(prompt, "[img]", imgTag, 1)
			}

			images = append(images, imgData)
		}
		msgs[currMsgIdx+cnt].Content = prefix + prompt
	}

	// truncate any messages that do not fit into the context window
	var b bytes.Buffer
	thinkVal := false
	if think != nil {
		thinkVal = *think
	}
	if err := m.Template.Execute(&b, template.Values{Messages: append(system, msgs[currMsgIdx:]...), Tools: tools, Think: thinkVal, IsThinkSet: think != nil}); err != nil {
		return "", nil, err
	}

	return b.String(), images, nil
}

func projectorImageTokens(path string) int {
	const defaultTokens = 768
	ggmlData, err := llm.LoadModel(path, 0)
	if err != nil {
		slog.Debug("unable to load projector", "path", path, "error", err)
		return defaultTokens
	}

	kv := ggmlData.KV()
	if kv.String("general.architecture") != "clip" {
		return defaultTokens
	}

	patchSize := kv.Uint("clip.vision.patch_size")
	imageSize := kv.Uint("clip.vision.image_size")
	if patchSize == 0 || imageSize == 0 {
		return defaultTokens
	}

	nPatches := (imageSize / patchSize) * (imageSize / patchSize)
	switch kv.String("clip.projector_type") {
	case "ldp", "ldpv2", "adapter":
		nPatches /= 4
	case "resampler":
		switch kv.Uint("clip.minicpmv_version") {
		case 2:
			nPatches = 96
		case 3, 4:
			nPatches = 64
		}
	case "qwen2vl_merger", "qwen2.5vl_merger":
		ps := patchSize * 2
		x := imageSize / ps
		if imageSize%ps > 0 {
			x++
		}
		nPatches = x * x
	case "gemma3":
		scale := kv.Uint("clip.vision.projector.scale_factor", 4)
		n := imageSize / patchSize
		n = n / scale
		nPatches = n * n
	case "idefics3", "internvl":
		scale := kv.Uint("clip.vision.projector.scale_factor", 1)
		if scale > 1 {
			nPatches /= scale * scale
		}
	case "pixtral":
		merge := kv.Uint("clip.vision.spatial_merge_size")
		if merge == 0 {
			merge = 1
		}
		nX := imageSize / patchSize / merge
		nY := imageSize / patchSize / merge
		nPatches = nY*nX + nY - 1
	}

	if nPatches == 0 {
		return defaultTokens
	}
	return int(nPatches)
}
