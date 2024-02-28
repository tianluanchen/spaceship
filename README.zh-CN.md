# Spaceship

[![GoVersion](https://img.shields.io/badge/Go-v1.20.2-blue?logo=Go&style=flat-square)](https://go.dev/)
[![Go Reference](https://pkg.go.dev/badge/github.com/tianluanchen/spaceship.svg)](https://pkg.go.dev/github.com/tianluanchen/spaceship)

一个简单的对远程文件进行管理的命令行程序，使用 HTTP 协议进行传输，支持并发下载和上传，不支持多级目录，另外也提供对常规 HTTP URL 进行并发下载的命令。

[English](./README.md) | 中文

## 安装

```bash
go install github.com/tianluanchen/spaceship@latest
```

你也可以进入仓库的Actions界面，从Artifact中获取已编译好的程序

## 使用

```bash
# 启用服务端，指定当前路径为管理的根路径
spaceship serve --root ./
# 查看远程目录的文件
spaceship ls
# 将远程文件a存储到当前目录
spaceship get ./a
# 将本地文件a存储到远程目录，这会强制覆盖已有的远程文件和终止其它目标路径相同的任务
spaceship put ./a  --overwrite
# 并发下载网络资源到本地
spaceship fetch https://example.com/largefile output-file
# 获取更多帮助
spaceship --help
```

## License

[GPL-3.0](./LICENSE) © Ayouth
