<p align="center">
  <img src="assets/Icon.png" alt="Flow Editor" width="128">
</p>

<h1 align="center">Flow Editor</h1>

<p align="center">
  <strong>简体中文</strong> |
  <a href="README.md">English</a>
</p>

<p align="center">
  <a href="https://github.com/oneo-me"><img alt="GitHub" src="https://img.shields.io/badge/GitHub-oneo--me-181717?logo=github&logoColor=white"></a>
  <a href="https://oneo.me"><img alt="博客" src="https://img.shields.io/badge/%E5%8D%9A%E5%AE%A2-oneo.me-0969da"></a>
  <a href="https://x.com/oneo666"><img alt="X" src="https://img.shields.io/badge/X-%40oneo666-000000?logo=x&logoColor=white"></a>
  <a href="https://github.com/sponsors/oneo-me"><img alt="GitHub 赞助" src="https://img.shields.io/badge/GitHub%20%E8%B5%9E%E5%8A%A9-ea4aaa?logo=githubsponsors&logoColor=white"></a>
</p>

<p align="center">
  一款桌面流程图编辑器：和 AI 助手对话来设计流程图，助手按商定的内容把它画出来。
</p>

<p align="center">
  <img src="assets/Preview.png" alt="Flow Editor">
</p>

## 功能

- 同一窗口内，左侧是流程图，右侧是对话。
- 助手先问清需要的信息，再用工具添加、连接和重排节点。
- 助手的每次修改可作为一步撤销。
- 支持任何兼容 OpenAI 且支持工具调用的服务商，包括本地服务。
- 流程图保存为纯文本 `.flow` 文件，可直接阅读、对比和手动编辑。
- 运行于 macOS、Windows 和 Linux，支持英文和简体中文。

## 运行

Flow Editor 需要 [Go](https://go.dev/dl/)。在仓库根目录执行：

```sh
go tool nsgo dev            # run
go tool nsgo build          # build macOS, Windows and Linux into dist/
```

命令行中给出的流程文件会在启动时打开。

## 快速开始

1. 打开**设置**（⌘,），填写服务商的接口地址（如 `https://api.openai.com/v1`）、API 密钥和支持工具调用的模型。
2. 在助手中描述一个流程，例如“画出我们处理订单的流程”。
3. 回答它的问题，看它在左侧画出流程图。
4. 将流程图保存为 `.flow` 文件，或打开 `examples/order.flow` 从示例开始。

文件格式见 [`flowfile` 包注释](flowfile/)。

## 使用统计

正式构建会向 Umami 发送匿名使用统计：应用的使用次数和时长，附带一个随机的安装 ID。不会发送流程图、消息、文件路径或 API 密钥。可在**设置** → **统计**中关闭。`nsgo dev` 的构建和测试不发送任何统计。

## 开发

在仓库根目录执行：

```sh
go tool nsgo check          # vet, catalogs and generated files
go tool nsgo test           # tests, headless against a fake model
```

设计见 [ARCHITECTURE.md](ARCHITECTURE.md)，贡献规则见 [AGENTS.md](AGENTS.md)。

变更记录见 [CHANGELOG.md](CHANGELOG.md)。

## 许可证

Flow Editor 以 [MIT 许可证](LICENSE) 发布。贡献内容按同一许可证接受。
