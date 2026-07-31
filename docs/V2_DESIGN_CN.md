# docxgo v2 设计文档

**版本**：2.11.0 | **最后更新**：2026 年 7 月 | **维护者**：Misael Monterroca

---

## 📋 目录

- [🎯 概述](#-概述)
- [🏗️ 架构](#-架构)
- [📦 模块结构](#-模块结构)
- [🎨 设计原则](#-设计原则)
- [📊 实现阶段](#-实现阶段)
- [🔄 文档流程](#-文档流程)
- [📅 历史笔记](#-历史笔记)
- [🤝 贡献](#-贡献)
- [📚 参考](#-参考)

---

## 🎯 概述

docxgo v2 是 Go 语言中用于创建、读取与修改 Microsoft Word (.docx) 文档的库。它是基于上游 `fumiama/go-docx` 的大幅重写，目标是：

- ✅ **清晰架构** —— 层次分明的接口/实现分离
- ✅ **类型安全** —— 编译期保证
- ✅ **现代 Go 实践** —— 惯用 API、错误累积、函数式选项
- ✅ **可扩展** —— 易于添加新功能
- ✅ **可维护** —— 模块化、可测试

**本设计文档是 docxgo v2 架构与实现阶段的权威参考。** 它包含了当前 MIT 发布产物为作者独立重写的审计结论（见 [V2 重写与 MIT 发布](#v2-重写与-mit-发布) 与 [PROVENANCE_AUDIT.md](./PROVENANCE_AUDIT.md)）。

---

## 🏗️ 架构

docxgo v2 采用**清晰架构（Clean Architecture）**模式，具有明确的依赖方向与职责分离。

### 分层设计

```
┌─────────────────────────────────────────────────────┐
│                 公共 API 层 (docx)                    │
│         NewDocument(), OpenDocument(), Builder        │
└───────────────────────┬─────────────────────────────┘
                        │ 使用
                        ▼
┌─────────────────────────────────────────────────────┐
│               Domain 层 (domain)                     │
│         Document, Paragraph, Run, Table, Image       │
│           纯接口 —— 无实现细节                        │
└───────────────────────┬─────────────────────────────┘
                        │ 由...实现
                        ▼
┌─────────────────────────────────────────────────────┐
│          Internal 实现层 (internal)                  │
│  ┌────────────┐ ┌────────────┐ ┌──────────────────┐  │
│  │   core/    │ │   xml/     │ │   reader/        │  │
│  │ (业务逻辑) │ │ (OOXML)    │ │   (解析)         │  │
│  └────────────┘ └────────────┘ └──────────────────┘  │
│  ┌────────────┐ ┌────────────┐ ┌──────────────────┐  │
│  │ serializer/│ │  writer/   │ │   manager/       │  │
│  │ (生成 XML) │ │ (ZIP)      │ │   (ID/关系/媒体) │  │
│  └────────────┘ └────────────┘ └──────────────────┘  │
└─────────────────────────────────────────────────────┘
```

### 依赖规则

1. **domain** 层不依赖任何其他层（最稳定、最抽象）
2. **internal** 层实现 domain 接口
3. **docx** 包是面向用户的门面
4. 依赖方向始终向内（向内 = 朝向 domain）

### 关键设计决策

| 决策 | 理由 |
|------|------|
| 接口在 `domain/` 中 | 稳定的契约，不随实现变动 |
| 实现在 `internal/` 中 | 实现细节可自由演进 |
| 管理器（ID/关系/媒体）分离 | 横切关注点集中管理 |
| 函数式选项模式 | 可扩展配置而不破坏 API |
| 构建器模式 | 流式 API、错误累积 |
| 错误累积（BuilderError） | 不因首个错误而中断链式调用 |

---

## 📦 模块结构

```
docxgo/
├── docx.go                 # 公共门面：NewDocument(), OpenDocument()
├── builder.go              # DocumentBuilder 流式 API
├── options.go              # 函数式选项
├── domain/                 # 接口层
│   ├── document.go         # Document 接口
│   ├── paragraph.go        # Paragraph 接口
│   ├── run.go              # Run 接口
│   ├── table.go            # Table 接口
│   ├── image.go            # Image 接口
│   ├── section.go          # Section 接口
│   ├── style.go            # Style 类型
│   └── errors.go           # DomainError 类型
├── internal/
│   ├── core/               # 核心实现
│   │   ├── document.go      # Document 实现
│   │   ├── paragraph.go     # Paragraph 实现
│   │   ├── run.go           # Run 实现
│   │   ├── table.go         # Table 实现
│   │   ├── image.go         # Image 实现
│   │   ├── section.go       # Section 实现
│   │   ├── style.go         # 样式管理器
│   │   └── manager.go       # 管理器协调
│   ├── xml/                # OOXML 数据结构
│   │   ├── document.go      # 文档 XML 类型
│   │   ├── paragraph.go     # 段落 XML 类型
│   │   ├── table.go         # 表格 XML 类型
│   │   ├── drawing.go       # 绘图/图片 XML 类型
│   │   └── ...
│   ├── reader/             # 解析已有 docx
│   │   ├── reader.go        # 主读取器
│   │   ├── parser.go        # 元素解析
│   │   ├── package.go       # ZIP 处理
│   │   ├── reconstruct.go   # 从内容重建缺失部件
│   │   └── hydrate*.go      # 字段水合（保留读取数据）
│   ├── serializer/         # XML 生成
│   │   └── serializer.go    # 序列化文档
│   ├── writer/             # ZIP 写入
│   │   └── zip.go           # 写 docx 包
│   └── manager/            # 横切管理器
│       ├── id.go           # ID 生成器
│       ├── relationship.go # 关系管理器
│       ├── media.go        # 媒体管理器
│       ├── style.go        # 样式管理器
│       ├── character_style.go
│       ├── paragraph_style.go
│       └── table_style.go
├── pkg/
│   ├── template/           # 模板/邮件合并引擎
│   └── errors/             # 自定义错误类型
└── examples/               # 13 个示例程序
```

---

## 🎨 设计原则

### 1. 接口隔离

每个 domain 类型都是聚焦的接口：

```go
type Paragraph interface {
    AddRun() (Run, error)
    SetStyle(styleID string) error
    SetAlignment(align Alignment) error
    SetText(text string) error
    Text() string
    // ... 聚焦的方法集
}
```

### 2. 错误累积（构建器）

```go
type BuilderError struct {
    err error
}

func (b *DocumentBuilder) AddHeading(text string, level int) *DocumentBuilder {
    if b.err != nil {
        return b  // 首个错误后不再累积
    }
    // ... 操作，出错时设置 b.err
    return b
}

func (b *DocumentBuilder) Build() (domain.Document, error) {
    return b.doc, b.err  // 返回首个错误
}
```

**好处：**
- 流式 API 不因首个错误而中断
- 易于组合多个操作
- 一次调用即可捕获首个错误

### 3. 函数式选项

```go
func WithTitle(title string) Option {
    return func(o *options) { o.title = title }
}

func WithAuthor(author string) Option {
    return func(o *options) { o.author = author }
}

doc, err := docx.NewDocument(
    docx.WithTitle("My Doc"),
    docx.WithAuthor("Jane"),
)
```

**好处：**
- 向后兼容（新增选项不破坏旧调用）
- 可读性强
- 可组合

### 4. 管理器模式

管理器集中处理横切关注点：

```go
type IDGenerator struct {
    mu   sync.Mutex
    next int
}

func (g *IDGenerator) Generate(prefix string) string {
    g.mu.Lock()
    defer g.mu.Unlock()
    g.next++
    return fmt.Sprintf("%s%d", prefix, g.next)
}
```

---

## 📊 实现阶段

本库经历了多个结构化的开发阶段，每一阶段都建立在前一阶段之上：

### 阶段 1：核心架构 ✅

- [x] Domain 接口定义
- [x] Core 实现
- [x] XML 数据结构
- [x] 序列化器
- [x] ZIP writer
- [x] 管理器（ID、关系、媒体、样式）

### 阶段 2：文档模型 ✅

- [x] 段落与 Run
- [x] 表格（合并、嵌套、样式）
- [x] 图片（内联、浮动）
- [x] 域
- [x] 节与页面布局
- [x] 样式系统

### 阶段 3：构建器 API ✅

- [x] DocumentBuilder
- [x] ParagraphBuilder
- [x] TableBuilder
- [x] 错误累积
- [x] 函数式选项

### 阶段 4：文档读取 ✅

- [x] ZIP 解包
- [x] XML 解析
- [x] 关系加载
- [x] 段落/Run 还原
- [x] 表格还原
- [x] 图片还原
- [x] 样式保留
- [x] 超链接 RelationshipID 保留
- [x] 往返修复（v2.2.1+）

### 阶段 5：高级功能 ✅

- [x] 主题系统（7 套预设，v2.1.0；v2.7.1 起可通过公共 API 发现全部 7 套）
- [x] 段落边框（v2.x）
- [x] 模板/邮件合并引擎（v2.3.0）
- [x] 单元格 run 格式化（v2.5.0）
- [x] 默认校对语言（v2.6.0、v2.7.0 `document.setLanguage`）
- [x] JSON-RPC CLI + Node.js 封装（v2.7.0）
- [x] 原地编辑 RPC 方法（v2.10.0，从 #64 恢复）
- [x] 域代码注入修复（v2.11.0）

### 阶段 6：质量与文档 ✅

- [x] 单元测试（domain 与 pkg/errors 100%，core 约 50-95%）
- [x] 集成测试
- [x] V2 API 指南
- [x] V2 设计文档
- [x] 迁移指南
- [x] 示例（13 个）
- [x] 错误处理指南
- [x] 段落边框文档
- [x] CLI 指南
- [x] 溯源审计（v2.8.0+）
- [x] 强制 CI（CodeQL 等，v2.10.0+）

### 阶段 7：发布与合规 ✅

- [x] MIT 许可证文本（v2.1.1、v2.8.0 最终判定）
- [x] CREDITS.md 归属
- [x] Go 模块路径修复（v2.0.1、v2.1.1）
- [x] npm 发布级联修复（v2.9.0）
- [x] 许可证对齐（v2.8.0+）

---

## 🔄 文档流程

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   source     │     │   internal   │     │    output    │
│   .docx      │ ──▶ │   reader     │ ──▶ │   document   │
│ (ZIP+XML)    │     │   (parse)    │     │   model      │
└──────────────┘     └──────────────┘     └──────┬───────┘
                                                 │
                              ┌──────────────────┼──────────────────┐
                              ▼                  ▼                  ▼
                       ┌────────────┐    ┌────────────┐     ┌────────────┐
                       │  modify    │    │  template  │     │   inspect  │
                       │  (builder) │    │  (merge)   │     │  (metadata)│
                       └─────┬──────┘    └─────┬──────┘     └─────┬──────┘
                             └─────────────────┼─────────────────┘
                                               ▼
                                       ┌──────────────┐
                                       │   serializer  │
                                       │   (XML gen)   │
                                       └──────┬───────┘
                                              ▼
                                       ┌──────────────┐
                                       │    writer    │
                                       │   (ZIP pkg)  │
                                       └──────┬───────┘
                                              ▼
                                       ┌──────────────┐
                                       │  output .docx│
                                       └──────────────┘
```

### 详细步骤

1. **读取**：`internal/reader` 解包 ZIP，解析 XML 部件（document.xml、styles.xml 等）
2. **水合**：保留读取元数据（超链接关系 ID、图片偏移、自定义样式）
3. **建模**：构建内存中的 domain 文档模型
4. **修改**：用户通过 builder 或直接 API 修改模型
5. **序列化**：`internal/serializer` 生成 OOXML XML
6. **写入**：`internal/writer` 打包为 ZIP (.docx)

---

## 📅 历史笔记

### V2 重写与 MIT 发布

v2 系列是对上游 AGPL 时期 `fumiama/go-docx` 的**独立、大幅重写**。经过溯源审计（[PROVENANCE_AUDIT.md](./PROVENANCE_AUDIT.md)）：

- 当前 MIT 发布产物（v2.11.0）**不含任何具有可保护性的 AGPL 实现**
- 任何表面相似源于 ECMA-376 标准与 Go 惯用法，而非上游实现
- 历史 AGPL 快照保留其原始许可证；仅 v2 重写部分以 MIT 分发

### 模块路径注意事项

> ⚠️ **重要**：Go 模块路径为 `github.com/mmonterroca/docxgo/v2`（自 v2.0.1/v2.1.1 起）。尽管 git 远程为 `git@github.com:ckytam/docxgo.git`，但**导入路径不变**。请勿在导入语句中使用 `ckytam/docxgo`。

正确：
```go
import "github.com/mmonterroca/docxgo/v2"
```

错误：
```go
import "github.com/ckytam/docxgo/v2"  // ❌ 会导致构建失败
```

### 版本一致性

> ⚠️ **重要**：截至 v2.9.0，单个语义化版本号在所有发布产物中一致：GitHub release 标签、Go 模块版本、npm 版本与 README 均对齐到同一版本（如 `v2.11.0`）。早期版本存在发布级联漂移（npm 落后于 Go 模块），已在 v2.9.0 修复。

### 已实现的功能（不再规划中）

- ✅ **模板/邮件合并引擎**（v2.3.0）—— 原规划为"未来功能"，现已在 `pkg/template/` 完整实现
- ✅ **单元格 run 格式化**（v2.5.0）—— 斜体/颜色/字号/下划线可在 `table.setCell` 段落项上设置
- ✅ **默认校对语言**（v2.6.0/2.7.0）
- ✅ **JSON-RPC CLI**（v2.7.0）

### 已知限制

- **样式取回**：`paragraph.Style()` 返回 nil（v2.2.0 状态，优先级低）
- **域计算**：以脏标记生成，Word 打开时重新计算（标准行为）
- **`Paragraph.AddField()`**：已弃用，改用 `AddRun()` 后的 `run.AddField()`

---

## 🤝 贡献

详见 [CONTRIBUTING.md](../CONTRIBUTING.md)。设计原则：

1. **保持分层** —— 新功能放在正确的层（domain 接口 / internal 实现）
2. **优先接口** —— 新增能力先在 `domain/` 声明接口
3. **测试先行** —— 为新功能写测试
4. **错误累积** —— 构建器方法返回 `*Builder` 而非 `error`
5. **文档同步** —— 代码与文档在同一 PR 中更新

---

## 📚 参考

- [ECMA-376 Office Open XML](https://www.ecma-international.org/publications-and-standards/standards/ecma-376/)
- [Open XML SDK 文档](https://learn.microsoft.com/en-us/dotnet/api/documentformat.openxml)
- [Go 标准库：encoding/xml](https://pkg.go.dev/encoding/xml)
- [docxgo README](../README.md)
- [V2 API 指南](./V2_API_GUIDE.md)
- [实现状态](./IMPLEMENTATION_STATUS.md)
- [错误处理指南](./ERROR_HANDLING.md)
- [迁移指南](../MIGRATION.md)
- [溯源审计](./PROVENANCE_AUDIT.md)

---

*文档创建：2025 年 10 月*
*最后更新：2026 年 7 月*
*维护者：Misael Monterroca*
