package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/goobla/goobla/api"
	"github.com/goobla/goobla/fs/ggml"
	"github.com/goobla/goobla/template"
	"github.com/goobla/goobla/types/model"
)

type layerGGML struct {
	Layer
	*ggml.GGML
}

func parseFromModel(ctx context.Context, name model.Name, fn func(api.ProgressResponse)) (layers []*layerGGML, err error) {
	m, err := ParseNamedManifest(name)
	switch {
	case errors.Is(err, os.ErrNotExist):
		if err := PullModel(ctx, name.String(), &registryOptions{Timeout: defaultHTTPTimeout}, fn); err != nil {
			return nil, err
		}

		m, err = ParseNamedManifest(name)
		if err != nil {
			return nil, err
		}
	case err != nil:
		return nil, err
	}

	for _, layer := range m.Layers {
		layer, err := NewLayerFromLayer(layer.Digest, layer.MediaType, name.DisplayShortest())
		if err != nil {
			return nil, err
		}

		switch layer.MediaType {
		case "application/vnd.goobla.image.model",
			"application/vnd.goobla.image.projector",
			"application/vnd.goobla.image.adapter":
			blobpath, err := GetBlobsPath(layer.Digest)
			if err != nil {
				return nil, err
			}

			blob, err := os.Open(blobpath)
			if err != nil {
				return nil, err
			}
			defer blob.Close()

			f, err := ggml.Decode(blob, -1)
			if err != nil {
				return nil, err
			}

			layers = append(layers, &layerGGML{layer, f})
		default:
			layers = append(layers, &layerGGML{layer, nil})
		}
	}

	return layers, nil
}

func detectChatTemplate(layers []*layerGGML) ([]*layerGGML, error) {
	for _, layer := range layers {
		if s := layer.GGML.KV().ChatTemplate(); s != "" {
			if t, err := template.Named(s); err != nil {
				slog.Debug("template detection", "error", err, "template", s)
			} else {
				layer, err := NewLayer(t.Reader(), "application/vnd.goobla.image.template")
				if err != nil {
					return nil, err
				}

				layer.status = fmt.Sprintf("using autodetected template %s", t.Name)
				layers = append(layers, &layerGGML{layer, nil})

				if t.Parameters != nil {
					var b bytes.Buffer
					if err := json.NewEncoder(&b).Encode(t.Parameters); err != nil {
						return nil, err
					}

					layer, err := NewLayer(&b, "application/vnd.goobla.image.params")
					if err != nil {
						return nil, err
					}

					layers = append(layers, &layerGGML{layer, nil})
				}
			}
		}
	}

	return layers, nil
}

func detectContentType(r io.Reader) (string, error) {
	var b bytes.Buffer
	if _, err := io.Copy(&b, r); err != nil {
		return "", err
	}

	if contentType := ggml.DetectContentType(b.Bytes()); contentType != "" {
		return contentType, nil
	}

	if contentType := http.DetectContentType(b.Bytes()); contentType != "application/octet-stream" {
		return contentType, nil
	}

	return "unknown", nil
}
