# Spaceship

[![GoVersion](https://img.shields.io/badge/Go-v1.22.1-blue?logo=Go&style=flat-square)](https://go.dev/)


一个简单的基于 HTTP 协议进行文件传输的命令行程序，支持并发下载和上传，不支持多级目录，另外也提供了常规的并发下载等功能。

[English](./README.md) | 中文

## 安装

进入仓库的Releases界面下载已编译好的程序

## 使用

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
