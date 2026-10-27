# v2bx_rev

基于 [V2bX](https://github.com/InazumaV/V2bX) 的二次开发维护版本。

A V2board node server based on multi core, modified from XrayR.
一个基于多种内核的 V2board 节点服务端，支持 Vmess/Vless、Trojan、Shadowsocks 协议。

**注意：本项目需要搭配[修改版 V2board](https://github.com/wyx2685/v2board)使用。**

## 特点

* 永久开源且免费。
* 支持 Vmess/Vless, Trojan， Shadowsocks, Hysteria1/2 多种协议。
* 支持 Vless 和 XTLS 等新特性。
* 支持单实例对接多节点，无需重复启动。
* 支持限制在线 IP。
* 支持限制 Tcp 连接数。
* 支持节点端口级别、用户级别限速。
* 配置简单明了。
* 修改配置自动重启实例。
* 支持多种内核，易扩展。
* 支持条件编译，可仅编译需要的内核。

## 功能介绍

| 功能        | v2ray | trojan | shadowsocks | hysteria1/2 |
|-----------|-------|--------|-------------|----------|
| 自动申请tls证书 | √     | √      | √           | √        |
| 自动续签tls证书 | √     | √      | √           | √        |
| 在线人数统计    | √     | √      | √           | √        |
| 审计规则      | √     | √      | √           | √         |
| 自定义DNS    | √     | √      | √           | √        |
| 在线IP数限制   | √     | √      | √           | √        |
| 连接数限制     | √     | √      | √           | √         |
| 跨节点IP数限制  |√      |√       |√            |√          |
| 按照用户限速    | √     | √      | √           | √         |
| 动态限速(未测试) | √     | √      | √           | √         |

## 构建

``` bash
# 通过 -tags 选项指定要编译的内核，可选 xray， sing, hysteria2
GOEXPERIMENT=jsonv2 go build -v -o build_assets/V2bX -tags "sing xray hysteria2 with_quic with_grpc with_utls with_wireguard with_acme with_gvisor" -trimpath -ldflags "-X 'github.com/InazumaV/V2bX/cmd.version=$version' -s -w -buildid="
```

## 发布

创建 GitHub Release（或打 tag 后创建 Release）即自动触发 Actions，构建多平台二进制并上传到 Release，同时构建 Docker 镜像推送到 ghcr.io：

``` bash
git tag v0.1.1 && git push origin v0.1.1
gh release create v0.1.1 --generate-notes
```

也可在仓库 Actions 页面手动触发 `Build and Release` / `Publish Docker image` 工作流。

## 配置说明

配置文件示例见 [`example/config.json`](example/config.json)。

## 免责声明

* 此项目用于本人自用，因此本人不能保证向后兼容性。
* 由于本人能力有限，不能保证所有功能的可用性，如果出现问题请在 Issues 反馈。
* 本人不对任何人使用本项目造成的任何后果承担责任。
* 本项目可能会随想法或思路的变动随性更改项目结构或大规模重构代码，若不能接受请勿使用。

## Changelog

### v0.1.1 (2026-10-27)

* 修复 CI 权限：Release 工作流补充 `contents: write`，Docker 工作流补充 `packages: write`，解决上传资产/推送镜像时报 `Resource not accessible by integration`。
* 发布触发精简：去掉 tag 推送触发，统一由「创建 Release」或手动触发，避免重复构建。
* 实测：多平台二进制构建 23/23 通过，v0.1.1 为第一个带资产的正式版。

### v0.1.0 (2026-10-27)

* 首次发布：基于上游 V2bX 源码整理，清理上游社区标识（徽章、群链接、赞助、星图）。
* 移除空子模块声明（`.gitmodules`），依赖走 Go Modules `replace` 远程拉取，编译不受影响。
* CI 精简：构建/发版改为手动触发或 tag/release 触发，移除 push 宽触发与 CodeQL 模板。
* 新增本仓库发布流程：tag 推送后自动构建多平台二进制与 Docker 镜像。

## 许可证

[Mozilla Public License 2.0](LICENSE)