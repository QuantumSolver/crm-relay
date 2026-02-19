# GitHub Actions Workflows

This directory contains GitHub Actions workflows for the CRM Relay Server project.

## Docker Build and Push Workflow

**File**: `docker-build-push.yml`

### What It Does

This workflow automatically builds and pushes Docker images to GitHub Container Registry (GHCR) whenever you:
- Push to `main` or `master` branch
- Create a tag (e.g., `v1.0.0`)
- Open a pull request

### Images Built

- **Relay Server**: `ghcr.io/QuantumSolver/crm-relay/relay-server`
- **Relay Client**: `ghcr.io/QuantumSolver/crm-relay/relay-client`

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

#### Pull Images

```bash
# Pull latest images
docker pull ghcr.io/QuantumSolver/crm-relay/relay-server:latest
docker pull ghcr.io/QuantumSolver/crm-relay/relay-client:latest

# Pull specific version
docker pull ghcr.io/QuantumSolver/crm-relay/relay-server:v1.0.0
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
