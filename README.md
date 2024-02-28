# Spaceship

[![GoVersion](https://img.shields.io/badge/Go-v1.20.2-blue?logo=Go&style=flat-square)](https://go.dev/)
[![Go Reference](https://pkg.go.dev/badge/github.com/tianluanchen/spaceship.svg)](https://pkg.go.dev/github.com/tianluanchen/spaceship)

A simple command-line program for managing remote files, using the HTTP protocol for transmission. It supports concurrent downloading and uploading, does not support multi-level directories, and also provides commands for concurrent downloading of regular HTTP URLs.

English | [中文](./README.zh-CN.md)

## Installation

```bash
go install github.com/tianluanchen/spaceship@latest
```

You can also go to the Actions page of the repository to download the pre-compiled binary from the Artifact.

## Usage

```bash
# Start the server, specify the current path as the root path for management
spaceship serve --root ./
# List files in the remote directory
spaceship ls
# Store the remote file 'a' in the current directory
spaceship get ./a
# Store the local file 'a' to the remote directory, this will forcibly overwrite existing remote files and terminate tasks with the same target path
spaceship put ./a --overwrite
# Concurrently download network resources to local
spaceship fetch https://example.com/largefile output-file
# Get more help
spaceship --help
```

## License

[GPL-3.0](./LICENSE) © Ayouth