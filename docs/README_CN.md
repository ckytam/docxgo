# 文档索引

**最后更新**：2026 年 7 月
**版本**：2.11.0（稳定版）

欢迎使用 docxgo v2 文档！本索引帮助您根据角色与主题快速找到需要的文档。

---

## 🎯 按角色导航

### 📖 想立即使用库的用户

1. **[README.md](../README.md)** —— 从这里开始！快速入门、安装、基础示例
2. **[V2_API_GUIDE.md](./V2_API_GUIDE.md)** —— 完整 API 参考与代码示例
3. **[examples/](../examples/)** —— 13 个可运行的示例程序
4. **[MIGRATION.md](../MIGRATION.md)** —— 从 v1 迁移到 v2

### 🛠️ 想贡献代码的开发者

1. **[CONTRIBUTING.md](../CONTRIBUTING.md)** —— 贡献指南与开发工作流
2. **[V2_DESIGN.md](./V2_DESIGN.md)** —— 架构与实现阶段（权威架构参考）
3. **[IMPLEMENTATION_STATUS.md](./IMPLEMENTATION_STATUS.md)** —— 功能状态与路线图
4. **[ERROR_HANDLING.md](./ERROR_HANDLING.md)** —— 错误处理模式
5. **[CODEBUDDY.md](../CODEBUDDY.md)** —— AI 编程代理（CodeBuddy）操作指引

### 🔍 维护者 / 审核者

1. **[IMPLEMENTATION_STATUS.md](./IMPLEMENTATION_STATUS.md)** —— 发布历史与功能跟踪
2. **[PROVENANCE_AUDIT.md](./PROVENANCE_AUDIT.md)** —— 溯源与许可证审计
3. **[TROUBLESHOOTING_DOCX_VALIDATION.md](./TROUBLESHOOTING_DOCX_VALIDATION.md)** —— OOXML 校验问题排查
4. **[CLI_GUIDE.md](./CLI_GUIDE.md)** —— JSON-RPC CLI 用法
5. **[PARAGRAPH_BORDERS.md](./PARAGRAPH_BORDERS.md)** —— 段落边框 API

---

## 📚 按主题导航

### 核心概念

| 主题 | 文档 | 说明 |
|------|------|------|
| 架构 | [V2_DESIGN.md](./V2_DESIGN.md) | 清晰架构、分层、设计模式 |
| API 用法 | [V2_API_GUIDE.md](./V2_API_GUIDE.md) | 所有公共 API 与示例 |
| 构建器 | [V2_API_GUIDE.md](./V2_API_GUIDE.md#builder-pattern) | 流式 API 与错误累积 |
| 直接 API | [V2_API_GUIDE.md](./V2_API_GUIDE.md#direct-domain-api) | 接口驱动设计 |

### 功能文档

| 功能 | 文档 | 说明 |
|------|------|------|
| 段落与文本 | [V2_API_GUIDE.md](./V2_API_GUIDE.md#paragraphs-and-text) | 格式化、对齐、缩进 |
| 表格 | [V2_API_GUIDE.md](./V2_API_GUIDE.md#tables) | 合并、嵌套、样式 |
| 图片 | [V2_API_GUIDE.md](./V2_API_GUIDE.md#images) | 内联/浮动图片、尺寸 |
| 域 | [V2_API_GUIDE.md](./V2_API_GUIDE.md#fields) | 页码、目录、超链接 |
| 节与页面 | [V2_API_GUIDE.md](./V2_API_GUIDE.md#sections-and-page-layout) | 页眉/页脚、分栏 |
| 样式 | [V2_API_GUIDE.md](./V2_API_GUIDE.md#styles) | 40+ 内置样式 |
| 主题 | [examples/13_themes/](../examples/13_themes/) | 7 套预设主题 |
| 段落边框 | [PARAGRAPH_BORDERS.md](./PARAGRAPH_BORDERS.md) | 段落边框 API |
| 模板/邮件合并 | [V2_API_GUIDE.md](./V2_API_GUIDE.md#template--mail-merge) | `{{placeholder}}` 替换 |
| CLI | [CLI_GUIDE.md](./CLI_GUIDE.md) | JSON-RPC 命令行 |

### 质量与运维

| 主题 | 文档 | 说明 |
|------|------|------|
| 实现状态 | [IMPLEMENTATION_STATUS.md](./IMPLEMENTATION_STATUS.md) | 功能完成度 |
| 错误处理 | [ERROR_HANDLING.md](./ERROR_HANDLING.md) | 错误模式与最佳实践 |
| 校验排查 | [TROUBLESHOOTING_DOCX_VALIDATION.md](./TROUBLESHOOTING_DOCX_VALIDATION.md) | OOXML 校验 |
| 溯源审计 | [PROVENANCE_AUDIT.md](./PROVENANCE_AUDIT.md) | 许可证判定 |
| 迁移 | [MIGRATION.md](../MIGRATION.md) | v1 → v2 |

---

## 🗺️ 建议阅读顺序

### 新用户（约 30 分钟）

```
1. README.md                    ← 快速入门
2. V2_API_GUIDE.md              ← 通读 "Quick Start" + "Core Features"
3. examples/01_basic/           ← 运行第一个示例
```

### 贡献者（约 2 小时）

```
1. README.md                    ← 概览
2. V2_DESIGN.md                 ← 理解架构
3. CONTRIBUTING.md              ← 工作流
4. ERROR_HANDLING.md            ← 错误处理规范
5. IMPLEMENTATION_STATUS.md     ← 找到可认领的功能
```

### 维护者（约 1 小时）

```
1. IMPLEMENTATION_STATUS.md     ← 当前状态
2. PROVENANCE_AUDIT.md          ← 许可证对齐
3. TROUBLESHOOTING_DOCX_VALIDATION.md ← 校验问题
4. CLI_GUIDE.md                 ← CLI 能力
```

---

## 🔍 快速参考

### 常用命令

```bash
# 运行所有测试
go test ./...

# 运行单个包的测试
go test ./pkg/template/...

# 构建 CLI
make build

# 运行示例
make examples

# 校验生成的 docx
dotnet run --project DocxValidator -- output.docx
```

### 模块信息

- **模块路径**：`github.com/mmonterroca/docxgo/v2`
- **最低 Go 版本**：1.23
- **许可证**：MIT
- **当前版本**：2.11.0

### 关键类型

| 类型 | 包 | 说明 |
|------|-----|------|
| `Document` | `domain` | 文档接口 |
| `Paragraph` | `domain` | 段落接口 |
| `Run` | `domain` | 文本 run 接口 |
| `Table` | `domain` | 表格接口 |
| `DocumentBuilder` | `docx` | 流式构建器 |

---

## 📋 文档清单

| 文档 | 行数（约） | 用途 |
|------|-----------|------|
| [README.md](../README.md) | 400+ | 用户快速入门 |
| [V2_API_GUIDE.md](./V2_API_GUIDE.md) | 850+ | API 参考 |
| [V2_DESIGN.md](./V2_DESIGN.md) | 900+ | 架构与设计 |
| [IMPLEMENTATION_STATUS.md](./IMPLEMENTATION_STATUS.md) | 450+ | 功能跟踪 |
| [ERROR_HANDLING.md](./ERROR_HANDLING.md) | 900+ | 错误模式 |
| [CLI_GUIDE.md](./CLI_GUIDE.md) | 1000+ | CLI 用法 |
| [PARAGRAPH_BORDERS.md](./PARAGRAPH_BORDERS.md) | 200+ | 边框 API |
| [PROVENANCE_AUDIT.md](./PROVENANCE_AUDIT.md) | 150+ | 许可证审计 |
| [TROUBLESHOOTING_DOCX_VALIDATION.md](./TROUBLESHOOTING_DOCX_VALIDATION.md) | 350+ | 校验排查 |
| [MIGRATION.md](../MIGRATION.md) | 200+ | v1 → v2 迁移 |

---

## 🆘 需要帮助？

- **问题**：在 GitHub 上开 Issue
- **讨论**：使用 GitHub Discussions
- **文档缺失**：欢迎提交 PR 改进文档

---

**最后更新**：2026 年 7 月
**版本**：2.11.0（稳定版）
