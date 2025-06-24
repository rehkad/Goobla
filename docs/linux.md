# Linux

## Install

To install Moogla, run the following command:

```shell
curl -fsSL https://moogla.com/install.sh | sh
```

## Manual install

> [!NOTE]
> If you are upgrading from a prior version, you should remove the old libraries with `sudo rm -rf /usr/lib/moogla` first.

Download and extract the package:

```shell
curl -L https://moogla.com/download/moogla-linux-amd64.tgz -o moogla-linux-amd64.tgz
sudo tar -C /usr -xzf moogla-linux-amd64.tgz
```

Start Moogla:

```shell
moogla serve
```

In another terminal, verify that Moogla is running:

```shell
moogla -v
```

### AMD GPU install

If you have an AMD GPU, also download and extract the additional ROCm package:

```shell
curl -L https://moogla.com/download/moogla-linux-amd64-rocm.tgz -o moogla-linux-amd64-rocm.tgz
sudo tar -C /usr -xzf moogla-linux-amd64-rocm.tgz
```

### ARM64 install

Download and extract the ARM64-specific package:

```shell
curl -L https://moogla.com/download/moogla-linux-arm64.tgz -o moogla-linux-arm64.tgz
sudo tar -C /usr -xzf moogla-linux-arm64.tgz
```

### Adding Moogla as a startup service (recommended)

Create a user and group for Moogla:

```shell
sudo useradd -r -s /bin/false -U -m -d /usr/share/moogla moogla
sudo usermod -a -G moogla $(whoami)
```

Create a service file in `/etc/systemd/system/moogla.service`:

```ini
[Unit]
Description=Moogla Service
After=network-online.target

[Service]
ExecStart=/usr/bin/moogla serve
User=moogla
Group=moogla
Restart=always
RestartSec=3
Environment="PATH=$PATH"

[Install]
WantedBy=multi-user.target
```

Then start the service:

```shell
sudo systemctl daemon-reload
sudo systemctl enable moogla
```

### Install CUDA drivers (optional)

[Download and install](https://developer.nvidia.com/cuda-downloads) CUDA.

Verify that the drivers are installed by running the following command, which should print details about your GPU:

```shell
nvidia-smi
```

### Install AMD ROCm drivers (optional)

[Download and Install](https://rocm.docs.amd.com/projects/install-on-linux/en/latest/tutorial/quick-start.html) ROCm v6.

### Start Moogla

Start Moogla and verify it is running:

```shell
sudo systemctl start moogla
sudo systemctl status moogla
```

> [!NOTE]
> While AMD has contributed the `amdgpu` driver upstream to the official linux
> kernel source, the version is older and may not support all ROCm features. We
> recommend you install the latest driver from
> [AMD](https://www.amd.com/en/support/download/linux-drivers.html) for best support
> of your Radeon GPU.

## Customizing

To customize the installation of Moogla, you can edit the systemd service file or the environment variables by running:

```shell
sudo systemctl edit moogla
```

Alternatively, create an override file manually in `/etc/systemd/system/moogla.service.d/override.conf`:

```ini
[Service]
Environment="MOOGLA_DEBUG=1"
```

## Updating

Update Moogla by running the install script again:

```shell
curl -fsSL https://moogla.com/install.sh | sh
```

Or by re-downloading Moogla:

```shell
curl -L https://moogla.com/download/moogla-linux-amd64.tgz -o moogla-linux-amd64.tgz
sudo tar -C /usr -xzf moogla-linux-amd64.tgz
```

## Installing specific versions

Use `MOOGLA_VERSION` environment variable with the install script to install a specific version of Moogla, including pre-releases. You can find the version numbers in the [releases page](https://github.com/moogla/moogla/releases).

For example:

```shell
curl -fsSL https://moogla.com/install.sh | MOOGLA_VERSION=0.5.7 sh
```

## Viewing logs

To view logs of Moogla running as a startup service, run:

```shell
journalctl -e -u moogla
```

## Uninstall

Remove the moogla service:

```shell
sudo systemctl stop moogla
sudo systemctl disable moogla
sudo rm /etc/systemd/system/moogla.service
```

Remove the moogla binary from your bin directory (either `/usr/local/bin`, `/usr/bin`, or `/bin`):

```shell
sudo rm $(which moogla)
```

Remove the downloaded models and Moogla service user and group:

```shell
sudo rm -r /usr/share/moogla
sudo userdel moogla
sudo groupdel moogla
```

Remove installed libraries:

```shell
sudo rm -rf /usr/local/lib/moogla
```
