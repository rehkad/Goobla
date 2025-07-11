//go:build linux || windows

package discover

/*
#cgo linux LDFLAGS: -lrt -lpthread -ldl -lstdc++ -lm
#cgo windows LDFLAGS: -lpthread

#include "gpu_info.h"
*/
import "C"

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"unsafe"

	"github.com/goobla/goobla/envconfig"
	"github.com/goobla/goobla/format"
)

type cudaHandles struct {
	deviceCount int
	cudart      *C.cudart_handle_t
	nvcuda      *C.nvcuda_handle_t
	nvml        *C.nvml_handle_t
}

type oneapiHandles struct {
	oneapi      *C.oneapi_handle_t
	deviceCount int
}

const (
	// cudaMinimumMemory is the minimum VRAM required for CUDA GPUs.
	// This value was derived from profiling the smallest supported models
	// and represents the absolute minimum amount of free VRAM needed for
	// inference to succeed.
	cudaMinimumMemory = 457 * format.MebiByte
	// rocmMinimumMemory is the minimum VRAM required for ROCm GPUs.
	// It mirrors the CUDA requirement so behaviour is consistent across
	// vendors.
	rocmMinimumMemory = 457 * format.MebiByte
	// oneapiMinimumMemory is the minimum VRAM required for Intel GPUs when
	// using the oneAPI backend.  This matches the CUDA/ROCm thresholds.
	oneapiMinimumMemory = 457 * format.MebiByte
)

var (
	gpuMutex      sync.Mutex
	bootstrapped  bool
	cpus          []CPUInfo
	cudaGPUs      []CudaGPUInfo
	nvcudaLibPath string
	cudartLibPath string
	oneapiLibPath string
	nvmlLibPath   string
	rocmGPUs      []RocmGPUInfo
	oneapiGPUs    []OneapiGPUInfo

	// If any discovered GPUs are incompatible, report why
	unsupportedGPUs []UnsupportedGPUInfo

	// Keep track of errors during bootstrapping so that if GPUs are missing
	// they expected to be present this may explain why
	bootstrapErrors []error
)

// With our current CUDA compile flags, older than 5.0 will not work properly
// (string values used to allow ldflags overrides at build time)
var (
	CudaComputeMajorMin = "5"
	CudaComputeMinorMin = "0"
)

var RocmComputeMajorMin = "9"

// IGPUMemLimit is retained for backwards compatibility but should no longer be
// used for iGPU detection.  Integrated GPUs are now detected using GPU names
// obtained from the libraries or the PCI scan fallback.
const IGPUMemLimit = 1 * format.GibiByte

// Note: gpuMutex must already be held
func initCudaHandles() *cudaHandles {
	// If the build ships without GPU libraries this is effectively a CPU only
	// build.  Skip discovery in that case to avoid confusing warning logs.
	if isCPUOnlyBuild() {
		slog.Debug("CPU only build detected, skipping CUDA discovery")
		return &cudaHandles{}
	}

	cHandles := &cudaHandles{}
	// Short Circuit if we already know which library to use
	// ignore bootstrap errors in this case since we already recorded them
	if nvmlLibPath != "" {
		cHandles.nvml, _, _ = loadNVMLMgmt([]string{nvmlLibPath})
		return cHandles
	}
	if nvcudaLibPath != "" {
		cHandles.deviceCount, cHandles.nvcuda, _, _ = loadNVCUDAMgmt([]string{nvcudaLibPath})
		return cHandles
	}
	if cudartLibPath != "" {
		cHandles.deviceCount, cHandles.cudart, _, _ = loadCUDARTMgmt([]string{cudartLibPath})
		return cHandles
	}

	slog.Debug("searching for GPU discovery libraries for NVIDIA")
	var cudartMgmtPatterns []string

	// Aligned with driver, we can't carry as payloads
	nvcudaMgmtPatterns := NvcudaGlobs
	cudartMgmtPatterns = append(cudartMgmtPatterns, filepath.Join(LibGooblaPath, "cuda_v*", CudartMgmtName))
	cudartMgmtPatterns = append(cudartMgmtPatterns, CudartGlobs...)

	if len(NvmlGlobs) > 0 {
		nvmlLibPaths := FindGPULibs(NvmlMgmtName, NvmlGlobs)
		if len(nvmlLibPaths) > 0 {
			nvml, libPath, err := loadNVMLMgmt(nvmlLibPaths)
			if nvml != nil {
				slog.Debug("nvidia-ml loaded", "library", libPath)
				cHandles.nvml = nvml
				nvmlLibPath = libPath
			}
			if err != nil {
				bootstrapErrors = append(bootstrapErrors, err)
			}
		}
	}

	nvcudaLibPaths := FindGPULibs(NvcudaMgmtName, nvcudaMgmtPatterns)
	if len(nvcudaLibPaths) > 0 {
		deviceCount, nvcuda, libPath, err := loadNVCUDAMgmt(nvcudaLibPaths)
		if nvcuda != nil {
			slog.Debug("detected GPUs", "count", deviceCount, "library", libPath)
			cHandles.nvcuda = nvcuda
			cHandles.deviceCount = deviceCount
			nvcudaLibPath = libPath
			return cHandles
		}
		if err != nil {
			bootstrapErrors = append(bootstrapErrors, err)
		}
	}

	cudartLibPaths := FindGPULibs(CudartMgmtName, cudartMgmtPatterns)
	if len(cudartLibPaths) > 0 {
		deviceCount, cudart, libPath, err := loadCUDARTMgmt(cudartLibPaths)
		if cudart != nil {
			slog.Debug("detected GPUs", "library", libPath, "count", deviceCount)
			cHandles.cudart = cudart
			cHandles.deviceCount = deviceCount
			cudartLibPath = libPath
			return cHandles
		}
		if err != nil {
			bootstrapErrors = append(bootstrapErrors, err)
		}
	}

	return cHandles
}

// Note: gpuMutex must already be held
func initOneAPIHandles() *oneapiHandles {
	oHandles := &oneapiHandles{}

	// Short Circuit if we already know which library to use
	// ignore bootstrap errors in this case since we already recorded them
	if oneapiLibPath != "" {
		oHandles.deviceCount, oHandles.oneapi, _, _ = loadOneapiMgmt([]string{oneapiLibPath})
		return oHandles
	}

	oneapiLibPaths := FindGPULibs(OneapiMgmtName, OneapiGlobs)
	if len(oneapiLibPaths) > 0 {
		var err error
		oHandles.deviceCount, oHandles.oneapi, oneapiLibPath, err = loadOneapiMgmt(oneapiLibPaths)
		if err != nil {
			bootstrapErrors = append(bootstrapErrors, err)
		}
	}

	return oHandles
}

func GetCPUInfo() GpuInfoList {
	gpuMutex.Lock()
	if !bootstrapped {
		gpuMutex.Unlock()
		GetGPUInfo()
	} else {
		gpuMutex.Unlock()
	}
	return GpuInfoList{cpus[0].GpuInfo}
}

func GetGPUInfo() GpuInfoList {
	// TODO - consider exploring lspci (and equivalent on windows) to check for
	// GPUs so we can report warnings if we see Nvidia/AMD but fail to load the libraries
	gpuMutex.Lock()
	defer gpuMutex.Unlock()
	needRefresh := true
	var cHandles *cudaHandles
	var oHandles *oneapiHandles
	defer func() {
		if cHandles != nil {
			if cHandles.cudart != nil {
				C.cudart_release(*cHandles.cudart)
			}
			if cHandles.nvcuda != nil {
				C.nvcuda_release(*cHandles.nvcuda)
			}
			if cHandles.nvml != nil {
				C.nvml_release(*cHandles.nvml)
			}
		}
		if oHandles != nil {
			if oHandles.oneapi != nil {
				// TODO - is this needed?
				C.oneapi_release(*oHandles.oneapi)
			}
		}
	}()

	if !bootstrapped {
		slog.Info("looking for compatible GPUs")
		cudaComputeMajorMin, err := strconv.Atoi(CudaComputeMajorMin)
		if err != nil {
			slog.Error("invalid CudaComputeMajorMin setting", "value", CudaComputeMajorMin, "error", err)
		}
		cudaComputeMinorMin, err := strconv.Atoi(CudaComputeMinorMin)
		if err != nil {
			slog.Error("invalid CudaComputeMinorMin setting", "value", CudaComputeMinorMin, "error", err)
		}
		bootstrapErrors = []error{}
		needRefresh = false
		var memInfo C.mem_info_t

		mem, err := GetCPUMem()
		if err != nil {
			slog.Warn("error looking up system memory", "error", err)
		}

		details, err := GetCPUDetails()
		if err != nil {
			slog.Warn("failed to lookup CPU details", "error", err)
		}
		cpus = []CPUInfo{
			{
				GpuInfo: GpuInfo{
					memInfo: mem,
					Library: "cpu",
					ID:      "0",
				},
				CPUs: details,
			},
		}

		// Load ALL libraries
		cHandles = initCudaHandles()

		// NVIDIA
		for i := range cHandles.deviceCount {
			if cHandles.cudart != nil || cHandles.nvcuda != nil {
				gpuInfo := CudaGPUInfo{
					GpuInfo: GpuInfo{
						Library: "cuda",
					},
					index: i,
				}
				var driverMajor int
				var driverMinor int
				if cHandles.cudart != nil {
					C.cudart_bootstrap(*cHandles.cudart, C.int(i), &memInfo)
				} else {
					C.nvcuda_bootstrap(*cHandles.nvcuda, C.int(i), &memInfo)
					driverMajor = int(cHandles.nvcuda.driver_major)
					driverMinor = int(cHandles.nvcuda.driver_minor)
				}
				if memInfo.err != nil {
					slog.Info("error looking up nvidia GPU memory", "error", C.GoString(memInfo.err))
					C.free(unsafe.Pointer(memInfo.err))
					continue
				}
				gpuInfo.TotalMemory = uint64(memInfo.total)
				gpuInfo.FreeMemory = uint64(memInfo.free)
				gpuInfo.ID = C.GoString(&memInfo.gpu_id[0])
				gpuInfo.Compute = fmt.Sprintf("%d.%d", memInfo.major, memInfo.minor)
				gpuInfo.computeMajor = int(memInfo.major)
				gpuInfo.computeMinor = int(memInfo.minor)
				gpuInfo.MinimumMemory = cudaMinimumMemory
				gpuInfo.DriverMajor = driverMajor
				gpuInfo.DriverMinor = driverMinor
				variant := cudaVariant(gpuInfo)

				// Start with our bundled libraries
				if variant != "" {
					variantPath := filepath.Join(LibGooblaPath, "cuda_"+variant)
					if _, err := os.Stat(variantPath); err == nil {
						// Put the variant directory first in the search path to avoid runtime linking to the wrong library
						gpuInfo.DependencyPath = append([]string{variantPath}, gpuInfo.DependencyPath...)
					}
				}
				gpuInfo.Name = C.GoString(&memInfo.gpu_name[0])
				gpuInfo.Variant = variant

				if int(memInfo.major) < cudaComputeMajorMin || (int(memInfo.major) == cudaComputeMajorMin && int(memInfo.minor) < cudaComputeMinorMin) {
					unsupportedGPUs = append(unsupportedGPUs,
						UnsupportedGPUInfo{
							GpuInfo: gpuInfo.GpuInfo,
						})
					slog.Info(fmt.Sprintf("[%d] CUDA GPU is too old. Compute Capability detected: %d.%d", i, memInfo.major, memInfo.minor))
					continue
				}

				if isIntegratedGPU(gpuInfo.Name) {
					reason := "unsupported NVIDIA iGPU detected skipping"
					slog.Info(reason, "id", gpuInfo.ID, "name", gpuInfo.Name)
					unsupportedGPUs = append(unsupportedGPUs, UnsupportedGPUInfo{GpuInfo: gpuInfo.GpuInfo, Reason: reason})
					continue
				}

				if gpuInfo.TotalMemory < gpuInfo.MinimumMemory {
					reason := fmt.Sprintf("GPU memory below minimum: %s < %s", format.HumanBytes2(gpuInfo.TotalMemory), format.HumanBytes2(gpuInfo.MinimumMemory))
					slog.Info(reason, "gpu", gpuInfo.ID)
					unsupportedGPUs = append(unsupportedGPUs, UnsupportedGPUInfo{GpuInfo: gpuInfo.GpuInfo, Reason: reason})
					continue
				}

				// query the management library as well so we can record any skew between the two
				// which represents overhead on the GPU we must set aside on subsequent updates
				if cHandles.nvml != nil {
					uuid := C.CString(gpuInfo.ID)
					defer C.free(unsafe.Pointer(uuid))
					C.nvml_get_free(*cHandles.nvml, uuid, &memInfo.free, &memInfo.total, &memInfo.used)
					if memInfo.err != nil {
						slog.Warn("error looking up nvidia GPU memory", "error", C.GoString(memInfo.err))
						C.free(unsafe.Pointer(memInfo.err))
					} else {
						if memInfo.free != 0 && uint64(memInfo.free) > gpuInfo.FreeMemory {
							gpuInfo.OSOverhead = uint64(memInfo.free) - gpuInfo.FreeMemory
							slog.Info("detected OS VRAM overhead",
								"id", gpuInfo.ID,
								"library", gpuInfo.Library,
								"compute", gpuInfo.Compute,
								"driver", fmt.Sprintf("%d.%d", gpuInfo.DriverMajor, gpuInfo.DriverMinor),
								"name", gpuInfo.Name,
								"overhead", format.HumanBytes2(gpuInfo.OSOverhead),
							)
						}
					}
				}

				// TODO potentially sort on our own algorithm instead of what the underlying GPU library does...
				cudaGPUs = append(cudaGPUs, gpuInfo)
			}
		}

		// Intel
		if envconfig.IntelGPU() {
			oHandles = initOneAPIHandles()
			if oHandles != nil && oHandles.oneapi != nil {
				for d := range oHandles.oneapi.num_drivers {
					if oHandles.oneapi == nil {
						// shouldn't happen
						slog.Warn("nil oneapi handle with driver count", "count", int(oHandles.oneapi.num_drivers))
						continue
					}
					devCount := C.oneapi_get_device_count(*oHandles.oneapi, C.int(d))
					for i := range devCount {
						gpuInfo := OneapiGPUInfo{
							GpuInfo: GpuInfo{
								Library: "oneapi",
							},
							driverIndex: int(d),
							gpuIndex:    int(i),
						}
						// TODO - split bootstrapping from updating free memory
						C.oneapi_check_vram(*oHandles.oneapi, C.int(d), i, &memInfo)
						// leave a small reserve of VRAM for the MKL library used in the SYCL backend
						var totalFreeMem float64 = float64(memInfo.free) * 0.95
						memInfo.free = C.uint64_t(totalFreeMem)
						gpuInfo.TotalMemory = uint64(memInfo.total)
						gpuInfo.FreeMemory = uint64(memInfo.free)
						gpuInfo.ID = C.GoString(&memInfo.gpu_id[0])
						gpuInfo.Name = C.GoString(&memInfo.gpu_name[0])
						gpuInfo.DependencyPath = []string{LibGooblaPath}
						gpuInfo.MinimumMemory = oneapiMinimumMemory

						if isIntegratedGPU(gpuInfo.Name) {
							reason := "unsupported Intel iGPU detected skipping"
							slog.Info(reason, "id", gpuInfo.ID, "name", gpuInfo.Name)
							unsupportedGPUs = append(unsupportedGPUs, UnsupportedGPUInfo{GpuInfo: gpuInfo.GpuInfo, Reason: reason})
							continue
						}

						if gpuInfo.TotalMemory < gpuInfo.MinimumMemory {
							reason := fmt.Sprintf("GPU memory below minimum: %s < %s", format.HumanBytes2(gpuInfo.TotalMemory), format.HumanBytes2(gpuInfo.MinimumMemory))
							slog.Info(reason, "gpu", gpuInfo.ID)
							unsupportedGPUs = append(unsupportedGPUs, UnsupportedGPUInfo{GpuInfo: gpuInfo.GpuInfo, Reason: reason})
							continue
						}
						oneapiGPUs = append(oneapiGPUs, gpuInfo)
					}
				}
			}
		}

		rocmGPUs, err = AMDGetGPUInfo()
		if err != nil {
			bootstrapErrors = append(bootstrapErrors, err)
		}
		bootstrapped = true
		if len(cudaGPUs) == 0 && len(rocmGPUs) == 0 && len(oneapiGPUs) == 0 {
			slog.Info("no compatible GPUs were discovered")
			if len(bootstrapErrors) > 0 {
				if devices := scanPCIGPUs(); len(devices) > 0 {
					for _, d := range devices {
						slog.Info("pci scan detected gpu", "device", d)
					}
				}
			}
		}

		// verify we have runners for the discovered GPUs
		filter := func(in []CudaGPUInfo) []CudaGPUInfo {
			out := in[:0]
			for _, g := range in {
				if gpuHasRunner(g.GpuInfo) {
					out = append(out, g)
				} else {
					reason := "no runner available"
					slog.Info(reason, "gpu", g.ID, "runner", g.RunnerName())
					unsupportedGPUs = append(unsupportedGPUs, UnsupportedGPUInfo{GpuInfo: g.GpuInfo, Reason: reason})
				}
			}
			return out
		}
		cudaGPUs = filter(cudaGPUs)

		filterROCM := func(in []RocmGPUInfo) []RocmGPUInfo {
			out := in[:0]
			for _, g := range in {
				if gpuHasRunner(g.GpuInfo) {
					out = append(out, g)
				} else {
					reason := "no runner available"
					slog.Info(reason, "gpu", g.ID, "runner", g.RunnerName())
					unsupportedGPUs = append(unsupportedGPUs, UnsupportedGPUInfo{GpuInfo: g.GpuInfo, Reason: reason})
				}
			}
			return out
		}
		rocmGPUs = filterROCM(rocmGPUs)

		filterOneAPI := func(in []OneapiGPUInfo) []OneapiGPUInfo {
			out := in[:0]
			for _, g := range in {
				if gpuHasRunner(g.GpuInfo) {
					out = append(out, g)
				} else {
					reason := "no runner available"
					slog.Info(reason, "gpu", g.ID, "runner", g.RunnerName())
					unsupportedGPUs = append(unsupportedGPUs, UnsupportedGPUInfo{GpuInfo: g.GpuInfo, Reason: reason})
				}
			}
			return out
		}
		oneapiGPUs = filterOneAPI(oneapiGPUs)
	}

	// For detected GPUs, load library if not loaded

	// Refresh free memory usage
	if needRefresh {
		mem, err := GetCPUMem()
		if err != nil {
			slog.Warn("error looking up system memory", "error", err)
		} else {
			slog.Debug("updating system memory data",
				slog.Group(
					"before",
					"total", format.HumanBytes2(cpus[0].TotalMemory),
					"free", format.HumanBytes2(cpus[0].FreeMemory),
					"free_swap", format.HumanBytes2(cpus[0].FreeSwap),
				),
				slog.Group(
					"now",
					"total", format.HumanBytes2(mem.TotalMemory),
					"free", format.HumanBytes2(mem.FreeMemory),
					"free_swap", format.HumanBytes2(mem.FreeSwap),
				),
			)
			cpus[0].FreeMemory = mem.FreeMemory
			cpus[0].FreeSwap = mem.FreeSwap
		}

		var memInfo C.mem_info_t
		if cHandles == nil && len(cudaGPUs) > 0 {
			cHandles = initCudaHandles()
		}
		for i, gpu := range cudaGPUs {
			if cHandles.nvml != nil {
				uuid := C.CString(gpu.ID)
				defer C.free(unsafe.Pointer(uuid))
				C.nvml_get_free(*cHandles.nvml, uuid, &memInfo.free, &memInfo.total, &memInfo.used)
			} else if cHandles.cudart != nil {
				C.cudart_bootstrap(*cHandles.cudart, C.int(gpu.index), &memInfo)
			} else if cHandles.nvcuda != nil {
				C.nvcuda_get_free(*cHandles.nvcuda, C.int(gpu.index), &memInfo.free, &memInfo.total)
				memInfo.used = memInfo.total - memInfo.free
			} else {
				// shouldn't happen
				slog.Warn("no valid cuda library loaded to refresh vram usage")
				break
			}
			if memInfo.err != nil {
				slog.Warn("error looking up nvidia GPU memory", "error", C.GoString(memInfo.err))
				C.free(unsafe.Pointer(memInfo.err))
				continue
			}
			if memInfo.free == 0 {
				slog.Warn("error looking up nvidia GPU memory")
				continue
			}
			if cHandles.nvml != nil && gpu.OSOverhead > 0 {
				// When using the management library update based on recorded overhead
				memInfo.free -= C.uint64_t(gpu.OSOverhead)
			}
			slog.Debug("updating cuda memory data",
				"gpu", gpu.ID,
				"name", gpu.Name,
				"overhead", format.HumanBytes2(gpu.OSOverhead),
				slog.Group(
					"before",
					"total", format.HumanBytes2(gpu.TotalMemory),
					"free", format.HumanBytes2(gpu.FreeMemory),
				),
				slog.Group(
					"now",
					"total", format.HumanBytes2(uint64(memInfo.total)),
					"free", format.HumanBytes2(uint64(memInfo.free)),
					"used", format.HumanBytes2(uint64(memInfo.used)),
				),
			)
			cudaGPUs[i].FreeMemory = uint64(memInfo.free)
		}

		if oHandles == nil && len(oneapiGPUs) > 0 {
			oHandles = initOneAPIHandles()
		}
		for i, gpu := range oneapiGPUs {
			if oHandles.oneapi == nil {
				// shouldn't happen
				slog.Warn("nil oneapi handle with device count", "count", oHandles.deviceCount)
				continue
			}
			C.oneapi_check_vram(*oHandles.oneapi, C.int(gpu.driverIndex), C.int(gpu.gpuIndex), &memInfo)
			// leave a small reserve of VRAM for the SYCL runtime
			var totalFreeMem float64 = float64(memInfo.free) * 0.95
			memInfo.free = C.uint64_t(totalFreeMem)
			oneapiGPUs[i].FreeMemory = uint64(memInfo.free)
		}

		err = RocmGPUInfoList(rocmGPUs).RefreshFreeMemory()
		if err != nil {
			slog.Debug("problem refreshing ROCm free memory", "error", err)
		}
	}

	resp := []GpuInfo{}
	for _, gpu := range cudaGPUs {
		resp = append(resp, gpu.GpuInfo)
	}
	for _, gpu := range rocmGPUs {
		resp = append(resp, gpu.GpuInfo)
	}
	for _, gpu := range oneapiGPUs {
		resp = append(resp, gpu.GpuInfo)
	}
	if len(resp) == 0 {
		resp = append(resp, cpus[0].GpuInfo)
	}
	return resp
}

func FindGPULibs(baseLibName string, defaultPatterns []string) []string {
	// Multiple GPU libraries may exist, and some may not work, so keep trying until we exhaust them
	gpuLibPaths := []string{}
	slog.Debug("Searching for GPU library", "name", baseLibName)

	// search our bundled libraries first
	patterns := []string{filepath.Join(LibGooblaPath, baseLibName)}

	var ldPaths []string
	switch runtime.GOOS {
	case "windows":
		ldPaths = strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))
	case "linux":
		ldPaths = strings.Split(os.Getenv("LD_LIBRARY_PATH"), string(os.PathListSeparator))
	}

	// then search the system's LD_LIBRARY_PATH
	for _, p := range ldPaths {
		p, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		patterns = append(patterns, filepath.Join(p, baseLibName))
	}

	// finally, search the default patterns provided by the caller
	patterns = append(patterns, defaultPatterns...)
	slog.Debug("gpu library search", "globs", patterns)
	for _, pattern := range patterns {
		// Nvidia PhysX known to return bogus results
		if strings.Contains(pattern, "PhysX") {
			slog.Debug("skipping PhysX cuda library path", "path", pattern)
			continue
		}
		// Ignore glob discovery errors
		matches, _ := filepath.Glob(pattern)
		for _, match := range matches {
			// Resolve any links so we don't try the same lib multiple times
			// and weed out any dups across globs
			libPath := match
			tmp := match
			var err error
			for ; err == nil; tmp, err = os.Readlink(libPath) {
				if !filepath.IsAbs(tmp) {
					tmp = filepath.Join(filepath.Dir(libPath), tmp)
				}
				libPath = tmp
			}
			new := true
			for _, cmp := range gpuLibPaths {
				if cmp == libPath {
					new = false
					break
				}
			}
			if new {
				gpuLibPaths = append(gpuLibPaths, libPath)
			}
		}
	}
	slog.Debug("discovered GPU libraries", "paths", gpuLibPaths)
	return gpuLibPaths
}

// Bootstrap the runtime library
// Returns: num devices, handle, libPath, error
func loadCUDARTMgmt(cudartLibPaths []string) (int, *C.cudart_handle_t, string, error) {
	var resp C.cudart_init_resp_t
	resp.ch.verbose = getVerboseState()
	var err error
	for _, libPath := range cudartLibPaths {
		lib := C.CString(libPath)
		defer C.free(unsafe.Pointer(lib))
		C.cudart_init(lib, &resp)
		if resp.err != nil {
			err = fmt.Errorf("Unable to load cudart library %s: %s", libPath, C.GoString(resp.err))
			slog.Debug(err.Error())
			C.free(unsafe.Pointer(resp.err))
		} else {
			err = nil
			return int(resp.num_devices), &resp.ch, libPath, err
		}
	}
	return 0, nil, "", err
}

// Bootstrap the driver library
// Returns: num devices, handle, libPath, error
func loadNVCUDAMgmt(nvcudaLibPaths []string) (int, *C.nvcuda_handle_t, string, error) {
	var resp C.nvcuda_init_resp_t
	resp.ch.verbose = getVerboseState()
	var err error
	for _, libPath := range nvcudaLibPaths {
		lib := C.CString(libPath)
		defer C.free(unsafe.Pointer(lib))
		C.nvcuda_init(lib, &resp)
		if resp.err != nil {
			// Decide what log level based on the type of error message to help users understand why
			switch resp.cudaErr {
			case C.CUDA_ERROR_INSUFFICIENT_DRIVER, C.CUDA_ERROR_SYSTEM_DRIVER_MISMATCH:
				err = fmt.Errorf("version mismatch between driver and cuda driver library - reboot or upgrade may be required: library %s", libPath)
				slog.Warn(err.Error())
			case C.CUDA_ERROR_NO_DEVICE:
				err = fmt.Errorf("no nvidia devices detected by library %s", libPath)
				slog.Info(err.Error())
			case C.CUDA_ERROR_UNKNOWN:
				err = fmt.Errorf("unknown error initializing cuda driver library %s: %s. see https://github.com/goobla/goobla/blob/main/docs/troubleshooting.md for more information", libPath, C.GoString(resp.err))
				slog.Warn(err.Error())
			default:
				msg := C.GoString(resp.err)
				if strings.Contains(msg, "wrong ELF class") {
					slog.Debug("skipping 32bit library", "library", libPath)
				} else {
					err = fmt.Errorf("Unable to load cudart library %s: %s", libPath, C.GoString(resp.err))
					slog.Info(err.Error())
				}
			}
			C.free(unsafe.Pointer(resp.err))
		} else {
			err = nil
			return int(resp.num_devices), &resp.ch, libPath, err
		}
	}
	return 0, nil, "", err
}

// Bootstrap the management library
// Returns: handle, libPath, error
func loadNVMLMgmt(nvmlLibPaths []string) (*C.nvml_handle_t, string, error) {
	var resp C.nvml_init_resp_t
	resp.ch.verbose = getVerboseState()
	var err error
	for _, libPath := range nvmlLibPaths {
		lib := C.CString(libPath)
		defer C.free(unsafe.Pointer(lib))
		C.nvml_init(lib, &resp)
		if resp.err != nil {
			err = fmt.Errorf("Unable to load NVML management library %s: %s", libPath, C.GoString(resp.err))
			slog.Info(err.Error())
			C.free(unsafe.Pointer(resp.err))
		} else {
			err = nil
			return &resp.ch, libPath, err
		}
	}
	return nil, "", err
}

// bootstrap the Intel GPU library
// Returns: num devices, handle, libPath, error
func loadOneapiMgmt(oneapiLibPaths []string) (int, *C.oneapi_handle_t, string, error) {
	var resp C.oneapi_init_resp_t
	num_devices := 0
	resp.oh.verbose = getVerboseState()
	var err error
	for _, libPath := range oneapiLibPaths {
		lib := C.CString(libPath)
		defer C.free(unsafe.Pointer(lib))
		C.oneapi_init(lib, &resp)
		if resp.err != nil {
			err = fmt.Errorf("Unable to load oneAPI management library %s: %s", libPath, C.GoString(resp.err))
			slog.Debug(err.Error())
			C.free(unsafe.Pointer(resp.err))
		} else {
			err = nil
			for i := range resp.oh.num_drivers {
				num_devices += int(C.oneapi_get_device_count(resp.oh, C.int(i)))
			}
			return num_devices, &resp.oh, libPath, err
		}
	}
	return 0, nil, "", err
}

func getVerboseState() C.uint16_t {
	if envconfig.LogLevel() < slog.LevelInfo {
		return C.uint16_t(1)
	}
	return C.uint16_t(0)
}

// Given the list of GPUs this instantiation is targeted for,
// figure out the visible devices environment variable
//
// If different libraries are detected, the first one is what we use
func (l GpuInfoList) GetVisibleDevicesEnv() (string, string) {
	if len(l) == 0 {
		return "", ""
	}
	switch l[0].Library {
	case "cuda":
		return cudaGetVisibleDevicesEnv(l)
	case "rocm":
		return rocmGetVisibleDevicesEnv(l)
	case "oneapi":
		return oneapiGetVisibleDevicesEnv(l)
	default:
		slog.Debug("no filter required for library " + l[0].Library)
		return "", ""
	}
}

func GetSystemInfo() SystemInfo {
	gpus := GetGPUInfo()
	gpuMutex.Lock()
	defer gpuMutex.Unlock()
	discoveryErrors := []string{}
	for _, err := range bootstrapErrors {
		discoveryErrors = append(discoveryErrors, err.Error())
	}
	if len(gpus) == 1 && gpus[0].Library == "cpu" {
		gpus = []GpuInfo{}
	}

	return SystemInfo{
		System:          cpus[0],
		GPUs:            gpus,
		UnsupportedGPUs: unsupportedGPUs,
		DiscoveryErrors: discoveryErrors,
	}
}

// isCPUOnlyBuild returns true when no GPU libraries are present in the
// installation directory.  This is used to skip GPU discovery logic when the
// binary was built without GPU support.
func isCPUOnlyBuild() bool {
	entries, err := os.ReadDir(LibGooblaPath)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := strings.ToLower(e.Name())
		if strings.HasPrefix(name, "cuda_") || strings.HasPrefix(name, "rocm") || strings.HasPrefix(name, "oneapi") {
			return false
		}
	}
	return true
}

// parseLspci parses output from lspci or wmic and returns any lines referencing
// common GPU vendors.
func parseLspci(out string) []string {
	devices := []string{}
	for _, line := range strings.Split(out, "\n") {
		l := strings.ToLower(strings.TrimSpace(line))
		if l == "" {
			continue
		}
		if strings.Contains(l, "nvidia") || strings.Contains(l, "amd") || strings.Contains(l, "advanced micro devices") || strings.Contains(l, "intel") {
			devices = append(devices, strings.TrimSpace(line))
		}
	}
	return devices
}

// scanPCIGPUs attempts to detect GPUs via lspci on linux or wmic on windows.
func scanPCIGPUs() []string {
	var out []byte
	var err error
	switch runtime.GOOS {
	case "linux":
		out, err = exec.Command("lspci").Output()
	case "windows":
		out, err = exec.Command("wmic", "path", "win32_VideoController", "get", "name").Output()
	default:
		return nil
	}
	if err != nil {
		slog.Debug("pci scan failed", "error", err)
		return nil
	}
	return parseLspci(string(out))
}

// isIntegratedGPU attempts to determine if a GPU is integrated based on its name.
func isIntegratedGPU(name string) bool {
	n := strings.ToLower(name)
	if strings.Contains(n, "intel") && !strings.Contains(n, "arc") {
		return true
	}
	if strings.Contains(n, "radeon(tm)") && strings.Contains(n, "graphics") {
		return true
	}
	if strings.Contains(n, "vega") && strings.Contains(n, "graphics") {
		return true
	}
	return false
}

// gpuHasRunner returns true if a runner directory exists for the GPU.
func gpuHasRunner(g GpuInfo) bool {
	path := filepath.Join(LibGooblaPath, g.RunnerName())
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}
