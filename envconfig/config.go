package envconfig

import (
	"fmt"
	"log/slog"
	"math"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// configDir returns the base configuration directory for Goobla.
// It first checks GOOBLA_CONFIG_DIR. If unset, it uses the user's
// configuration directory as determined by os.UserConfigDir and
// falls back to the home directory if necessary.
func configDir() string {
	if dir := Var("GOOBLA_CONFIG_DIR"); dir != "" {
		return dir
	}
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "goobla")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".goobla")
	}
	return ".goobla"
}

// ConfigDir exposes the configured directory for other packages.
func ConfigDir() string {
	return configDir()
}

// Host returns the scheme and host. Host can be configured via the GOOBLA_HOST environment variable.
// Default is scheme "http" and host "127.0.0.1:11434"
func Host() *url.URL {
	defaultPort := "11434"

	s := strings.TrimSpace(Var("GOOBLA_HOST"))
	scheme, hostport, ok := strings.Cut(s, "://")
	switch {
	case !ok:
		scheme, hostport = "http", s
	case scheme == "http":
		defaultPort = "80"
	case scheme == "https":
		defaultPort = "443"
	}

	hostport, path, _ := strings.Cut(hostport, "/")
	host, port, err := net.SplitHostPort(hostport)
	if err != nil {
		host, port = "127.0.0.1", defaultPort
		if ip := net.ParseIP(strings.Trim(hostport, "[]")); ip != nil {
			host = ip.String()
		} else if hostport != "" {
			host = hostport
		}
	}

	if n, err := strconv.ParseInt(port, 10, 32); err != nil || n > 65535 || n < 0 {
		slog.Warn("invalid port, using default", "port", port, "default", defaultPort)
		port = defaultPort
	}

	return &url.URL{
		Scheme: scheme,
		Host:   net.JoinHostPort(host, port),
		Path:   path,
	}
}

// AllowedOrigins returns a list of allowed origins. AllowedOrigins can be configured via the GOOBLA_ORIGINS environment variable.
func AllowedOrigins() (origins []string) {
	if s := Var("GOOBLA_ORIGINS"); s != "" {
		origins = strings.Split(s, ",")
	}

	for _, origin := range []string{"localhost", "127.0.0.1", "0.0.0.0"} {
		origins = append(origins,
			fmt.Sprintf("http://%s", origin),
			fmt.Sprintf("https://%s", origin),
			fmt.Sprintf("http://%s", net.JoinHostPort(origin, "*")),
			fmt.Sprintf("https://%s", net.JoinHostPort(origin, "*")),
		)
	}

	origins = append(origins,
		"app://*",
		"file://*",
		"tauri://*",
		"vscode-webview://*",
		"vscode-file://*",
	)

	return origins
}

// Models returns the path to the models directory. Models directory can be configured via the GOOBLA_MODELS environment variable.
// Default is $HOME/.goobla/models
func Models() (string, error) {
	if s := Var("GOOBLA_MODELS"); s != "" {
		return s, nil
	}

	dir := configDir()
	return filepath.Join(dir, "models"), nil
}

// KeepAlive returns the duration that models stay loaded in memory. KeepAlive can be configured via the GOOBLA_KEEP_ALIVE environment variable.
// Negative values are treated as infinite. Zero is treated as no keep alive.
// Default is 5 minutes.
func KeepAlive() (keepAlive time.Duration) {
	keepAlive = 5 * time.Minute
	if s := Var("GOOBLA_KEEP_ALIVE"); s != "" {
		if d, err := time.ParseDuration(s); err == nil {
			keepAlive = d
		} else if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			keepAlive = time.Duration(n) * time.Second
		}
	}

	if keepAlive < 0 {
		return time.Duration(math.MaxInt64)
	}

	return keepAlive
}

// LoadTimeout returns the duration for stall detection during model loads. LoadTimeout can be configured via the GOOBLA_LOAD_TIMEOUT environment variable.
// Zero or Negative values are treated as infinite.
// Default is 5 minutes.
func LoadTimeout() (loadTimeout time.Duration) {
	loadTimeout = 5 * time.Minute
	if s := Var("GOOBLA_LOAD_TIMEOUT"); s != "" {
		if d, err := time.ParseDuration(s); err == nil {
			loadTimeout = d
		} else if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			loadTimeout = time.Duration(n) * time.Second
		}
	}

	if loadTimeout <= 0 {
		return time.Duration(math.MaxInt64)
	}

	return loadTimeout
}

// RegistryTimeout returns the HTTP read timeout used for registry
// operations. RegistryTimeout can be configured via the
// GOOBLA_REGISTRY_TIMEOUT environment variable. Zero or negative values are
// treated as infinite. Default is 30 seconds.
func RegistryTimeout() (d time.Duration) {
	d = 30 * time.Second
	if s := Var("GOOBLA_REGISTRY_TIMEOUT"); s != "" {
		if v, err := time.ParseDuration(s); err == nil {
			d = v
		} else if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			d = time.Duration(n) * time.Second
		}
	}
	if d <= 0 {
		return time.Duration(math.MaxInt64)
	}
	return d
}

func httpTimeout(key string, d time.Duration) time.Duration {
	if s := Var(key); s != "" {
		if v, err := time.ParseDuration(s); err == nil {
			d = v
		} else if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			d = time.Duration(n) * time.Second
		}
	}
	if d <= 0 {
		return time.Duration(math.MaxInt64)
	}
	return d
}

// ReadTimeout returns the HTTP server read timeout. It is configured via the
// GOOBLA_HTTP_READ_TIMEOUT environment variable. Default is 30 seconds.
func ReadTimeout() time.Duration {
	return httpTimeout("GOOBLA_HTTP_READ_TIMEOUT", 30*time.Second)
}

// WriteTimeout returns the HTTP server write timeout. It is configured via the
// GOOBLA_HTTP_WRITE_TIMEOUT environment variable. Default is 30 seconds.
func WriteTimeout() time.Duration {
	return httpTimeout("GOOBLA_HTTP_WRITE_TIMEOUT", 30*time.Second)
}

// IdleTimeout returns the HTTP server idle timeout. It is configured via the
// GOOBLA_HTTP_IDLE_TIMEOUT environment variable. Default is 2 minutes.
func IdleTimeout() time.Duration {
	return httpTimeout("GOOBLA_HTTP_IDLE_TIMEOUT", 2*time.Minute)
}

// ShutdownTimeout returns the HTTP server shutdown timeout. It is configured via
// the GOOBLA_SHUTDOWN_TIMEOUT environment variable. Default is 5 seconds.
func ShutdownTimeout() time.Duration {
	return httpTimeout("GOOBLA_SHUTDOWN_TIMEOUT", 5*time.Second)
}

func Bool(k string) func() bool {
	return func() bool {
		if s := Var(k); s != "" {
			b, err := strconv.ParseBool(s)
			if err != nil {
				return true
			}

			return b
		}

		return false
	}
}

// LogLevel returns the log level for the application.
// Values are 0 or false INFO (Default), 1 or true DEBUG, 2 TRACE
func LogLevel() slog.Level {
	level := slog.LevelInfo
	if s := Var("GOOBLA_DEBUG"); s != "" {
		if b, _ := strconv.ParseBool(s); b {
			level = slog.LevelDebug
		} else if i, _ := strconv.ParseInt(s, 10, 64); i != 0 {
			level = slog.Level(i * -4)
		}
	}

	return level
}

var (
	// FlashAttention enables the experimental flash attention feature.
	FlashAttention = Bool("GOOBLA_FLASH_ATTENTION")
	// KvCacheType is the quantization type for the K/V cache.
	KvCacheType = String("GOOBLA_KV_CACHE_TYPE")
	// NoHistory disables readline history.
	NoHistory = Bool("GOOBLA_NOHISTORY")
	// NoPrune disables pruning of model blobs on startup.
	NoPrune = Bool("GOOBLA_NOPRUNE")
	// SchedSpread allows scheduling models across all GPUs.
	SchedSpread = Bool("GOOBLA_SCHED_SPREAD")
	// IntelGPU enables experimental Intel GPU detection.
	IntelGPU = Bool("GOOBLA_INTEL_GPU")
	// MultiUserCache optimizes prompt caching for multi-user scenarios
	MultiUserCache = Bool("GOOBLA_MULTIUSER_CACHE")
	// Enable the new Goobla engine
	NewEngine = Bool("GOOBLA_NEW_ENGINE")
	// ContextLength sets the default context length
	ContextLength = Uint("GOOBLA_CONTEXT_LENGTH", 4096)
	// Auth enables authentication between the Goobla client and server
	UseAuth = Bool("GOOBLA_AUTH")
	// PprofAddr configures the pprof server address. Set to "off" to disable
	// pprof or specify a custom address (e.g. 127.0.0.1:6060).
	PprofAddr = String("GOOBLA_PPROF")
)

func String(s string) func() string {
	return func() string {
		return Var(s)
	}
}

var (
	LLMLibrary = String("GOOBLA_LLM_LIBRARY")

	CudaVisibleDevices    = String("CUDA_VISIBLE_DEVICES")
	HipVisibleDevices     = String("HIP_VISIBLE_DEVICES")
	RocrVisibleDevices    = String("ROCR_VISIBLE_DEVICES")
	GpuDeviceOrdinal      = String("GPU_DEVICE_ORDINAL")
	HsaOverrideGfxVersion = String("HSA_OVERRIDE_GFX_VERSION")
)

func Uint(key string, defaultValue uint) func() uint {
	return func() uint {
		if s := Var(key); s != "" {
			if n, err := strconv.ParseUint(s, 10, 64); err != nil {
				slog.Warn("invalid environment variable, using default", "key", key, "value", s, "default", defaultValue)
			} else {
				return uint(n)
			}
		}

		return defaultValue
	}
}

var (
	// NumParallel sets the number of parallel model requests. NumParallel can be configured via the GOOBLA_NUM_PARALLEL environment variable.
	NumParallel = Uint("GOOBLA_NUM_PARALLEL", 0)
	// MaxRunners sets the maximum number of loaded models. MaxRunners can be configured via the GOOBLA_MAX_LOADED_MODELS environment variable.
	MaxRunners = Uint("GOOBLA_MAX_LOADED_MODELS", 0)
	// MaxQueue sets the maximum number of queued requests. MaxQueue can be configured via the GOOBLA_MAX_QUEUE environment variable.
	MaxQueue = Uint("GOOBLA_MAX_QUEUE", 512)
)

func Uint64(key string, defaultValue uint64) func() uint64 {
	return func() uint64 {
		if s := Var(key); s != "" {
			if n, err := strconv.ParseUint(s, 10, 64); err != nil {
				slog.Warn("invalid environment variable, using default", "key", key, "value", s, "default", defaultValue)
			} else {
				return n
			}
		}

		return defaultValue
	}
}

// Set aside VRAM per GPU
var GpuOverhead = Uint64("GOOBLA_GPU_OVERHEAD", 0)

type EnvVar struct {
	Name        string
	Value       any
	Description string
}

func AsMap() map[string]EnvVar {
	ret := map[string]EnvVar{
		"GOOBLA_DEBUG":              {"GOOBLA_DEBUG", LogLevel(), "Show additional debug information (e.g. GOOBLA_DEBUG=1)"},
		"GOOBLA_FLASH_ATTENTION":    {"GOOBLA_FLASH_ATTENTION", FlashAttention(), "Enabled flash attention"},
		"GOOBLA_KV_CACHE_TYPE":      {"GOOBLA_KV_CACHE_TYPE", KvCacheType(), "Quantization type for the K/V cache (default: f16)"},
		"GOOBLA_GPU_OVERHEAD":       {"GOOBLA_GPU_OVERHEAD", GpuOverhead(), "Reserve a portion of VRAM per GPU (bytes)"},
		"GOOBLA_HOST":               {"GOOBLA_HOST", Host(), "IP Address for the goobla server (default 127.0.0.1:11434)"},
		"GOOBLA_KEEP_ALIVE":         {"GOOBLA_KEEP_ALIVE", KeepAlive(), "The duration that models stay loaded in memory (default \"5m\")"},
		"GOOBLA_LLM_LIBRARY":        {"GOOBLA_LLM_LIBRARY", LLMLibrary(), "Set LLM library to bypass autodetection"},
		"GOOBLA_LOAD_TIMEOUT":       {"GOOBLA_LOAD_TIMEOUT", LoadTimeout(), "How long to allow model loads to stall before giving up (default \"5m\")"},
		"GOOBLA_REGISTRY_TIMEOUT":   {"GOOBLA_REGISTRY_TIMEOUT", RegistryTimeout(), "HTTP read timeout for registry operations (default \"30s\")"},
		"GOOBLA_HTTP_READ_TIMEOUT":  {"GOOBLA_HTTP_READ_TIMEOUT", ReadTimeout(), "HTTP server read timeout (default \"30s\")"},
		"GOOBLA_HTTP_WRITE_TIMEOUT": {"GOOBLA_HTTP_WRITE_TIMEOUT", WriteTimeout(), "HTTP server write timeout (default \"30s\")"},
		"GOOBLA_HTTP_IDLE_TIMEOUT":  {"GOOBLA_HTTP_IDLE_TIMEOUT", IdleTimeout(), "HTTP server idle timeout (default \"2m\")"},
		"GOOBLA_SHUTDOWN_TIMEOUT":   {"GOOBLA_SHUTDOWN_TIMEOUT", ShutdownTimeout(), "HTTP server shutdown timeout (default \"5s\")"},
		"GOOBLA_MAX_LOADED_MODELS":  {"GOOBLA_MAX_LOADED_MODELS", MaxRunners(), "Maximum number of loaded models per GPU"},
		"GOOBLA_MAX_QUEUE":          {"GOOBLA_MAX_QUEUE", MaxQueue(), "Maximum number of queued requests"},
		func() EnvVar {
			m, _ := Models()
			return EnvVar{"GOOBLA_MODELS", m, "The path to the models directory"}
		}(),
		"GOOBLA_CONFIG":          {"GOOBLA_CONFIG", String("GOOBLA_CONFIG")(), "Path to the configuration file"},
		"GOOBLA_CONFIG_DIR":      {"GOOBLA_CONFIG_DIR", configDir(), "Base directory for configuration and models"},
		"GOOBLA_NOHISTORY":       {"GOOBLA_NOHISTORY", NoHistory(), "Do not preserve readline history"},
		"GOOBLA_NOPRUNE":         {"GOOBLA_NOPRUNE", NoPrune(), "Do not prune model blobs on startup"},
		"GOOBLA_NUM_PARALLEL":    {"GOOBLA_NUM_PARALLEL", NumParallel(), "Maximum number of parallel requests"},
		"GOOBLA_ORIGINS":         {"GOOBLA_ORIGINS", AllowedOrigins(), "A comma separated list of allowed origins"},
		"GOOBLA_SCHED_SPREAD":    {"GOOBLA_SCHED_SPREAD", SchedSpread(), "Always schedule model across all GPUs"},
		"GOOBLA_MULTIUSER_CACHE": {"GOOBLA_MULTIUSER_CACHE", MultiUserCache(), "Optimize prompt caching for multi-user scenarios"},
		"GOOBLA_CONTEXT_LENGTH":  {"GOOBLA_CONTEXT_LENGTH", ContextLength(), "Context length to use unless otherwise specified (default: 4096)"},
		"GOOBLA_NEW_ENGINE":      {"GOOBLA_NEW_ENGINE", NewEngine(), "Enable the new Goobla engine"},
		"GOOBLA_PPROF":           {"GOOBLA_PPROF", PprofAddr(), "Bind pprof to this address or 'off' to disable"},

		// Informational
		"HTTP_PROXY":  {"HTTP_PROXY", String("HTTP_PROXY")(), "HTTP proxy"},
		"HTTPS_PROXY": {"HTTPS_PROXY", String("HTTPS_PROXY")(), "HTTPS proxy"},
		"NO_PROXY":    {"NO_PROXY", String("NO_PROXY")(), "No proxy"},
	}

	if runtime.GOOS != "windows" {
		// Windows environment variables are case-insensitive so there's no need to duplicate them
		ret["http_proxy"] = EnvVar{"http_proxy", String("http_proxy")(), "HTTP proxy"}
		ret["https_proxy"] = EnvVar{"https_proxy", String("https_proxy")(), "HTTPS proxy"}
		ret["no_proxy"] = EnvVar{"no_proxy", String("no_proxy")(), "No proxy"}
	}

	if runtime.GOOS != "darwin" {
		ret["CUDA_VISIBLE_DEVICES"] = EnvVar{"CUDA_VISIBLE_DEVICES", CudaVisibleDevices(), "Set which NVIDIA devices are visible"}
		ret["HIP_VISIBLE_DEVICES"] = EnvVar{"HIP_VISIBLE_DEVICES", HipVisibleDevices(), "Set which AMD devices are visible by numeric ID"}
		ret["ROCR_VISIBLE_DEVICES"] = EnvVar{"ROCR_VISIBLE_DEVICES", RocrVisibleDevices(), "Set which AMD devices are visible by UUID or numeric ID"}
		ret["GPU_DEVICE_ORDINAL"] = EnvVar{"GPU_DEVICE_ORDINAL", GpuDeviceOrdinal(), "Set which AMD devices are visible by numeric ID"}
		ret["HSA_OVERRIDE_GFX_VERSION"] = EnvVar{"HSA_OVERRIDE_GFX_VERSION", HsaOverrideGfxVersion(), "Override the gfx used for all detected AMD GPUs"}
		ret["GOOBLA_INTEL_GPU"] = EnvVar{"GOOBLA_INTEL_GPU", IntelGPU(), "Enable experimental Intel GPU detection"}
	}

	return ret
}

func Values() map[string]string {
	vals := make(map[string]string)
	for k, v := range AsMap() {
		vals[k] = fmt.Sprintf("%v", v.Value)
	}
	return vals
}

// Var returns an environment variable stripped of leading and trailing quotes or spaces
func Var(key string) string {
	return strings.Trim(strings.TrimSpace(os.Getenv(key)), "\"'")
}
