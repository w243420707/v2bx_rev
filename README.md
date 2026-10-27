# v2bx_rev

## 一键安装

适用 Debian / Ubuntu / CentOS / Alpine / Arch 等主流 Linux（需 root）。脚本自动识别系统与架构（x86_64 / aarch64 / s390x），从 Releases 下载对应二进制，安装为 systemd 或 OpenRC 服务：

``` bash
bash <(curl -Ls https://raw.githubusercontent.com/w243420707/v2bx_rev/main/scripts/install.sh)
```

安装后可用 `V2bX` 管理命令（安装于 `/usr/bin/V2bX`，兼容 `v2bx` 小写）：

| 命令 | 作用 |
|------|------|
| `V2bX` | 显示管理菜单 |
| `V2bX start` / `stop` / `restart` | 启动 / 停止 / 重启 |
| `V2bX status` | 查看运行状态 |
| `V2bX log` | 查看日志 |
| `V2bX update` / `update x.x.x` | 更新到最新 / 指定版本 |
| `V2bX install` / `uninstall` | 安装 / 卸载 |
| `V2bX x25519` | 生成 x25519 密钥 |
| `V2bX generate` | 生成配置文件 |

首次安装时脚本会询问是否自动生成配置文件，也可随时用 `V2bX generate` 重新生成。配置文件位于 `/etc/V2bX/config.json`。

## Changelog

### v0.1.7 (2026-10-27)

* 整理 README：清除其余全部内容，只保留一键安装与 Changelog 两部分。

### v0.1.6 (2026-10-27)

* 清理仓库分支：关闭并删除 5 个 Dependabot 自动升级 PR 及其分支，仓库只保留 `main` 主分支。
* 彻底移除 `.github/dependabot.yml`（Dependabot 自动依赖更新是这些分支的来源），不再自动创建升级 PR/分支，依赖升级改为手动。

### v0.1.5 (2026-10-27)

* 依赖保险：编译命脉的三个上游个人仓库已 fork 到本账号并接入——`w243420707/xray-core`（替换 xray 内核）、`w243420707/sing-box_mod`（替换 sing-box 内核）、`w243420707/sing-vmess`（vmess 扩展，含 module 路径修正）。`go.mod` 两条 `replace` 与兜底 replace 全部指向 fork，`go.sum` 不再引用 wyx2685。
* 意义：即使上游 `wyx2685` 删库/停更，本仓库依赖链仍完全独立可编译。已在本地验证干净缓存从零下载零错误 + linux/amd64 全内核完整编译通过；CI 全平台构建作最终确认。

### v0.1.4 (2026-10-27)

* 一键安装改用上游原版脚本（`scripts/install.sh` + `V2bX.sh` + `initconfig.sh`），仅替换仓库地址为 `w243420707/v2bx_rev`，其余逻辑保持原版：安装路径 `/usr/local/V2bX/`、systemd/OpenRC 双支持、x86_64/aarch64/s390x 架构识别、安装后附带 `V2bX` 管理命令与首次配置生成向导。
* ARM 架构支持与原版完全一致：CI 构建矩阵（linux arm5/6/7、arm64、darwin arm64、android arm64、freebsd arm、windows arm7）未做任何改动，安装脚本从 Releases 拉取对应架构资产。

### v0.1.3 (2026-10-27)

* 恢复被过度清理的原有功能：还原 CodeQL 安全扫描工作流、Issue 模板，恢复 push/PR 自动构建触发（`master` / `dev_new` / `main` 分支），README 恢复 TODO、手动安装教程、使用教程链接与 Thanks 致谢。
* 保留必要修复：Release/Docker 工作流的 `permissions` 权限声明（缺失会导致资产上传与镜像推送失败）。

### v0.1.2 (2026-10-27)

* 新增 `scripts/install.sh` 一键安装/更新/卸载脚本：自动识别架构下载对应 Release 资产，安装为 systemd 服务，支持交互菜单与 `install/update/uninstall` 非交互参数。

### v0.1.1 (2026-10-27)

* 修复 CI 权限：Release 工作流补充 `contents: write`，Docker 工作流补充 `packages: write`，解决上传资产/推送镜像时报 `Resource not accessible by integration`。
* 发布触发精简：去掉 tag 推送触发，统一由「创建 Release」或手动触发，避免重复构建。
* 实测：多平台二进制构建 23/23 通过，v0.1.1 为第一个带资产的正式版。

### v0.1.0 (2026-10-27)

* 首次发布：基于上游 V2bX 源码整理，清理上游社区标识（徽章、群链接、赞助、星图）。
* 移除空子模块声明（`.gitmodules`），依赖走 Go Modules `replace` 远程拉取，编译不受影响。
* CI 精简：构建/发版改为手动触发或 tag/release 触发，移除 push 宽触发与 CodeQL 模板。
* 新增本仓库发布流程：tag 推送后自动构建多平台二进制与 Docker 镜像。
