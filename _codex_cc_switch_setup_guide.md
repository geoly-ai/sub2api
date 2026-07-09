# Codex 接入第三方 API（基于 CC Switch）配置指南

更新时间：2026-06-29

本文目标：帮助你在 macOS 或 Windows 上安装 Codex、安装 CC Switch，并通过 CC Switch 让 Codex 使用第三方 API，同时尽量保留 Codex 官方登录态相关能力。

## 1. 先看结论

推荐顺序：

1. 安装 Codex
2. 安装 CC Switch
3. 先用 `OpenAI Official` 登录一次 Codex
4. 在 CC Switch 中开启 `Codex App Enhancements`
5. 添加第三方 Provider
6. 如果该 Provider 只支持 Chat Completions，开启 `Local Routing`
7. 重启 Codex 并验证模型请求是否已走第三方 API

如果你只是想最快完成配置，可以直接跳到“第 5 章：实操教程”。

## 2. 你最终会得到什么

- Codex 可以继续正常启动
- Codex 的模型请求可以切到第三方 API
- 对于 CC Switch 官方指南支持的场景，可以尽量保留官方登录态，不必把第三方 Key 直接覆盖到 Codex 的官方认证缓存

注意：

- 这不代表所有 Codex 官方能力都一定完全可用，是否可用仍取决于 OpenAI 当前产品策略、你的账号状态，以及第三方 Provider 的协议兼容程度
- 如果第三方 Provider 只支持 `/chat/completions`，通常必须开启 CC Switch 的本地路由转换

## 3. 官方下载入口

### 3.1 Codex 官方下载与文档

macOS：

- Codex App 下载入口：[https://chatgpt.com/codex/download/mac](https://chatgpt.com/codex/download/mac)
- Codex App 文档页：[https://developers.openai.com/codex/app](https://developers.openai.com/codex/app)

Windows：

- Codex App 下载入口：[https://chatgpt.com/codex/download/windows](https://chatgpt.com/codex/download/windows)
- Codex Windows 文档页：[https://developers.openai.com/codex/windows](https://developers.openai.com/codex/windows)

CLI / Quickstart：

- Codex Quickstart：[https://developers.openai.com/codex/quickstart](https://developers.openai.com/codex/quickstart)
- Codex Authentication：[https://developers.openai.com/codex/authentication](https://developers.openai.com/codex/authentication)

补充说明：

- 如果你准备同时使用 Codex App 和 Codex CLI，建议两者都安装
- 对于 CLI，OpenAI 官方 Quickstart 仍是最稳妥的参考入口

### 3.2 CC Switch 官方下载与文档

CC Switch Releases：

- GitHub Releases：[https://github.com/farion1231/cc-switch/releases](https://github.com/farion1231/cc-switch/releases)

截至 2026-06-29，GitHub 最新 release 为 `v3.16.4`。

推荐下载：

macOS：

- [CC-Switch-v3.16.4-macOS.dmg](https://github.com/farion1231/cc-switch/releases/download/v3.16.4/CC-Switch-v3.16.4-macOS.dmg)
- 备选：[CC-Switch-v3.16.4-macOS.zip](https://github.com/farion1231/cc-switch/releases/download/v3.16.4/CC-Switch-v3.16.4-macOS.zip)

Windows：

- [CC-Switch-v3.16.4-Windows.msi](https://github.com/farion1231/cc-switch/releases/download/v3.16.4/CC-Switch-v3.16.4-Windows.msi)
- 便携版：[CC-Switch-v3.16.4-Windows-Portable.zip](https://github.com/farion1231/cc-switch/releases/download/v3.16.4/CC-Switch-v3.16.4-Windows-Portable.zip)
- ARM 设备：[CC-Switch-v3.16.4-Windows-arm64.msi](https://github.com/farion1231/cc-switch/releases/download/v3.16.4/CC-Switch-v3.16.4-Windows-arm64.msi)

安装文档：

- CC Switch Installation Guide：
  [https://github.com/farion1231/cc-switch/blob/main/docs/user-manual/en/1-getting-started/1.2-installation.md](https://github.com/farion1231/cc-switch/blob/main/docs/user-manual/en/1-getting-started/1.2-installation.md)

## 4. 前置准备

开始前建议准备好以下内容：

- 一台 macOS 或 Windows 机器
- 已安装 Codex App，最好同时装好 Codex CLI
- 已安装 CC Switch
- 一个可用的第三方 API Key
- 一个可用于登录 Codex 的 OpenAI / ChatGPT 账号

如果你希望通过 CC Switch 托管 CLI 工具，CC Switch 官方安装文档还建议准备：

- Node.js 18 LTS 或更高版本

Node.js 官方网站：

- [https://nodejs.org/](https://nodejs.org/)

## 5. 实操教程

### 第 1 步：安装 Codex

1. 按你的系统打开官方下载入口
2. macOS 下载 `Codex App`
3. Windows 下载 `Codex for Windows`
4. 完成安装后先确认应用能够正常启动

如果你还需要 CLI：

```bash
brew install codex
```

或：

```bash
npm install -g @openai/codex
```

### 第 2 步：安装 CC Switch

1. 打开 Releases 页面
2. 按系统下载对应安装包
3. macOS 建议直接使用 `.dmg`
4. Windows 建议优先使用 `.msi`
5. 安装后启动 CC Switch

### 第 3 步：先完成一次官方登录

这一点很重要。

根据 CC Switch 的官方指南，推荐先在 CC Switch 的 `Codex` 面板中切回：

```text
OpenAI Official
```

然后启动 Codex，使用你的官方 ChatGPT / Codex 账号完成一次登录。

这样做的目的，是先让 Codex 生成并保留官方登录缓存。CC Switch 的说明中提到，后续增强模式会尽量把：

- 官方登录态保留在 `~/.codex/auth.json`
- 第三方 Provider 配置写入 `~/.codex/config.toml`

这样可以避免“切第三方时把官方登录缓存覆盖掉”。

### 第 4 步：开启 CC Switch 的 Codex 增强开关

在 CC Switch 中打开：

```text
Settings -> General -> Codex App Enhancements
```

启用：

```text
Keep official login when switching third-party providers
```

这一步是本文最关键的设置之一。

它的意义是：

- 保留官方登录态
- 切换第三方 API 时，尽量通过 `config.toml` 驱动 Codex 使用第三方 Provider
- 避免旧行为直接改写官方认证缓存

### 第 5 步：添加第三方 Provider

回到 CC Switch 的 `Codex` 面板，点击右上角的加号添加 Provider。

优先建议使用 CC Switch 已内置的 Preset，例如：

- DeepSeek
- Kimi
- GLM
- MiniMax
- SiliconFlow

如果使用预设，通常只需要：

1. 选择 Provider
2. 填入 API Key
3. 保存

预设的优势是通常已经带好：

- Base URL
- 默认模型
- 模型映射
- 是否需要本地路由

### 第 6 步：按需开启 Local Routing

这一步不是所有 Provider 都必须做，但对于只支持 OpenAI Chat Completions 协议的 Provider，通常必须开启。

进入：

```text
Settings -> Routing -> Local Routing
```

完成两件事：

1. 打开本地路由总开关
2. 在 `Routing Enabled` 里打开 `Codex`

CC Switch 官方文档解释的核心原因是：

- 新版 Codex 更偏向 OpenAI Responses API
- 很多第三方厂商提供的是 `/chat/completions`
- 这两种协议在请求体、流式事件和返回结构上并不完全一致

所以 CC Switch 会在本地做一层转换：

```text
Codex -> CC Switch Local Route -> 第三方 Chat API -> 转回 Codex 可识别的响应
```

如果你把只支持 Chat Completions 的第三方地址直接手写进 Codex，常见问题会包括：

- 404
- 400
- 模型列表不对
- 流式输出异常

### 第 7 步：切换到第三方 Provider

回到 CC Switch 的 `Codex` Provider 列表，把刚创建的第三方 Provider 设为当前启用项。

然后重启 Codex。

重启原因通常有两个：

- Codex 会在启动时读取 `config.toml`
- 模型目录或模型菜单可能不会热更新

### 第 8 步：验证是否成功

可以按下面顺序验证：

1. Codex 能正常启动
2. CC Switch 中当前 Provider 已经是目标第三方
3. 如果启用了 Local Routing，路由面板里能看到 Codex 请求
4. 第三方 Provider 后台能看到实际请求或额度变化

注意：

- 即使 Codex 里显示的仍然是官方账号，也不代表流量还在走 OpenAI
- 是否真的走第三方，应以 CC Switch 当前 Provider、路由日志、第三方后台请求记录为准

## 6. macOS 和 Windows 的建议做法

### macOS

推荐顺序：

1. 安装 Codex App
2. 安装 Codex CLI
3. 安装 Node.js
4. 安装 CC Switch
5. 先做一次官方登录
6. 再接入第三方 Provider

原因：

- macOS 下 CLI 和桌面 App 配合通常更顺手
- Homebrew 安装维护更方便

可参考命令：

```bash
brew install node
brew install codex
brew install --cask cc-switch
```

### Windows

推荐顺序：

1. 安装 Codex for Windows
2. 安装 Node.js LTS
3. 安装 CC Switch 的 `.msi`
4. 启动 Codex 完成官方登录
5. 打开 CC Switch 添加第三方 Provider
6. 需要时启用 Local Routing

## 7. 常见问题

### 1. 为什么我切到第三方后，Codex 里还是显示官方账号？

这通常是预期行为。

CC Switch 官方说明里提到，开启 `Keep official login when switching third-party providers` 后，官方登录态仍保存在 `auth.json`，而真正的 Provider 信息和模型配置写在 `config.toml`。所以“显示官方账号”和“实际请求走第三方 API”可以同时成立。

### 2. 哪些场景必须开启 Local Routing？

一般来说，如果第三方只提供 OpenAI Chat Completions 兼容接口，就建议开启。

常见需要路由转换的类型：

- DeepSeek 风格接口
- Kimi
- MiniMax
- 部分聚合平台

如果上游原生支持 OpenAI Responses API，则不一定需要。

### 3. 如果报 404 或模型列表不对怎么办？

优先检查这几项：

1. 当前启用的是否真的是第三方 Provider
2. `Needs Local Routing` 是否应当开启
3. `Settings -> Routing -> Local Routing` 是否已启动
4. `Routing Enabled` 中的 `Codex` 是否已打开
5. 切换后是否重启过 Codex

### 4. 需要把第三方 Key 手动写进 Codex 配置文件吗？

不建议手工改，优先让 CC Switch 管理。

原因：

- 更容易出错
- 容易把官方登录缓存覆盖掉
- 后续切换模型或 Provider 也不方便

## 8. 推荐的最稳妥配置方案

如果你想要一套最不容易踩坑的方案，建议直接照这个顺序做：

1. 安装 Codex App
2. 安装 Codex CLI
3. 安装 Node.js
4. 安装 CC Switch
5. 在 CC Switch 中切回 `OpenAI Official`
6. 启动 Codex 并完成一次官方登录
7. 开启 `Keep official login when switching third-party providers`
8. 添加第三方 Provider
9. 如果 Provider 只支持 Chat Completions，开启 `Local Routing`
10. 切换 Provider 后重启 Codex
11. 用 CC Switch 路由日志和第三方后台请求记录做最终验证

## 9. 参考资料

OpenAI 官方：

- Codex App：[https://developers.openai.com/codex/app](https://developers.openai.com/codex/app)
- Codex Windows：[https://developers.openai.com/codex/windows](https://developers.openai.com/codex/windows)
- Codex Quickstart：[https://developers.openai.com/codex/quickstart](https://developers.openai.com/codex/quickstart)
- Codex Authentication：[https://developers.openai.com/codex/authentication](https://developers.openai.com/codex/authentication)

CC Switch 官方：

- Releases：[https://github.com/farion1231/cc-switch/releases](https://github.com/farion1231/cc-switch/releases)
- Installation Guide：[https://github.com/farion1231/cc-switch/blob/main/docs/user-manual/en/1-getting-started/1.2-installation.md](https://github.com/farion1231/cc-switch/blob/main/docs/user-manual/en/1-getting-started/1.2-installation.md)
- Codex 官方登录保留指南：
  [https://github.com/farion1231/cc-switch/blob/main/docs/guides/codex-official-auth-preservation-guide-en.md](https://github.com/farion1231/cc-switch/blob/main/docs/guides/codex-official-auth-preservation-guide-en.md)
- Codex 第三方 Chat API 路由指南：
  [https://github.com/farion1231/cc-switch/blob/main/docs/guides/codex-deepseek-routing-guide-en.md](https://github.com/farion1231/cc-switch/blob/main/docs/guides/codex-deepseek-routing-guide-en.md)
