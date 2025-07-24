# Image Mirroring with IDMS and ITMS

This document describes the image mirroring functionality implemented in ROSA CLI using the Image Distribution Management System (IDMS) and Image Transfer Management System (ITMS).

## Overview

The image mirroring functionality provides comprehensive support for:

- **IDMS (Image Distribution Management System)**: Lists and validates images in container registries
- **ITMS (Image Transfer Management System)**: Synchronizes images between registries
- **Authentication**: Supports username/password, token-based, and credential file authentication
- **Error Handling**: Comprehensive error handling with retry logic and detailed error reporting
- **Logging**: Detailed logging for troubleshooting and audit purposes
- **Concurrency**: Configurable concurrent operations for efficient large-scale mirroring

## CLI Commands

### Create Image Mirror Operations

```bash
# Mirror images from one registry to another
rosa create image-mirror \
  --source-registry source.io \
  --target-registry target.io \
  --images image1:tag1,image2:tag2

# Mirror with authentication
rosa create image-mirror \
  --source-registry source.io \
  --target-registry target.io \
  --username user \
  --password pass \
  --images image1:latest

# Dry run mode
rosa create image-mirror \
  --source-registry source.io \
  --target-registry target.io \
  --images image1:latest \
  --dry-run
```

### List Images in Registry

```bash
# List images in a registry
rosa list image-mirror --registry registry.io

# List images with JSON output
rosa list image-mirror \
  --registry registry.io \
  --output json
```

### Describe Images and Operations

```bash
# Describe an image
rosa describe image-mirror --image registry.io/repo:tag

# Describe a sync operation
rosa describe image-mirror --operation-id abc123
```

## Features

### IDMS (Image Distribution Management System)

- List all images in a container registry
- Get detailed information about specific images
- Validate access permissions to images
- Support for multiple authentication methods

### ITMS (Image Transfer Management System)

- Synchronize single images between registries
- Batch synchronization of multiple images
- Track operation status and progress
- Configurable concurrency for performance optimization

## Architecture

The implementation includes:

1. **Types**: Core data structures and interfaces (`pkg/imagemirror/types.go`)
2. **IDMS Service**: Image distribution management (`pkg/imagemirror/idms.go`)
3. **ITMS Service**: Image transfer management (`pkg/imagemirror/itms.go`)
4. **Main Service**: Combined service implementation (`pkg/imagemirror/service.go`)
5. **Error Handling**: Comprehensive error management (`pkg/imagemirror/errors.go`)
6. **CLI Commands**: User interface components (`cmd/*/imagemirror/`)

## Testing

Comprehensive test coverage with 60 test cases:

```bash
# Run all tests
go test ./pkg/imagemirror/...
```