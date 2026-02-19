# GitHub Actions Workflows

This directory contains GitHub Actions workflows for the CRM Relay Server project.

## Docker Build and Push Workflow

**File**: `docker-build-push.yml`

### What It Does

This workflow automatically builds and pushes **multi-architecture Docker images** to GitHub Container Registry (GHCR) whenever you:
- Push to `main` or `master` branch
- Create a tag (e.g., `v1.0.0`)
- Open a pull request

### Supported Architectures

The workflow builds Docker images for multiple platforms:

| Platform | Architecture | Use Case |
|----------|-------------|----------|
| `linux/amd64` | x86_64 | Standard servers, cloud VMs, most desktops |
| `linux/arm64` | ARM64 | Apple Silicon Macs, ARM servers, Raspberry Pi 4/5 |
| `linux/arm/v7` | ARMv7 | Raspberry Pi 3 and older ARM devices |

### Images Built

- **Relay Server**: `ghcr.io/QuantumSolver/crm-relay/relay-server`
- **Relay Client**: `ghcr.io/QuantumSolver/crm-relay/relay-client`

Each image contains binaries for all supported architectures. Docker will automatically pull the correct architecture for your system.

### Authentication

**No additional credentials needed!** 🎉

The workflow uses GitHub's built-in `GITHUB_TOKEN` secret, which is automatically provided to workflows. This token has permission to:
- Read repository contents
- Write packages (push images to GHCR)

The workflow automatically authenticates as `${{ github.actor }}` (your GitHub username), so it uses your GitHub account's permissions.

### Image Tags

The workflow generates multiple tags for each image:

| Tag Type | Example | When Created |
|----------|---------|--------------|
| Branch | `main`, `master` | On push to branch |
| PR | `pr-123` | On pull request |
| Semantic Version | `v1.0.0`, `v1.0` | On tag push |
| SHA | `main-abc1234` | On push to branch |
| Latest | `latest` | On push to default branch |

### Using the Images

#### Pull Images (Automatic Architecture Selection)

Docker automatically selects the correct architecture for your system:

```bash
# Pull latest images (Docker picks the right architecture)
docker pull ghcr.io/QuantumSolver/crm-relay/relay-server:latest
docker pull ghcr.io/QuantumSolver/crm-relay/relay-client:latest

# Pull specific version
docker pull ghcr.io/QuantumSolver/crm-relay/relay-server:v1.0.0
```

#### Pull Specific Architecture

If you need a specific architecture (e.g., for cross-platform deployment):

```bash
# Pull ARM64 image explicitly
docker pull --platform linux/arm64 ghcr.io/QuantumSolver/crm-relay/relay-server:latest

# Pull AMD64 image explicitly
docker pull --platform linux/amd64 ghcr.io/QuantumSolver/crm-relay/relay-server:latest
```

#### Verify Architecture

```bash
# Check which architecture was pulled
docker inspect ghcr.io/QuantumSolver/crm-relay/relay-server:latest | grep Architecture
```

#### Update docker-compose.yml

Replace the build sections with image references:

```yaml
services:
  relay-server:
    image: ghcr.io/QuantumSolver/crm-relay/relay-server:latest
    # Remove build section

  relay-client:
    image: ghcr.io/QuantumSolver/crm-relay/relay-client:latest
    # Remove build section
```

### Workflow Triggers

```yaml
on:
  push:
    branches: [main, master]
    tags: ['v*']
  pull_request:
    branches: [main, master]
```

### Permissions

The workflow requires these permissions:

```yaml
permissions:
  contents: read      # Read repository code
  packages: write     # Push images to GHCR
```

### Caching

The workflow uses GitHub Actions cache for Docker layers, speeding up subsequent builds:

```yaml
cache-from: type=gha
cache-to: type=gha,mode=max
```

### Multi-Architecture Build Details

The workflow uses Docker Buildx with QEMU for cross-platform builds:

1. **QEMU Setup**: Enables emulation for non-native architectures
2. **Buildx**: Docker's enhanced build system with multi-platform support
3. **Platform Matrix**: Builds for `linux/amd64`, `linux/arm64`, and `linux/arm/v7`
4. **Manifest Creation**: Creates a multi-arch manifest that Docker uses to select the right image

#### Build Time Considerations

Multi-architecture builds take longer than single-architecture builds:
- **Single arch**: ~2-3 minutes
- **Multi-arch**: ~5-8 minutes (builds all platforms in parallel)

The trade-off is worth it for:
- Wider platform support
- Automatic architecture selection
- Single image tag for all platforms

### Local Multi-Architecture Builds

You can build multi-architecture binaries locally using the Makefile:

```bash
# Build all architectures
make build-multiarch

# Build specific architecture
make build-server-amd64
make build-server-arm64
make build-server-armv7

# Build client for specific architecture
make build-client-arm64
```

This creates binaries like:
- `bin/relay-server-linux-amd64`
- `bin/relay-server-linux-arm64`
- `bin/relay-server-linux-armv7`

#### Local Docker Multi-Arch Build

To build multi-arch Docker images locally:

```bash
# Install buildx (if not already installed)
docker buildx version

# Create and use a multi-arch builder
docker buildx create --name multiarch --use
docker buildx inspect --bootstrap

# Build and push multi-arch images
docker buildx build --platform linux/amd64,linux/arm64,linux/arm/v7 \
  -t ghcr.io/QuantumSolver/crm-relay/relay-server:latest \
  -f Dockerfile.relay-server \
  --push .
```

### Pull Request Behavior

On pull requests, images are **built but not pushed**. This allows you to:
- Test the build process
- Verify image creation
- Catch errors before merging

### Viewing Workflow Runs

1. Go to your repository on GitHub
2. Click the "Actions" tab
3. Select a workflow run to view details
4. Check the "Summary" section for published image tags

### Troubleshooting

#### Workflow Fails with "Permission Denied"

Ensure your repository has packages enabled:
1. Go to repository Settings
2. Click "Actions" → "General"
3. Under "Workflow permissions", ensure "Read and write permissions" is selected

#### Images Not Appearing in GHCR

1. Check workflow logs for errors
2. Verify the workflow completed successfully
3. Visit `https://github.com/QuantumSolver?tab=packages` to see your packages

#### Authentication Issues

The workflow uses `GITHUB_TOKEN`, which should work automatically. If you see authentication errors:
1. Verify the workflow has `packages: write` permission
2. Check that you're pushing to the correct repository
3. Ensure your GitHub account has permission to write packages

#### Build Fails on Specific Architecture

If the build fails for a specific platform:
1. Check the workflow logs for the specific architecture
2. Verify Go supports the target architecture
3. Check for platform-specific dependencies in your code
4. Consider removing problematic platforms from the `PLATFORMS` env var

#### Slow Build Times

Multi-architecture builds are slower. To speed up:
1. Reduce the number of platforms (e.g., remove `linux/arm/v7` if not needed)
2. Use GitHub Actions cache (already enabled)
3. Optimize Dockerfile layers
4. Consider building only on tag releases instead of every push

#### QEMU Emulation Issues

If you see QEMU-related errors:
1. Ensure `docker/setup-qemu-action@v3` is running
2. Check that the platforms are supported by QEMU
3. Verify the workflow has sufficient resources

### Platform-Specific Considerations

#### ARM64 (Apple Silicon, ARM Servers)

- **Performance**: Native ARM64 builds are fast
- **Emulation**: AMD64 builds on ARM64 use QEMU (slower)
- **Use Cases**: Raspberry Pi 4/5, AWS Graviton, Apple Silicon Macs

#### ARMv7 (Raspberry Pi 3 and older)

- **Performance**: Slower than ARM64
- **Compatibility**: Good for older ARM devices
- **Use Cases**: Raspberry Pi 3, older ARM boards
- **Note**: Consider removing if you only support ARM64+

#### AMD64 (x86_64)

- **Performance**: Fastest on x86_64 systems
- **Compatibility**: Universal for most servers
- **Use Cases**: Cloud VMs, standard servers, most desktops

### Adding or Removing Platforms

To modify supported platforms, edit the workflow:

```yaml
env:
  PLATFORMS: linux/amd64,linux/arm64  # Removed linux/arm/v7
```

Or add more platforms:

```yaml
env:
  PLATFORMS: linux/amd64,linux/arm64,linux/arm/v7,linux/ppc64le
```

**Note**: Only add platforms that:
1. Are supported by Go
2. Are supported by Alpine Linux
3. Are supported by QEMU (for emulation)
4. You actually need for deployment

### Security Best Practices

✅ **Do's**:
- Use the built-in `GITHUB_TOKEN` (no need for personal tokens)
- Keep workflow permissions minimal
- Review workflow logs regularly
- Use semantic versioning for production releases

❌ **Don'ts**:
- Don't commit personal access tokens
- Don't grant unnecessary permissions
- Don't ignore workflow failures
- Don't push sensitive data in images

### Next Steps

1. **Enable Packages**: Ensure packages are enabled in your repository settings
2. **Test the Workflow**: Push a commit to trigger the workflow
3. **Verify Images**: Check that images appear in GHCR
4. **Update Documentation**: Update your README with image usage instructions

### Additional Workflows

You can add more workflows in this directory:

- **CI/CD**: Run tests on every push
- **Security Scanning**: Scan images for vulnerabilities
- **Release Automation**: Create GitHub releases on tags
- **Deployment**: Deploy to production environments

Example CI workflow:

```yaml
name: CI

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      - run: go test ./...
```

## Support

For issues with GitHub Actions workflows:
- Check the [GitHub Actions documentation](https://docs.github.com/en/actions)
- Review workflow logs in the Actions tab
- Open an issue in this repository
# Testing workflow trigger
