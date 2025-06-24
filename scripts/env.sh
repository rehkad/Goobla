# Common environment setup across build*.sh scripts

export VERSION=${VERSION:-$(git describe --tags --first-parent --abbrev=7 --long --dirty --always | sed -e "s/^v//g")}
export GOFLAGS="'-mod=vendor -ldflags=-w -s \"-X=github.com/goobla/goobla/version.Version=$VERSION\" \"-X=github.com/goobla/goobla/server.mode=release\"'"
# TODO - consider `docker buildx ls --format=json` to autodiscover platform capability
PLATFORM=${PLATFORM:-"linux/arm64,linux/amd64"}
DOCKER_ORG=${DOCKER_ORG:-"goobla"}
FINAL_IMAGE_REPO=${FINAL_IMAGE_REPO:-"${DOCKER_ORG}/goobla"}
GOOBLA_COMMON_BUILD_ARGS="--build-arg=VERSION \
    --build-arg=GOFLAGS \
    --build-arg=GOOBLA_CUSTOM_CPU_DEFS \
    --build-arg=GOOBLA_SKIP_CUDA_GENERATE \
    --build-arg=GOOBLA_SKIP_CUDA_12_GENERATE \
    --build-arg=CUDA_V12_ARCHITECTURES \
    --build-arg=GOOBLA_SKIP_ROCM_GENERATE \
    --build-arg=GOOBLA_FAST_BUILD \
    --build-arg=CUSTOM_CPU_FLAGS \
    --build-arg=GPU_RUNNER_CPU_FLAGS \
    --build-arg=AMDGPU_TARGETS"

echo "Building Goobla"
echo "VERSION=$VERSION"
echo "PLATFORM=$PLATFORM"