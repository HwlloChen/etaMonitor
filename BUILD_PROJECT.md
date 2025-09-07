# etaMonitor Project Build Guide

This project uses a front-end and back-end separation architecture and supports one-click fully automated builds. You can use the Makefile in the project root directory for all common operations.

## Version Information

Version number, build time, and Git commit information are automatically injected during the build. You can specify the version number through the VERSION environment variable:

```sh
# Build with default version number
make

# Build with specified version number
VERSION=1.2.0 make
```

Version information will be displayed at startup:

```
=== etaMonitor 1.2.0 ===
Build Time: 2025-08-14T21:30:45+0800
Git Commit: a1b2c3d
========================
```

## Dependencies

- Go 1.20 or higher
- Node.js and npm
- Linux/macOS/WSL (Windows users recommended to use WSL)
- tar utility (required for release packaging)

## One-Click Build

Execute in the project root directory:

```sh
make
```

Equivalent to:

```sh
make build
```

This command will automatically:

1. Enter the backend directory
2. Build the frontend (automatically install dependencies and build)
3. Copy frontend artifacts to backend embedding directory
4. Build the backend executable `etamonitor` (output to output directory)

## Running

After the build is complete, run directly:

```sh
./output/etamonitor
```

Or use:

```sh
make run
```

(Equivalent to entering the output directory and executing ./etamonitor)

## Release Packaging

### Create Multi-Platform Release Packages

Use the following command to create release packages for multiple operating systems and architectures:

```sh
make release
```

Or specify a version number:

```sh
VERSION=1.2.0 make release
```

This command will:

1. Build frontend resources
2. Cross-compile binary files for the following platforms:
   - **Linux**: amd64, arm64, arm, 386
   - **Windows**: amd64, 386, arm64
   - **macOS**: amd64 (Intel), arm64 (Apple Silicon)
   - **FreeBSD**: amd64, arm64

3. Create tar.gz compressed packages for each platform containing the following files:
   - Corresponding platform executable (`etamonitor` or `etamonitor.exe`)
   - `LICENSE` file
   - `README.md` file

4. Generated compressed packages are named in the format: `etamonitor-{version}-{os}-{arch}.tar.gz`

### Release Package Examples

After the build is complete, the `release/` directory will contain files similar to the following:

```
release/
├── etamonitor-1.2.0-linux-amd64.tar.gz
├── etamonitor-1.2.0-linux-arm64.tar.gz
├── etamonitor-1.2.0-linux-arm.tar.gz
├── etamonitor-1.2.0-linux-386.tar.gz
├── etamonitor-1.2.0-windows-amd64.tar.gz
├── etamonitor-1.2.0-windows-386.tar.gz
├── etamonitor-1.2.0-windows-arm64.tar.gz
├── etamonitor-1.2.0-darwin-amd64.tar.gz
├── etamonitor-1.2.0-darwin-arm64.tar.gz
├── etamonitor-1.2.0-freebsd-amd64.tar.gz
└── etamonitor-1.2.0-freebsd-arm64.tar.gz
```

Users can download the corresponding platform's compressed package and extract it for use:

```sh
# Example: Linux AMD64 users
tar -xzf etamonitor-1.2.0-linux-amd64.tar.gz
cd etamonitor-1.2.0-linux-amd64/
./etamonitor
```

## Other Common Commands

- Build frontend only:

  ```sh
  make frontend
  ```

- Build backend only:

  ```sh
  make backend
  ```

- Clean all build artifacts:

  ```sh
  make clean
  ```

  Note: `make clean` will clean `output/`, `release/`, and frontend build artifacts simultaneously.

## Directory Structure

- `frontend/`: Frontend source code
- `backend/`: Backend source code and Makefile
- `backend/internal/static/frontend-dist/`: Frontend build artifacts, automatically embedded by go:embed
- `output/etamonitor`: Backend binary file for development builds
- `release/`: Multi-platform compressed packages for release packaging

## Development Workflow Recommendations

1. **Daily Development**: Use `make` for quick builds and testing
2. **Local Testing**: Use `make run` to start the service
3. **Version Release**: Use `VERSION=x.y.z make release` to create release packages
4. **Environment Cleanup**: Use `make clean` to clean all build artifacts

## Common Issues

- **go:embed cannot find files**: Please ensure `make frontend` has been executed and there is content under `backend/internal/static/frontend-dist/`.
- **Node/npm not installed**: Please install Node.js and npm first.
- **Port occupied**: Please check if port 8080 is occupied by other programs (you can specify the port number through the configuration file).
- **Version information shows unknown**: Please ensure building in a Git repository, otherwise Git commit information cannot be obtained.
- **Cross-compilation failure**: Some platforms may fail to compile due to CGO dependencies, which is normal. Successful platforms will be packaged normally.
- **Release package too large**: If the binary file is too large, you can add `-s -w` parameters to GO_LDFLAGS to reduce file size:

  ```makefile
  GO_LDFLAGS := -s -w -X "etamonitor/internal/config.Version=$(VERSION)" ...
  ```

- **tar command not found**: Windows users please execute in WSL environment, or install tar utility.

## Version Management Recommendations

- Use Semantic Versioning: `major.minor.patch`
- Major version: Incompatible API changes
- Minor version: Backward-compatible feature additions
- Patch version: Backward-compatible bug fixes
