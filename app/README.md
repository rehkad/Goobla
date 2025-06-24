# Goobla App

## Linux

The Linux desktop application is built using Electron. You will need the
following tools installed:

- [Go](https://go.dev/doc/install) (1.21 or newer)
- [Node.js](https://nodejs.org/) with `npm`

From the root of the repository build the `goobla` binary and launch the
desktop UI:

```bash
go build .
cd macapp
npm install
npm start
```

Packaging Linux installers is not currently supported. Running `npm start`
launches the application for development.

## MacOS

Building the macOS desktop application requires:

- [Go](https://go.dev/doc/install) (1.21 or newer)
- [Node.js](https://nodejs.org/) with `npm`
- Xcode command line tools

Run the build script from the repository root:

```bash
./scripts/build_darwin.sh build
./scripts/build_darwin.sh macapp
```

If you have a valid Apple developer certificate you can also sign and notarize
the application by setting the `APPLE_IDENTITY`, `APPLE_ID`,
`APPLE_PASSWORD`, and `APPLE_TEAM_ID` environment variables and running:

```bash
./scripts/build_darwin.sh sign macapp
```

The packaged application will be written to `dist/Goobla-darwin.zip`.

## Windows

If you want to build the installer, youll need to install
- https://jrsoftware.org/isinfo.php


In the top directory of this repo, run the following powershell script
to build the goobla CLI, goobla app, and goobla installer.

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1
```
