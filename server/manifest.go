package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/moogla/moogla/types/model"
)

type Manifest struct {
	SchemaVersion int     `json:"schemaVersion"`
	MediaType     string  `json:"mediaType"`
	Config        Layer   `json:"config"`
	Layers        []Layer `json:"layers"`

	filepath string
	fi       os.FileInfo
	digest   string
}

func (m *Manifest) Size() (size int64) {
	for _, layer := range append(m.Layers, m.Config) {
		size += layer.Size
	}

	return
}

func (m *Manifest) Remove() error {
	if err := os.Remove(m.filepath); err != nil {
		return err
	}

	manifests, err := GetManifestPath()
	if err != nil {
		return err
	}

	return PruneDirectory(manifests)
}

func (m *Manifest) RemoveLayers() error {
	for _, layer := range append(m.Layers, m.Config) {
		if layer.Digest != "" {
			if err := layer.Remove(); errors.Is(err, os.ErrNotExist) {
				slog.Debug("layer does not exist", "digest", layer.Digest)
			} else if err != nil {
				return err
			}
		}
	}

	return nil
}

func ParseNamedManifest(n model.Name) (*Manifest, error) {
	if !n.IsFullyQualified() {
		return nil, model.Unqualified(n)
	}

	manifests, err := GetManifestPath()
	if err != nil {
		return nil, err
	}

	p := filepath.Join(manifests, n.Filepath())

	var m Manifest
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	sha256sum := sha256.New()
	if err := json.NewDecoder(io.TeeReader(f, sha256sum)).Decode(&m); err != nil {
		return nil, err
	}

	m.filepath = p
	m.fi = fi
	m.digest = hex.EncodeToString(sha256sum.Sum(nil))

	return &m, nil
}

func WriteManifest(name model.Name, config Layer, layers []Layer) error {
	manifests, err := GetManifestPath()
	if err != nil {
		return err
	}

	p := filepath.Join(manifests, name.Filepath())
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}

	f, err := os.Create(p)
	if err != nil {
		return err
	}
	defer f.Close()

	m := Manifest{
		SchemaVersion: 2,
		MediaType:     "application/vnd.docker.distribution.manifest.v2+json",
		Config:        config,
		Layers:        layers,
	}

	return json.NewEncoder(f).Encode(m)
}

func Manifests(continueOnError bool) (map[model.Name]*Manifest, error) {
	manifests, err := GetManifestPath()
	if err != nil {
		return nil, err
	}

	ms := make(map[model.Name]*Manifest)

	fsys := os.DirFS(manifests)
	walkFn := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if continueOnError {
				slog.Warn("walk error", "path", path, "error", err)
				return nil
			}
			return err
		}

		if d.IsDir() {
			if path == "." {
				return nil
			}
			parts := strings.Split(path, string(filepath.Separator))
			if len(parts) > 3 { // host/namespace/model
				return fs.SkipDir
			}
			return nil
		}

		n := model.ParseNameFromFilepath(path)
		if !n.IsValid() {
			if continueOnError {
				slog.Warn("bad manifest name", "path", path)
				return nil
			}
			return fmt.Errorf("%s %w", path, err)
		}

		m, err := ParseNamedManifest(n)
		if err != nil {
			if continueOnError {
				slog.Warn("bad manifest", "name", n, "error", err)
				return nil
			}
			return fmt.Errorf("%s %w", n, err)
		}

		ms[n] = m
		return nil
	}

	if err := fs.WalkDir(fsys, ".", walkFn); err != nil {
		return nil, err
	}

	return ms, nil
}
