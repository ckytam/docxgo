# CODEBUDDY.md 本文件为 CodeBuddy 在本仓库中工作提供指引。

docxgo 是一个用于读取、构建和修改 `.docx`（Office Open XML Word）文档的 Go 库。Go 模块路径为 `github.com/mmonterroca/docxgo/v2`（Go 1.23）；注意这与 git 远程地址相互独立，当前远程为 `git@github.com:ckytam/docxgo.git`。

## 常用命令

- `go test ./...` — 运行全部 Go 测试。
- `go test -run TestName ./pkg/template/` — 按名称在单个包中运行某个测试。
- `go test ./pkg/template/...` — 运行 template 包（或任意其他路径）的所有测试。
- `go build ./...` — 编译全部代码（可快速发现类型错误）。
- `go run ./cmd/docxgo <args>` 或 `make run ARGS=version` — 运行自带的 CLI。
- `make build` — 将 `docxgo` CLI 二进制构建到 `bin/`。
- `go fmt ./...` — 格式化代码（提交前必须执行）。
- `make examples`（即 `./examples/run_all_examples.sh`）— 运行示例程序。
- `dotnet run --project DocxValidator -- <file.docx>` — 用 OOXML 架构校验生成的 `.docx`（需要 .NET 8）。重新生成示例 fixture 后应运行此命令。
- `go test -race ./...` — 竞态检测；`go test -cover ./...` — 覆盖率。

> Windows 注意：部分 `internal/core` 测试仍硬编码了 `/tmp/`（例如 `TestDocument_SaveAs`），在 Windows 下会失败。这是既存问题，与功能改动无关；除非用户要求，否则不要把它当成手头工作顺手「修复」。

## 架构

docxgo 采用清晰的的分层架构。依赖方向严格为 `domain`（接口）→ `internal`（实现）→ `pkg`（公共辅助），顶层由 `docx.go`/`builder.go` 作为公开入口。

### 第一层 — `domain/`（接口与值类型）
纯 Go 接口与小型数据类型，**不含 XML、不含实现**。定义了 `Document`、`Paragraph`、`Run`、`Table`、`Section`、`Style`、`Image`，以及 `Block` 类型。一个 `Document` 以**两个并行切片**呈现段落与更高层的块（段落/表格）——`paragraphs` 与 `blocks`。任何插入或删除段落的代码都必须同步维护这两个切片（参见 `internal/core/document.go` 中的 `blockIndexForParagraphIndex`）。要为文档新增公开能力，几乎总是需要：(1) 在 `domain/document.go` 中声明，(2) 在 `internal/core/document.go` 中实现。

### 第二层 — `internal/`（实现）
- `internal/core/` — `domain` 接口的具体实现（如 `core.Document` 实现 `domain.Document`）。内存模型在此处，多数文档修改逻辑也在此实现。
- `internal/xml/` — 底层 OOXML（反）序列化类型，镜像 schema（`w:p`、`w:r`、`w:tbl`、编号、样式等）。可视为原始 XML 的 AST。
- `internal/reader/` — 读取 `.docx` zip 包并把 XML 解析为 `core` 模型。`package.go` 打开 zip，`parser.go`/`reader.go` 遍历 XML，`reconstruct.go` **重建缺失部件**（如推断的样式/关系），使文档即便 Word 省略了某些内容也能往返保存。正是这种重建机制导致部分内容（尤其是往返文档中的页眉/页脚）会被原样写回，并被某些查找替换逻辑有意跳过。
- `internal/serializer/` — 将模型转回 OOXML XML（`latent_styles.go` 负责样式序列化）。
- `internal/writer/` — `zip.go` 将序列化后的部件写回 `.docx`（即 zip 容器）。
- `internal/manager/` — 横向协调器：`id.go`（唯一 ID）、`relationship.go`（部件关系）、`media.go`（图像二进制及其关系）、以及样式管理器（`character_style.go`、`paragraph_style.go`、`table_style.go`）。`media.go.UpdateDataByPath` 在保持关系有效的前提下替换图像字节。

「读取→编辑→保存」的数据流：`docx.OpenDocument` → `reader` + `xml` 解析为 `core` 模型 → 调用方通过 `domain` API 修改 → `serializer` + `writer` + `manager` 生成新的 `.docx` zip。

### 第三层 — `pkg/`（公共辅助）
- `pkg/template/` — 邮件合并/模板引擎。关键类型：`MergeData`（标量 `{{key}}` 值映射）、`ForeachItem`（循环字段值映射）、`ForeachLoops`。关键函数：
  - `MergeTemplate(doc, data, opts...)` — 标量替换。
  - `MergeTemplateWithLoops(doc, data, loops)` — 重复包含 `{{#foreach name}}`…`{{/each}}` 标记的**表格行**（每个条目克隆一次）。
  - `MergeTemplateWithBodyLoops(doc, data, loops)` — 重复上述标记之间的**多段落块**。由 `foreach.go` 的 `expandBodyForeach` 实现，使用 `doc.InsertParagraph`/`doc.DeleteParagraph`；开/闭标记必须各自位于**单个 run** 内，空循环会移除其模板块。
  - `ReplaceImage(doc, marker, bytes)` / `ReplaceImageFromFile` / `ReplaceImages` — 替换 alt-text 含有字面 `{{IMAGE .KEY}}` 标记的图像，保留尺寸/位置/关系。底层依赖 `internal/core/image.go:replaceData` 与 `manager/media.go:UpdateDataByPath`；返回 `ReplaceImageResult{Replaced, Skipped, Errors}`（在往返保留的页眉/页脚中的匹配会被计入 `Skipped`）。
  - `merge.go:replaceParagraph` — 合并所有 run，对**段落拼接后的整段文本**做匹配，再通过 `replaceSpan` 写回。这是有意为之：Word 常把一个 `{{KEY}}` 占位符拆到多个不同格式的 run 中，因此必须基于整段文本而非逐 run 匹配才能保证正确。
- `pkg/errors/` — 带类型的哨兵错误（如 `errors.InvalidArgument`），被广泛使用；对非法入参应返回它们而非 `fmt.Errorf`。
- `pkg/constants/`、`pkg/color/` — 共享枚举与颜色辅助。

### 顶层
- `docx.go` — 公开的 `OpenDocument`/`SaveAs`。
- `builder.go` — 流式 `DocumentBuilder` API，用于从零创建文档。
- `options.go`、`themes/` — 配置与主题辅助。
- `cmd/docxgo/` — 封装公开 API 的 CLI。
- `npm/` — TypeScript/Node.js 绑定，把同一公开 API 暴露给 JS；不属于 Go 构建。
- `DocxValidator/` — 独立的 C#（.NET 8）项目，用 OOXML 架构校验 `.docx`；按上文用 `dotnet` 运行。

## 工作流约定（摘自 CONTRIBUTING.md）

- 主干式开发：从 `master` 分支，向 `master` 提 PR。分支前缀使用 `feature/`、`fix/`、`docs/`、`test/`、`refactor/`、`perf/`、`chore/`。
- 提交信息遵循 [Conventional Commits](https://www.conventionalcommits.org/)：`feat:`、`fix:`、`docs:`、`test:`、`refactor:`、`perf:`、`chore:`。
- 提交前运行 `go fmt ./...` 并确保 `go test ./...` 通过。新代码目标覆盖率 ≥95%。
- 新增功能时保持 README、示例与 CHANGELOG 同步。
- 重新生成 `.docx` fixture 后，用 `DocxValidator` 项目校验后再开 PR。
