# Spaceship

[![GoVersion](https://img.shields.io/badge/Go-v1.22.1-blue?logo=Go&style=flat-square)](https://go.dev/)

A simple command-line program for file transfer based on the HTTP protocol. It supports concurrent downloading and uploading but does not support multi-level directories. Additionally, it provides regular concurrent download functions.

English | [中文](./README.zh-CN.md)

## Installation

Download the precompiled program from the Releases page of the repository.

## Usage

```bash
$ spaceship --help
Concurrent HTTP downloader, uploader client and server

Usage:
  spaceship [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  conf        Read and set config
  fetch       Concurrent download of web content to local
  gencert     Generate tls certificate
  get         Concurrent download remote file to local
  help        Help about any command
  install     install to GOPATH BIN
  ls          List remote files
  mv          Move remote file to specified path
  ping        Ping server
  put         Concurrent upload local file to remote
  rm          Remove a remote file
  serve       Start the server
  unzip       Unarchive zip
  version     Print the version of spaceship
  zip         Archive files with zip

Flags:
  -h, --help           help for spaceship
      --level string   log level, DEBUG INFO WARN ERROR FATAL (default "INFO")

Use "spaceship [command] --help" for more information about a command.
```

## License

[GPL-3.0](./LICENSE) © Ayouth
