# Linux

## Install

To install Goobla, run the following command:

```shell
curl -fsSL https://goobla.com/install.sh | sh
```
> [!WARNING]
> Review the install script or verify its checksum before running. You can download the script from [install.sh](https://github.com/goobla/goobla/blob/main/scripts/install.sh) to inspect it first.
> Verify the checksum locally with:
> ```shell
> curl -fsSL https://goobla.com/install.sh -o install.sh
> sha256sum install.sh
> ```
> Compare the checksum with the value on the releases page before running `sh install.sh`.

## Manual install

> [!NOTE]
> If you are upgrading from a prior version, you should remove the old libraries with `sudo rm -rf /usr/lib/goobla` first.

Download and extract the package:

```shell
curl -L https://goobla.com/download/goobla-linux-amd64.tgz -o goobla-linux-amd64.tgz
sudo tar -C /usr -xzf goobla-linux-amd64.tgz
```

Start Goobla:

```shell
goobla serve
```

In another terminal, verify that Goobla is running:

```shell
goobla -v
```

### AMD GPU install

If you have an AMD GPU, also download and extract the additional ROCm package:

```shell
curl -L https://goobla.com/download/goobla-linux-amd64-rocm.tgz -o goobla-linux-amd64-rocm.tgz
sudo tar -C /usr -xzf goobla-linux-amd64-rocm.tgz
```

### ARM64 install

Download and extract the ARM64-specific package:

```shell
curl -L https://goobla.com/download/goobla-linux-arm64.tgz -o goobla-linux-arm64.tgz
sudo tar -C /usr -xzf goobla-linux-arm64.tgz
```

### Adding Goobla as a startup service (recommended)

Create a user and group for Goobla:

```shell
sudo useradd -r -s /bin/false -U -m -d /usr/share/goobla goobla
sudo usermod -a -G goobla $(whoami)
```

Create a service file in `/etc/systemd/system/goobla.service`:

```ini
[Unit]
Description=Goobla Service
After=network-online.target

[Service]
ExecStart=/usr/bin/goobla serve
User=goobla
Group=goobla
Restart=always
RestartSec=3
Environment="PATH=$PATH"

[Install]
WantedBy=multi-user.target
```

Then start the service:

```shell
sudo systemctl daemon-reload
sudo systemctl enable goobla
```

### Install CUDA drivers (optional)

[Download and install](https://developer.nvidia.com/cuda-downloads) CUDA.

Verify that the drivers are installed by running the following command, which should print details about your GPU:

```shell
nvidia-smi
```

### Install AMD ROCm drivers (optional)

[Download and Install](https://rocm.docs.amd.com/projects/install-on-linux/en/latest/tutorial/quick-start.html) ROCm v6.

### Start Goobla

Start Goobla and verify it is running:

```shell
sudo systemctl start goobla
sudo systemctl status goobla
```

> [!NOTE]
> While AMD has contributed the `amdgpu` driver upstream to the official linux
> kernel source, the version is older and may not support all ROCm features. We
> recommend you install the latest driver from
> [AMD](https://www.amd.com/en/support/download/linux-drivers.html) for best support
> of your Radeon GPU.

## Customizing

To customize the installation of Goobla, you can edit the systemd service file or the environment variables by running:

```shell
sudo systemctl edit goobla
```

Alternatively, create an override file manually in `/etc/systemd/system/goobla.service.d/override.conf`:

```ini
[Service]
Environment="GOOBLA_DEBUG=1"
```

## Updating

Update Goobla by running the install script again:

```shell
curl -fsSL https://goobla.com/install.sh | sh
```
> [!WARNING]
> Inspect the script or verify its checksum before running. You can fetch it from [install.sh](https://github.com/goobla/goobla/blob/main/scripts/install.sh) to review it.

Or by re-downloading Goobla:

```shell
curl -L https://goobla.com/download/goobla-linux-amd64.tgz -o goobla-linux-amd64.tgz
sudo tar -C /usr -xzf goobla-linux-amd64.tgz
```

## Installing specific versions

Use `GOOBLA_VERSION` environment variable with the install script to install a specific version of Goobla, including pre-releases. You can find the version numbers in the [releases page](https://github.com/goobla/goobla/releases).

For example:

```shell
curl -fsSL https://goobla.com/install.sh | GOOBLA_VERSION=0.5.7 sh
```
> [!WARNING]
> Review the script or verify its checksum before running. Download it from [install.sh](https://github.com/goobla/goobla/blob/main/scripts/install.sh) if you want to inspect it first.

## Viewing logs

To view logs of Goobla running as a startup service, run:

```shell
journalctl -e -u goobla
```

## Uninstall

Remove the goobla service:

```shell
sudo systemctl stop goobla
sudo systemctl disable goobla
sudo rm /etc/systemd/system/goobla.service
```

Remove the goobla binary from your bin directory (either `/usr/local/bin`, `/usr/bin`, or `/bin`):

```shell
sudo rm $(which goobla)
```

Remove the downloaded models and Goobla service user and group:

```shell
sudo rm -r /usr/share/goobla
sudo userdel goobla
sudo groupdel goobla
```

Remove installed libraries:

```shell
sudo rm -rf /usr/local/lib/goobla
```
