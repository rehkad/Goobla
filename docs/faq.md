# FAQ

## How can I upgrade Goobla?

Goobla on macOS and Windows will automatically download updates. Click on the taskbar or menubar item and then click "Restart to update" to apply the update. Updates can also be installed by downloading the latest version [manually](https://goobla.com/download/).

On Linux, re-run the install script:

```shell
curl -fsSL https://goobla.com/install.sh | sh
```

## How can I view the logs?

Review the [Troubleshooting](./troubleshooting.md) docs for more about using logs.

## Is my GPU compatible with Goobla?

Please refer to the [GPU docs](./gpu.md).

## How can I specify the context window size?

By default, Goobla uses a context window size of 4096 tokens. 

This can be overridden with the `GOOBLA_CONTEXT_LENGTH` environment variable. For example, to set the default context window to 8K, use: 

```shell
GOOBLA_CONTEXT_LENGTH=8192 goobla serve
```

To change this when using `goobla run`, use `/set parameter`:

```shell
/set parameter num_ctx 4096
```

When using the API, specify the `num_ctx` parameter:

```shell
curl http://localhost:11434/api/generate -d '{
  "model": "llama3.2",
  "prompt": "Why is the sky blue?",
  "options": {
    "num_ctx": 4096
  }
}'
```

## How can I tell if my model was loaded onto the GPU?

Use the `goobla ps` command to see what models are currently loaded into memory.

```shell
goobla ps
```

> **Output**:
>
> ```
> NAME      	ID          	SIZE 	PROCESSOR	UNTIL
> llama3:70b	bcfb190ca3a7	42 GB	100% GPU 	4 minutes from now
> ```

The `Processor` column will show which memory the model was loaded in to:
* `100% GPU` means the model was loaded entirely into the GPU
* `100% CPU` means the model was loaded entirely in system memory
* `48%/52% CPU/GPU` means the model was loaded partially onto both the GPU and into system memory

## How do I configure Goobla server?

Goobla server can be configured with environment variables.

### Setting environment variables on Mac

If Goobla is run as a macOS application, environment variables should be set using `launchctl`:

1. For each environment variable, call `launchctl setenv`.

    ```bash
    launchctl setenv GOOBLA_HOST "0.0.0.0:11434"
    ```

2. Restart Goobla application.

### Setting environment variables on Linux

If Goobla is run as a systemd service, environment variables should be set using `systemctl`:

1. Edit the systemd service by calling `systemctl edit goobla.service`. This will open an editor.

2. For each environment variable, add a line `Environment` under section `[Service]`:

    ```ini
    [Service]
    Environment="GOOBLA_HOST=0.0.0.0:11434"
    ```

3. Save and exit.

4. Reload `systemd` and restart Goobla:

   ```shell
   systemctl daemon-reload
   systemctl restart goobla
   ```

### Setting environment variables on Windows

On Windows, Goobla inherits your user and system environment variables.

1. First Quit Goobla by clicking on it in the task bar.

2. Start the Settings (Windows 11) or Control Panel (Windows 10) application and search for _environment variables_.

3. Click on _Edit environment variables for your account_.

4. Edit or create a new variable for your user account for `GOOBLA_HOST`, `GOOBLA_MODELS`, etc.

5. Click OK/Apply to save.

6. Start the Goobla application from the Windows Start menu.

## How do I use Goobla behind a proxy?

Goobla pulls models from the Internet and may require a proxy server to access the models. Use `HTTPS_PROXY` to redirect outbound requests through the proxy. Ensure the proxy certificate is installed as a system certificate. Refer to the section above for how to use environment variables on your platform.

> [!NOTE]
> Avoid setting `HTTP_PROXY`. Goobla does not use HTTP for model pulls, only HTTPS. Setting `HTTP_PROXY` may interrupt client connections to the server.

### How do I use Goobla behind a proxy in Docker?

The Goobla Docker container image can be configured to use a proxy by passing `-e HTTPS_PROXY=https://proxy.example.com` when starting the container.

Alternatively, the Docker daemon can be configured to use a proxy. Instructions are available for Docker Desktop on [macOS](https://docs.docker.com/desktop/settings/mac/#proxies), [Windows](https://docs.docker.com/desktop/settings/windows/#proxies), and [Linux](https://docs.docker.com/desktop/settings/linux/#proxies), and Docker [daemon with systemd](https://docs.docker.com/config/daemon/systemd/#httphttps-proxy).

Ensure the certificate is installed as a system certificate when using HTTPS. This may require a new Docker image when using a self-signed certificate.

```dockerfile
FROM goobla/goobla
COPY my-ca.pem /usr/local/share/ca-certificates/my-ca.crt
RUN update-ca-certificates
```

Build and run this image:

```shell
docker build -t goobla-with-ca .
docker run -d -e HTTPS_PROXY=https://my.proxy.example.com -p 11434:11434 goobla-with-ca
```

## Does Goobla send my prompts and answers back to goobla.com?

No. Goobla runs locally, and conversation data does not leave your machine.

## How can I expose Goobla on my network?

Goobla binds 127.0.0.1 port 11434 by default. Change the bind address with the `GOOBLA_HOST` environment variable.

Refer to the section [above](#how-do-i-configure-goobla-server) for how to set environment variables on your platform.

## Where is the configuration stored?

- macOS: `~/Library/Application Support/Goobla/config.json`
- Linux: `~/.config/goobla/config.json` (root: `/etc/goobla/config.json`)
- Windows: `%AppData%\Goobla\config.json`

Set the `GOOBLA_CONFIG` environment variable to override the location. The base
directory can be changed with `GOOBLA_CONFIG_DIR`.

## How can I use Goobla with a proxy server?

Goobla runs an HTTP server and can be exposed using a proxy server such as Nginx. To do so, configure the proxy to forward requests and optionally set required headers (if not exposing Goobla on the network). For example, with Nginx:

```nginx
server {
    listen 80;
    server_name example.com;  # Replace with your domain or IP
    location / {
        proxy_pass http://localhost:11434;
        proxy_set_header Host localhost:11434;
    }
}
```

## How can I use Goobla with ngrok?

Goobla can be accessed using a range of tools for tunneling tools. For example with Ngrok:

```shell
ngrok http 11434 --host-header="localhost:11434"
```

## How can I use Goobla with Cloudflare Tunnel?

To use Goobla with Cloudflare Tunnel, use the `--url` and `--http-host-header` flags:

```shell
cloudflared tunnel --url http://localhost:11434 --http-host-header="localhost:11434"
```

## How can I allow additional web origins to access Goobla?

Goobla allows cross-origin requests from `127.0.0.1` and `0.0.0.0` by default. Additional origins can be configured with `GOOBLA_ORIGINS`.

For browser extensions, you'll need to explicitly allow the extension's origin pattern. Set `GOOBLA_ORIGINS` to include `chrome-extension://*`, `moz-extension://*`, and `safari-web-extension://*` if you wish to allow all browser extensions access, or specific extensions as needed:

```
# Allow all Chrome, Firefox, and Safari extensions
GOOBLA_ORIGINS=chrome-extension://*,moz-extension://*,safari-web-extension://* goobla serve
```

Refer to the section [above](#how-do-i-configure-goobla-server) for how to set environment variables on your platform.

## Where are models stored?

- macOS: `~/.goobla/models`
- Linux: `~/.config/goobla/models`
- Windows: `%AppData%\Goobla\models`

### How do I set them to a different location?

If a different directory needs to be used, set the environment variable `GOOBLA_MODELS` to the chosen directory.

> Note: on Linux using the standard installer, the `goobla` user needs read and write access to the specified directory. To assign the directory to the `goobla` user run `sudo chown -R goobla:goobla <directory>`.

Refer to the section [above](#how-do-i-configure-goobla-server) for how to set environment variables on your platform.

## How can I use Goobla in Visual Studio Code?

There is already a large collection of plugins available for VSCode as well as other editors that leverage Goobla. See the list of [extensions & plugins](https://github.com/goobla/goobla#extensions--plugins) at the bottom of the main repository readme.

## How do I use Goobla with GPU acceleration in Docker?

The Goobla Docker container can be configured with GPU acceleration in Linux or Windows (with WSL2). This requires the [nvidia-container-toolkit](https://github.com/NVIDIA/nvidia-container-toolkit). See [goobla/goobla](https://hub.docker.com/r/goobla/goobla) for more details.

GPU acceleration is not available for Docker Desktop in macOS due to the lack of GPU passthrough and emulation.

## Why is networking slow in WSL2 on Windows 10?

This can impact both installing Goobla, as well as downloading models.

Open `Control Panel > Networking and Internet > View network status and tasks` and click on `Change adapter settings` on the left panel. Find the `vEthernel (WSL)` adapter, right click and select `Properties`.
Click on `Configure` and open the `Advanced` tab. Search through each of the properties until you find `Large Send Offload Version 2 (IPv4)` and `Large Send Offload Version 2 (IPv6)`. *Disable* both of these
properties.

## How can I preload a model into Goobla to get faster response times?

If you are using the API you can preload a model by sending the Goobla server an empty request. This works with both the `/api/generate` and `/api/chat` API endpoints.

To preload the mistral model using the generate endpoint, use:

```shell
curl http://localhost:11434/api/generate -d '{"model": "mistral"}'
```

To use the chat completions endpoint, use:

```shell
curl http://localhost:11434/api/chat -d '{"model": "mistral"}'
```

To preload a model using the CLI, use the command:

```shell
goobla run llama3.2 ""
```

## How do I keep a model loaded in memory or make it unload immediately?

By default models are kept in memory for 5 minutes before being unloaded. This allows for quicker response times if you're making numerous requests to the LLM. If you want to immediately unload a model from memory, use the `goobla stop` command:

```shell
goobla stop llama3.2
```

If you're using the API, use the `keep_alive` parameter with the `/api/generate` and `/api/chat` endpoints to set the amount of time that a model stays in memory. The `keep_alive` parameter can be set to:
* a duration string (such as "10m" or "24h")
* a number in seconds (such as 3600)
* any negative number which will keep the model loaded in memory (e.g. -1 or "-1m")
* '0' which will unload the model immediately after generating a response

For example, to preload a model and leave it in memory use:

```shell
curl http://localhost:11434/api/generate -d '{"model": "llama3.2", "keep_alive": -1}'
```

To unload the model and free up memory use:

```shell
curl http://localhost:11434/api/generate -d '{"model": "llama3.2", "keep_alive": 0}'
```

Alternatively, you can change the amount of time all models are loaded into memory by setting the `GOOBLA_KEEP_ALIVE` environment variable when starting the Goobla server. The `GOOBLA_KEEP_ALIVE` variable uses the same parameter types as the `keep_alive` parameter types mentioned above. Refer to the section explaining [how to configure the Goobla server](#how-do-i-configure-goobla-server) to correctly set the environment variable.

The `keep_alive` API parameter with the `/api/generate` and `/api/chat` API endpoints will override the `GOOBLA_KEEP_ALIVE` setting.

## How do I manage the maximum number of requests the Goobla server can queue?

If too many requests are sent to the server, it will respond with a 503 error indicating the server is overloaded.  You can adjust how many requests may be queue by setting `GOOBLA_MAX_QUEUE`.

## How does Goobla handle concurrent requests?

Goobla supports two levels of concurrent processing.  If your system has sufficient available memory (system memory when using CPU inference, or VRAM for GPU inference) then multiple models can be loaded at the same time.  For a given model, if there is sufficient available memory when the model is loaded, it is configured to allow parallel request processing.

If there is insufficient available memory to load a new model request while one or more models are already loaded, all new requests will be queued until the new model can be loaded.  As prior models become idle, one or more will be unloaded to make room for the new model.  Queued requests will be processed in order.  When using GPU inference new models must be able to completely fit in VRAM to allow concurrent model loads.

Parallel request processing for a given model results in increasing the context size by the number of parallel requests.  For example, a 2K context with 4 parallel requests will result in an 8K context and additional memory allocation.

The following server settings may be used to adjust how Goobla handles concurrent requests on most platforms:

- `GOOBLA_MAX_LOADED_MODELS` - The maximum number of models that can be loaded concurrently provided they fit in available memory.  The default is 3 * the number of GPUs or 3 for CPU inference.
- `GOOBLA_NUM_PARALLEL` - The maximum number of parallel requests each model will process at the same time.  The default will auto-select either 4 or 1 based on available memory.
- `GOOBLA_MAX_QUEUE` - The maximum number of requests Goobla will queue when busy before rejecting additional requests. The default is 512

Note: Windows with Radeon GPUs currently default to 1 model maximum due to limitations in ROCm v5.7 for available VRAM reporting.  Once ROCm v6.2 is available, Windows Radeon will follow the defaults above.  You may enable concurrent model loads on Radeon on Windows, but ensure you don't load more models than will fit into your GPUs VRAM.

## How does Goobla load models on multiple GPUs?

When loading a new model, Goobla evaluates the required VRAM for the model against what is currently available.  If the model will entirely fit on any single GPU, Goobla will load the model on that GPU.  This typically provides the best performance as it reduces the amount of data transferring across the PCI bus during inference.  If the model does not fit entirely on one GPU, then it will be spread across all the available GPUs.

## How can I enable Flash Attention?

Flash Attention is a feature of most modern models that can significantly reduce memory usage as the context size grows.  To enable Flash Attention, set the `GOOBLA_FLASH_ATTENTION` environment variable to `1` when starting the Goobla server.

## How can I set the quantization type for the K/V cache?

The K/V context cache can be quantized to significantly reduce memory usage when Flash Attention is enabled.

To use quantized K/V cache with Goobla you can set the following environment variable:

- `GOOBLA_KV_CACHE_TYPE` - The quantization type for the K/V cache.  Default is `f16`.

> Note: Currently this is a global option - meaning all models will run with the specified quantization type.

The currently available K/V cache quantization types are:

- `f16` - high precision and memory usage (default).
- `q8_0` - 8-bit quantization, uses approximately 1/2 the memory of `f16` with a very small loss in precision, this usually has no noticeable impact on the model's quality (recommended if not using f16).
- `q4_0` - 4-bit quantization, uses approximately 1/4 the memory of `f16` with a small-medium loss in precision that may be more noticeable at higher context sizes.

How much the cache quantization impacts the model's response quality will depend on the model and the task.  Models that have a high GQA count (e.g. Qwen2) may see a larger impact on precision from quantization than models with a low GQA count.

You may need to experiment with different quantization types to find the best balance between memory usage and quality.
