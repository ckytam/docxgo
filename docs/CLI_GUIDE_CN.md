# docxgo CLI 指南

`docxgo` 命令行二进制程序通过 stdin/stdout 以 JSON-RPC 服务的形式暴露了完整的 docxgo 库 API，使得从任何语言创建和处理文档成为可能。

## 目录

- [安装](#installation)
- [执行模式](#execution-modes)
- [JSON-RPC 协议](#json-rpc-protocol)
- [方法参考](#methods-reference)
  - [system.ping](#systemping)
  - [system.version](#systemversion)
  - [system.capabilities](#systemcapabilities)
  - [system.batch](#systembatch)
  - [document.create](#documentcreate)
  - [document.open](#documentopen)
  - [document.save](#documentsave)
  - [document.validate](#documentvalidate)
  - [document.inspect](#documentinspect)
  - [document.setMetadata](#documentsetmetadata)
  - [document.setBackgroundColor](#documentsetbackgroundcolor)
  - [document.setLanguage](#documentsetlanguage)
  - [document.addContent](#documentaddcontent)
  - [document.addPageBreak](#documentaddpagebreak)
  - [document.applyPatch](#documentapplypatch)
  - [document.replaceText](#documentreplacetext)
  - [paragraph.add](#paragraphadd)
  - [paragraph.list](#paragraphlist)
  - [paragraph.setText](#paragraphsettext)
  - [table.add](#tableadd)
  - [table.list](#tablelist)
  - [table.getCell](#tablegetcell)
  - [table.setCell](#tablesetcell)
  - [section.add](#sectionadd)
  - [template.inspect](#templateinspect)
  - [template.render](#templaterender)
  - [document.close](#documentclose)
- [内容类型](#content-types)
  - [段落](#paragraph)
  - [表格](#table)
  - [节](#section)
  - [分页符](#page-break)
- [Shell 示例](#shell-examples)
- [Node.js 集成](#nodejs-integration)
- [错误码参考](#error-code-reference)

---

## 安装

从源码构建二进制文件：

```bash
go install github.com/mmonterroca/docxgo/v2/cmd/docxgo@latest
```

或在本地构建：

```bash
git clone https://github.com/mmonterroca/docxgo.git
cd docxgo
go build -o docxgo ./cmd/docxgo/
```

验证安装：

```bash
docxgo version
# 2.11.0
```

---

## 执行模式

### 一次性模式（`docxgo exec`）

读取单个 JSON-RPC 请求，执行它，将 JSON 响应写入 stdout，然后退出。

```bash
# 从 stdin 读取：
echo '{"id":1,"method":"document.create","params":{...}}' | docxgo exec

# 从 flag 读取：
docxgo exec --request '{"id":1,"method":"document.create","params":{...}}'
```

适合：通过 `child_process.execFile`、Shell 脚本、CI 流水线进行的偶发调用。

**退出码：**
- `0` — 成功
- `1` — 错误（响应中会包含一个 JSON 错误对象）

### RPC 模式（`docxgo rpc`）

从 stdin 读取以换行分隔的 JSON 请求，并将以换行分隔的 JSON 响应写入 stdout。进程会在请求之间保持存活。

```bash
docxgo rpc
```

诊断日志会写入 **stderr**（而非 stdout），因此它们不会干扰 JSON 响应。

关闭触发条件：
- stdin 上的 EOF（管道关闭）
- `SIGTERM` 或 `SIGINT` 信号

适合：高频使用、Lambda 预热启动、通过 `child_process.spawn` 进行批处理。

---

## JSON-RPC 协议

### 请求格式

```json
{
  "id": 1,
  "method": "document.create",
  "params": { ... }
}
```

`id` 可以是任意 JSON 值（数字、字符串、null）。它会在响应中原样回传。

### 成功响应

```json
{
  "id": 1,
  "result": { ... }
}
```

### 错误响应

```json
{
  "id": 1,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "description of the error",
    "operation": "document.create"
  }
}
```

某些方法会返回带有 `data` 字段的增强错误，其中包含结构化的上下文信息：

```json
{
  "id": 1,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "unknown operation \"deleteAll\" at index 2",
    "operation": "document.applyPatch",
    "data": {
      "index": 2,
      "op": "deleteAll"
    }
  }
}
```

| `data` 字段 | 类型 | 说明 |
|--------------|------|-------------|
| `index` | Number | 失败操作的索引（在批处理/补丁中） |
| `category` | String | 错误类别（例如 `"merge"`） |
| `retryable` | Boolean | 该错误是否可重试 |
| `op` | String | 失败的操作 |

---

## 方法参考

### system.ping

健康检查 —— 验证 RPC 进程是否存活且可响应。

**参数：** 无（或 `{}`）。

**成功结果：**

```json
{ "status": "ok" }
```

---

### system.version

返回版本、协议版本以及平台信息。

**参数：** 无（或 `{}`）。

**成功结果：**

```json
{
  "name": "docxgo",
  "version": "2.11.0",
  "protocolVersion": "1.0",
  "goVersion": "go1.23.0",
  "platform": "darwin",
  "arch": "arm64"
}
```

---

### system.capabilities

返回当前二进制文件所支持特性的映射。

**参数：** 无（或 `{}`）。

**成功结果：**

```json
{
  "rpc": true,
  "template": true,
  "mailMerge": true,
  "inspect": true,
  "validate": true,
  "batch": true,
  "applyPatch": true,
  "setLanguage": true,
  "replaceText": true,
  "cellEdit": true,
  "streaming": false,
  "partialUpdate": false
}
```

在调用高级方法前，可用此结果做特性检测。

---

### system.batch

在单次往返中执行多个 RPC 请求。每个子请求按顺序依次处理。嵌套的 `system.batch` 调用会被拒绝。

**参数：**

| 字段 | 类型 | 必填 | 说明 |
|-------|------|----------|-------------|
| `requests` | Array | 是 | `{ method, params? }` 对象组成的数组 |

**成功结果：**

```json
{
  "responses": [
    { "result": { "status": "ok" } },
    { "result": { "name": "docxgo", "version": "2.11.0", ... } },
    { "error": { "code": "NOT_FOUND", "message": "..." } }
  ]
}
```

`responses` 中的每一项包含 `result` 或 `error`，顺序与输入 `requests` 数组一致。

**示例：**

```json
{
  "id": 1,
  "method": "system.batch",
  "params": {
    "requests": [
      { "method": "system.ping" },
      { "method": "document.create", "params": {
        "content": [{ "type": "paragraph", "runs": [{ "text": "Hello" }] }],
        "output": "buffer"
      }},
      { "method": "document.inspect", "params": { "documentId": "doc-1" } }
    ]
  }
}
```

---

### document.create

创建一个新 Word 文档，以 base64 形式返回，或保存到文件。

**参数：**

| 字段 | 类型 | 必填 | 说明 |
|-------|------|----------|-------------|
| `options` | Object | 否 | 文档配置（见下文） |
| `content` | Array | 否 | 有序的内容项列表 |
| `output` | `"buffer"` \| `"file"` | 否 | 输出格式（默认 `"buffer"`） |
| `filePath` | String | 当 `output="file"` 时 | 写入 `.docx` 的路径 |

**options 字段：**

| 字段 | 类型 | 说明 |
|-------|------|-------------|
| `title` | String | 文档标题 |
| `author` | String | 文档作者 |
| `subject` | String | 文档主题 |
| `pageSize` | `"A4"` \| `"Letter"` \| `"Legal"` \| `"A3"` \| `"Tabloid"` \| `{width, height}` | 页面尺寸（twips） |
| `margins` | `"normal"` \| `"narrow"` \| `"wide"` \| `{top, bottom, left, right}` | 页边距（twips） |
| `theme` | `"Corporate"` \| `"Startup"` \| `"Modern"` \| `"Fintech"` \| `"Academic"` | 套用预设主题 |

**成功结果：**

```json
{
  "data": "<base64-encoded .docx>",
  "documentId": "doc-1"
}
```

（当 `output="buffer"` 时）

```json
{
  "filePath": "/path/to/output.docx",
  "documentId": "doc-1"
}
```

（当 `output="file"` 时）

`documentId` 可用于后续 RPC 调用来修改或重新保存该文档。

**示例：**

```json
{
  "id": 1,
  "method": "document.create",
  "params": {
    "options": {
      "title": "My Report",
      "author": "Jane Smith",
      "pageSize": "A4",
      "margins": "normal",
      "theme": "Corporate"
    },
    "content": [
      {
        "type": "paragraph",
        "style": "Heading1",
        "runs": [{ "text": "Introduction" }]
      },
      {
        "type": "paragraph",
        "runs": [
          { "text": "This is " },
          { "text": "bold", "bold": true },
          { "text": " text." }
        ]
      }
    ],
    "output": "buffer"
  }
}
```

---

### document.open

打开一个已有的 `.docx` 文档并存入会话。

**参数：**

| 字段 | 类型 | 必填 | 说明 |
|-------|------|----------|-------------|
| `filePath` | String | 二选一 | 磁盘上 `.docx` 文件路径 |
| `base64` | String | 二选一 | Base64 编码的 `.docx` 字节 |

**成功结果：**

```json
{ "documentId": "doc-2" }
```

---

### document.save

保存或重新导出此前已创建或打开的文档。

**参数：**

| 字段 | 类型 | 必填 | 说明 |
|-------|------|----------|-------------|
| `documentId` | String | 是 | 会话文档 ID |
| `output` | `"buffer"` \| `"file"` | 否 | 输出格式（默认 `"buffer"`） |
| `filePath` | String | 当 `output="file"` 时 | 目标文件路径 |

**成功结果：** 与 `document.create` 结构相同。

---

### document.validate

校验文档结构并返回结果。

**参数：**

| 字段 | 类型 | 必填 | 说明 |
|-------|------|----------|-------------|
| `documentId` | String | 是 | 会话文档 ID |

**成功结果：**

```json
{ "valid": true }
```

或

```json
{ "valid": false, "message": "document is empty" }
```

---

### document.inspect

从文档中提取元数据、文本与结构信息。

**参数：**

| 字段 | 类型 | 必填 | 说明 |
|-------|------|----------|-------------|
| `documentId` | String | 是 | 会话文档 ID |

**成功结果：**

```json
{
  "paragraphCount": 5,
  "tableCount": 1,
  "text": ["First paragraph text", "Second paragraph text"],
  "metadata": {
    "title": "My Document",
    "subject": "",
    "creator": "Jane Smith",
    "description": "",
    "keywords": null,
    "created": "",
    "modified": ""
  },
  "backgroundColor": "#E0F0FF",
  "language": { "val": "es-MX", "eastAsia": "", "bidi": "" }
}
```

（`metadata` 未设置时省略；`backgroundColor` 未设置时省略；当文档未设置默认校订语言时 `language` 省略。）

---

### document.setMetadata

更新文档元数据。

**参数：**

| 字段 | 类型 | 必填 | 说明 |
|-------|------|----------|-------------|
| `documentId` | String | 是 | 会话文档 ID |
| `title` | String | 否 | 文档标题 |
| `subject` | String | 否 | 文档主题 |
| `creator` | String | 否 | 文档作者/创建者 |
| `description` | String | 否 | 文档描述 |
| `keywords` | Array\<String\> | 否 | 关键词列表 |
| `created` | String | 否 | 创建日期（ISO 8601） |
| `modified` | String | 否 | 修改日期（ISO 8601） |

**成功结果：** `{ "ok": true }`

---

### document.setBackgroundColor

设置整个文档的页面背景色。

**参数：**

| 字段 | 类型 | 必填 | 说明 |
|-------|------|----------|-------------|
| `documentId` | String | 是 | 会话文档 ID |
| `color` | String | 是 | 十六进制颜色字符串（如 `"#E8F0FE"` 或 `"E8F0FE"`） |

**成功结果：** `{ "ok": true }`

---

### document.setLanguage

设置文档的默认校订语言，Word 用它做拼写/语法检查。标签采用 BCP 47（如 `"es-MX"`、`"en-US"`）。`val`、`eastAsia`、`bidi` 中至少提供一个。

**仅对通过 `document.create` 创建的文档有效。** 通过 `document.open` 打开的文档保留了往返得到的 `styles.xml`/`settings.xml` 字节，`SetLanguage` 拒绝改动这些字节（语言变更永远无法真正写进保存的文件）。`document.inspect` 仍会报告打开文档的语言，因为读取器在文档打开时是从 `styles.xml` 之外单独注入该信息的。

**参数：**

| 字段 | 类型 | 必填 | 说明 |
|-------|------|----------|-------------|
| `documentId` | String | 是 | 会话文档 ID |
| `val` | String | `val`/`eastAsia`/`bidi` 至少一个 | 主语言标签，应用于拉丁字母文本 |
| `eastAsia` | String | `val`/`eastAsia`/`bidi` 至少一个 | 东亚（CJK）文字 run 的语言标签 |
| `bidi` | String | `val`/`eastAsia`/`bidi` 至少一个 | 从右向左（bidi）文字 run 的语言标签 |

**成功结果：** `{ "ok": true }`

**示例：**

```json
{
  "id": 6,
  "method": "document.setLanguage",
  "params": { "documentId": "doc-1", "val": "es-MX" }
}
```

---

### document.addContent

向已有文档会话追加内容。接受与 `document.create` 相同的内容数组格式。这是修改通过 `document.open` 打开的文档的主要方法。

**参数：**

| 字段 | 类型 | 必填 | 说明 |
|-------|------|----------|-------------|
| `documentId` | String | 是 | 会话文档 ID |
| `content` | Array | 是 | 有序的内容项列表（与 `document.create` 格式相同） |

**成功结果：** `{ "ok": true }`

**示例：**

```json
{
  "id": 5,
  "method": "document.addContent",
  "params": {
    "documentId": "doc-1",
    "content": [
      {
        "type": "paragraph",
        "runs": [{ "text": "Appended paragraph", "bold": true }]
      },
      { "type": "pageBreak" },
      {
        "type": "table",
        "rows": [
          { "cells": [{ "paragraphs": [{ "runs": [{ "text": "A1" }] }] }] }
        ]
      }
    ]
  }
}
```

---

### document.addPageBreak

向已有文档追加一个分页符。

**参数：**

| 字段 | 类型 | 必填 | 说明 |
|-------|------|----------|-------------|
| `documentId` | String | 是 | 会话文档 ID |

**成功结果：** `{ "ok": true }`

---

### document.applyPatch

按顺序对已有文档应用一系列补丁操作。**这不是原子的。** 如果某操作失败，后续操作**不会**被应用，并且错误会包含失败索引，以及失败前成功应用的操作数（`applied`）。失败前已应用的操作仍然生效——没有回滚，因此文档可能处于部分补丁状态。请利用错误中的 `applied` 计数来决定是重试剩余操作还是丢弃该文档。

**参数：**

| 字段 | 类型 | 必填 | 说明 |
|-------|------|----------|-------------|
| `documentId` | String | 是 | 会话文档 ID |
| `operations` | Array | 是 | 补丁操作对象数组 |

**支持的操作：**

| `op` 值 | 说明 | 附加字段 |
|------------|-------------|-------------------|
| `appendParagraph` | 追加段落 | 同 `paragraph.add` 字段（style、alignment、runs 等） |
| `appendTable` | 追加表格 | 同 `table.add` 字段（rows、style、alignment、width） |
| `appendSection` | 追加分节符 | 同 `section.add` 字段（breakType、pageSize、orientation 等） |
| `appendPageBreak` | 追加分页符 | 无 |
| `setMetadata` | 设置文档元数据 | 同 `document.setMetadata` 字段（title、creator 等） |
| `setBackgroundColor` | 设置背景色 | `color`（十六进制字符串） |
| `setLanguage` | 设置校订语言 | 同 `document.setLanguage` 字段（`val`、`eastAsia`、`bidi`）——在通过 `document.open` 打开的文档上同样会因往返保护而失败 |

**成功结果：**

```json
{ "ok": true, "applied": 3 }
```

**错误结果（带丰富 `data`）：**

```json
{
  "code": "VALIDATION_ERROR",
  "message": "unknown operation \"deleteAll\" at index 2",
  "operation": "document.applyPatch",
  "data": { "index": 2, "op": "deleteAll", "applied": 2 }
}
```

**示例：**

```json
{
  "id": 10,
  "method": "document.applyPatch",
  "params": {
    "documentId": "doc-1",
    "operations": [
      {
        "op": "appendParagraph",
        "style": "Heading1",
        "runs": [{ "text": "New Section" }]
      },
      { "op": "appendPageBreak" },
      {
        "op": "appendTable",
        "rows": [
          { "cells": [{ "paragraphs": [{ "runs": [{ "text": "A1" }] }] }] }
        ]
      },
      { "op": "setMetadata", "title": "Updated Title" },
      { "op": "setBackgroundColor", "color": "#F0F8FF" }
    ]
  }
}
```

---

### document.replaceText

将正文字段落、表格单元格、页眉与页脚中每一个字面字符串出现替换为另一个字符串。

匹配区分大小写。被拆到多个格式相同的 run 中的文本会被当作一个 run 来匹配；而跨越*不同*格式 run 的匹配仍会被替换，并采用它触碰到的第一个 run 的格式——这会抹平该格式。例如，在 `"Status: **PENDING** (review)"` 中把 `": PENDING ("` 替换为 `": DONE ("`，会得到 `"Status: DONE (review)"`，加粗丢失，因为匹配跨越了普通/加粗边界。

某些匹配会被计入 `skipped` 而非被替换：

- 触碰带有**域**（页码、目录、MERGEFIELD、超链接）的 run 的匹配总是被跳过。Word 在打开文档时会重新生成域的显示文本，因此替换它会报告一个从未出现的变更。这也包括自身不含文本的 run 上的域——这是 Word 用来承载 MERGEFIELD 的形式，即域处于空 run 中，紧挨着承载其显示文本的 run。
- 跨越**多个** run 的匹配，当其两端之间的任何 run 带有换行/分页或图片时也会跳过，因为折叠这些 run 会让该内容卡在替换结果中间。完全落在携带换行或图片的*单个* run 内的匹配会被正常替换，且换行或图片被保留。
- 对于通过 `document.open` 打开的文档，只要文档存在任何被保留的页眉或页脚部件，其中的**每个**页眉/页脚匹配都会被跳过。打开文档的页眉/页脚 XML 在保存时按字节原样写回（见 `document.save`），因此内存中的替换永远不会抵达保存的文件——把它报告为 `replaced` 将是一次虚假成功。这是一个文档级检查，而非逐页眉/页脚检查：只要任一页眉或页脚被保留，所有页眉/页脚匹配都被跳过，即使在自身页眉/页脚未被触碰的节中也是如此。

这不会深入嵌套在其他表格中的表格，也不会进入位于页眉或页脚中的表格——与 `template.render`/`template.inspect` 相同的遍历限制，它们共享底层的段落遍历。

**参数：**

| 字段 | 类型 | 必填 | 说明 |
|-------|------|----------|-------------|
| `documentId` | String | 是 | 会话文档 ID |
| `find` | String | 是 | 待搜索的字面文本；不得为空 |
| `replace` | String | 是 | 替换文本；空字符串会删除每个匹配 |

参数采用严格解码：无法识别的字段会被拒绝，而非静默忽略。

**成功结果：**

```json
{ "replaced": 2, "skipped": 1 }
```

**示例：**

```json
{
  "id": 11,
  "method": "document.replaceText",
  "params": {
    "documentId": "doc-1",
    "find": "{{status}}",
    "replace": "Approved"
  }
}
```

---

### paragraph.add

向已有文档追加单个段落。支持的段落属性与内容数组相同（style、alignment、spacing、runs 等）。

**参数：**

| 字段 | 类型 | 必填 | 说明 |
|-------|------|----------|-------------|
| `documentId` | String | 是 | 会话文档 ID |
| `style` | String | 否 | 段落样式名 |
| `alignment` | String | 否 | `left`、`center`、`right`、`justify`、`distribute` |
| `spacingBefore` | Number | 否 | 段前间距（twips）。省略则保留未设置；`0` 会被当作显式覆盖，包括覆盖样式自身的间距 |
| `spacingAfter` | Number | 否 | 段后间距（twips）。省略则保留未设置；`0` 会被当作显式覆盖，包括覆盖样式自身的间距 |
| `lineSpacing` | Object | 否 | `{ "rule": "auto", "value": 360 }` |
| `indent` | Object | 否 | `{ "left", "right", "firstLine", "hanging" }`。每一侧独立：省略某侧则保留未设置，或设为 `0` 以仅在该侧显式覆盖样式自身的值 |
| `numbering` | Object | 否 | `{ "id": 1, "level": 0 }` |
| `borders` | Object | 否 | 段落边框 |
| `runs` | Array | 否 | 文本 run（与内容段落格式相同） |

**成功结果：**

```json
{ "ok": true, "index": 3 }
```

`index` 是新段落的零基位置。

**示例：**

```json
{
  "id": 6,
  "method": "paragraph.add",
  "params": {
    "documentId": "doc-1",
    "style": "Heading1",
    "alignment": "center",
    "runs": [
      { "text": "New Section Title", "bold": true, "fontSize": 18 }
    ]
  }
}
```

---

### paragraph.list

列出文档中所有段落及其文本与样式。

**参数：**

| 字段 | 类型 | 必填 | 说明 |
|-------|------|----------|-------------|
| `documentId` | String | 是 | 会话文档 ID |

**成功结果：**

```json
{
  "count": 3,
  "paragraphs": [
    { "index": 0, "text": "Introduction", "style": "Heading1" },
    { "index": 1, "text": "Some body text." },
    { "index": 2, "text": "" }
  ]
}
```

---

### paragraph.setText

按索引（即 `paragraph.list` 报告的索引）替换正文段落的内容。它会在写入新内容前清空该段落所有已有 run，因此这是对该段落*内容*（文本与 run）的完整替换——而非追加。

它并非对该段落*属性*的完整替换。样式、对齐、缩进与间距只会被你传入的字段触及；省略的字段保留段落已有的内容。在 Heading1、居中的段落上用 `paragraph.setText({"index": 0, "text": "..."})` 修正错别字，会保持其为 Heading1、居中段落——不会重置为普通左对齐段落。要清除某个属性，请将其显式设为默认值（如 `"alignment": "left"`）。

接受与 `paragraph.add` 相同的内容字段，但采用严格解码：无法识别的字段或 `null` 值会被拒绝，而非静默当作缺失，因为会替换内容的请求中的拼写错误绝不能读作"清空它"。只传 `style`（无 `text` 且无 `runs`）会清空段落的 run 且不写回任何内容——段落被清空，而非保持不变。此方法只处理文档正文中的段落；位于表格单元格或页眉/页脚中的段落无法通过索引访问——单元格内容请用 `table.setCell`。

**参数：**

| 字段 | 类型 | 必填 | 说明 |
|-------|------|----------|-------------|
| `documentId` | String | 是 | 会话文档 ID |
| `index` | Number | 是 | 零基正文段落索引，由 `paragraph.list` 报告 |
| `text` \| `runs` \| `style` \| ... | — | 否 | 与 `paragraph.add` 相同的内容字段 |

**成功结果：**

```json
{ "ok": true, "index": 0 }
```

**示例：**

```json
{
  "id": 12,
  "method": "paragraph.setText",
  "params": {
    "documentId": "doc-1",
    "index": 0,
    "runs": [
      { "text": "Completed", "bold": true }
    ]
  }
}
```

---

### table.add

向已有文档追加表格。使用与内容数组相同的表格格式。

**参数：**

| 字段 | 类型 | 必填 | 说明 |
|-------|------|----------|-------------|
| `documentId` | String | 是 | 会话文档 ID |
| `rows` | Array | 是 | 表格行（与内容表格格式相同） |
| `style` | String | 否 | 表格样式名 |
| `alignment` | String | 否 | 表格对齐 |
| `width` | Object | 否 | `{ "type": "dxa", "value": 9000 }` |

**成功结果：**

```json
{ "ok": true, "index": 0 }
```

`index` 是新表格的零基位置。

**示例：**

```json
{
  "id": 7,
  "method": "table.add",
  "params": {
    "documentId": "doc-1",
    "style": "TableGrid",
    "rows": [
      {
        "cells": [
          { "paragraphs": [{ "runs": [{ "text": "Name", "bold": true }] }] },
          { "paragraphs": [{ "runs": [{ "text": "Value", "bold": true }] }] }
        ]
      },
      {
        "cells": [
          { "paragraphs": [{ "runs": [{ "text": "Score" }] }] },
          { "paragraphs": [{ "runs": [{ "text": "95" }] }] }
        ]
      }
    ]
  }
}
```

---

### table.list

列出文档中所有表格及其尺寸。

**参数：**

| 字段 | 类型 | 必填 | 说明 |
|-------|------|----------|-------------|
| `documentId` | String | 是 | 会话文档 ID |
| `includeText` | Boolean | 否 | 为 true 时在每个表格列表中附加单元格文本（默认 `false`） |

**成功结果：**

```json
{
  "count": 2,
  "tables": [
    { "index": 0, "rows": 3, "columns": 2 },
    { "index": 1, "rows": 5, "columns": 4 }
  ]
}
```

当 `includeText: true` 时，每个表格项还会获得一个 `cells` 键：`[][]string`，每行一个子数组，该行实际持有的每个单元格一个条目。每个单元格的段落以 `"\n"` 连接。`includeText` 省略或为 `false` 时，`cells` 键完全不存在。

注意：这**不**告诉你合并单元格的情况。通过 `document.open` 读取的文档在重建时每一行都填充到完整的列网格，被水平合并覆盖的单元格以空占位符保留——因此一个由两次 2 列合并构成的 4 列行会列为 `["Overview", "", "Detail", ""]`，而非两项。这些空字符串是合并的延续，而非空单元格，`table.setCell` 会拒绝写入它们（其内容永不会被序列化）。只有在本会话中通过 `document.create` 构建、确实参差不齐的行，报告的单元格数才会少于 `columns`。

```json
{
  "count": 1,
  "tables": [
    {
      "index": 0,
      "rows": 2,
      "columns": 2,
      "cells": [
        ["Question", "Answer"],
        ["Do you encrypt data at rest?", "TBD"]
      ]
    }
  ]
}
```

---

### table.getCell

按表格、行、列索引读取单个单元格的内容。

**参数：**

| 字段 | 类型 | 必填 | 说明 |
|-------|------|----------|-------------|
| `documentId` | String | 是 | 会话文档 ID |
| `tableIndex` | Number | 是 | 零基表格索引，由 `table.list` 报告 |
| `rowIndex` | Number | 是 | 零基行索引 |
| `columnIndex` | Number | 是 | 零基列索引 |

参数采用严格解码：无法识别的字段会被拒绝，而非静默忽略。

**成功结果：**

```json
{ "text": "TBD", "paragraphs": ["TBD"], "paragraphCount": 1 }
```

`text` 是单元格段落以 `"\n"` 连接的结果。

---

### table.setCell

按表格、行、列索引替换单个单元格的内容。

提供 `text`（写入一个普通段落的快捷方式）与 `paragraphs`（富段落项，与 `paragraph.add` 同形）中的恰好一个。两者**都**提供会被拒绝，**都**不提供也会被拒绝——此方法替换单元格内容，因此一个完全不指明内容的请求（比如字段名拼写错误）属于错误，而非静默清空。要清空单元格，传 `"paragraphs": []` 或 `"text": ""`。

与 `paragraph.setText` 一样，这替换的是段落*内容*，而非段落*属性*——复用的段落槽位保留其已有的样式、对齐、缩进与间距，除非对应的段落项字段另有说明。

在触碰单元格之前会先校验整个请求：如果任何段落项格式错误或被拒，调用失败且单元格保留原内容。每个字段——顶层参数与每个段落项——都采用严格解码：无法识别的字段或 `null` 项会被拒绝，而非静默当作空段落，因为会替换内容的请求中的拼写错误绝不能读作"清空它"。

写入被水平合并覆盖的单元格会被拒绝。该单元格永远不会被写入文件，因此接受写入会报告成功，却在保存时静默丢弃内容——请改为以合并区域的起始单元格为目标。

写入垂直合并延续单元格也会被拒绝。与水平合并延续不同，该单元格仍会被写入文件，但 Word 会在其位置渲染垂直合并重启单元格的内容，因此写入会报告成功却不可见——请改为以合并区域最顶端的单元格为目标。

单元格的段落数始终与你所提供项数一致：项少于单元格原有数会移除尾部段落，多于则追加新段落。结果中的 `paragraphCount` 始终等于写入的项数。

**参数：**

| 字段 | 类型 | 必填 | 说明 |
|-------|------|----------|-------------|
| `documentId` | String | 是 | 会话文档 ID |
| `tableIndex` | Number | 是 | 零基表格索引 |
| `rowIndex` | Number | 是 | 零基行索引 |
| `columnIndex` | Number | 是 | 零基列索引 |
| `text` | String | 二选一 | 纯文本快捷方式；与 `paragraphs` 互斥。`""` 会清空单元格 |
| `paragraphs` | Array | 二选一 | 富段落项；与 `text` 互斥。`[]` 会清空单元格 |

**成功结果：**

```json
{ "ok": true, "paragraphCount": 1 }
```

**示例：**

```json
{
  "id": 13,
  "method": "table.setCell",
  "params": {
    "documentId": "doc-1",
    "tableIndex": 0,
    "rowIndex": 1,
    "columnIndex": 1,
    "text": "Yes"
  }
}
```

---

### section.add

向已有文档追加新节。支持页面尺寸、边距、方向、分栏以及页眉/页脚。

**参数：**

| 字段 | 类型 | 必填 | 说明 |
|-------|------|----------|-------------|
| `documentId` | String | 是 | 会话文档 ID |
| `breakType` | String | 否 | `nextPage`（默认）、`continuous`、`evenPage`、`oddPage` |
| `pageSize` | String/Object | 否 | 页面尺寸预设或 `{width, height}` |
| `margins` | String/Object | 否 | 边距预设或 `{top, bottom, left, right}` |
| `orientation` | String | 否 | `portrait` 或 `landscape` |
| `columns` | Number | 否 | 文字栏数 |
| `headers` | Object | 否 | 按类型（`default`、`first`、`even`）的页眉 |
| `footers` | Object | 否 | 按类型（`default`、`first`、`even`）的页脚 |

**成功结果：**

```json
{ "ok": true, "index": 1 }
```

`index` 是新节的零基位置。

**示例：**

```json
{
  "id": 8,
  "method": "section.add",
  "params": {
    "documentId": "doc-1",
    "breakType": "nextPage",
    "pageSize": "A4",
    "orientation": "landscape",
    "columns": 2
  }
}
```

---

### template.inspect

扫描文档中的模板占位符（默认 `{{key}}`）并返回每个出现的详细信息。

**参数：**

| 字段 | 类型 | 必填 | 说明 |
|-------|------|----------|-------------|
| `documentId` | String | 是 | 会话文档 ID |
| `openDelimiter` | String | 否 | 自定义起始分隔符（默认 `"{{"`） |
| `closeDelimiter` | String | 否 | 自定义结束分隔符（默认 `"}}"`） |

**成功结果：**

```json
{
  "placeholders": ["Name", "Company", "Role"],
  "count": 3,
  "occurrences": 4,
  "details": [
    {
      "name": "Name",
      "fullMatch": "{{Name}}",
      "location": "paragraph",
      "paragraph": 0,
      "run": 0
    },
    {
      "name": "Company",
      "fullMatch": "{{Company}}",
      "location": "tableCell",
      "paragraph": 0,
      "run": 0,
      "table": 0,
      "row": 1,
      "cell": 0
    }
  ]
}
```

| 结果字段 | 类型 | 说明 |
|--------------|------|-------------|
| `placeholders` | Array\<String\> | 唯一的占位符名（首次出现顺序） |
| `count` | Number | 唯一占位符数量 |
| `occurrences` | Number | 占位符实例总数 |
| `details` | Array | 每次出现的位置详情 |

**位置类型：** `paragraph`、`tableCell`、`header`、`footer`

**示例：**

```json
{
  "id": 11,
  "method": "template.inspect",
  "params": {
    "documentId": "doc-1"
  }
}
```

---

### template.render

用提供的数据值替换文档中的模板占位符。可选校验所有占位符都被覆盖（严格模式）。

**参数：**

| 字段 | 类型 | 必填 | 说明 |
|-------|------|----------|-------------|
| `documentId` | String | 是 | 会话文档 ID |
| `data` | Object | 是 | 替换的键值映射（`{ "Name": "Alice" }`） |
| `strictMode` | Boolean | 否 | 为 `true` 时，缺失键会失败（默认 `false`） |
| `openDelimiter` | String | 否 | 自定义起始分隔符（默认 `"{{"`） |
| `closeDelimiter` | String | 否 | 自定义结束分隔符（默认 `"}}"`） |

**成功结果：**

```json
{ "ok": true }
```

带校验警告时：

```json
{
  "ok": true,
  "warnings": [
    { "severity": "warning", "key": "OptionalField", "message": "key OptionalField not found in data" }
  ]
}
```

**错误（严格模式，缺失键）：**

```json
{
  "code": "TEMPLATE_ERROR",
  "message": "template: missing keys: Code",
  "operation": "template.render",
  "data": { "category": "merge", "retryable": false }
}
```

**示例：**

```json
{
  "id": 12,
  "method": "template.render",
  "params": {
    "documentId": "doc-1",
    "data": {
      "Name": "Alice Johnson",
      "Company": "Acme Corp",
      "Date": "2025-01-15"
    },
    "strictMode": true
  }
}
```

---

### document.close

从会话中移除文档，释放关联的内存。在 RPC 模式下文档不再需要时应当调用。

**参数：**

| 字段 | 类型 | 必填 | 说明 |
|-------|------|----------|-------------|
| `documentId` | String | 是 | 会话文档 ID |

**成功结果：** `{ "ok": true }`

---

## 内容类型

内容项在 `document.create` 与 `document.addContent` 参数中以数组形式传递。每个项有 `type` 字段。

### 段落

```json
{
  "type": "paragraph",
  "style": "Heading1",
  "alignment": "center",
  "spacingBefore": 240,
  "spacingAfter": 120,
  "lineSpacing": { "rule": "auto", "value": 360 },
  "indent": { "left": 720, "right": 0, "firstLine": 0, "hanging": 0 },
  "numbering": { "id": 1, "level": 0 },
  "borders": {
    "bottom": { "style": "single", "width": 6, "color": "#000000" }
  },
  "runs": [ ... ]
}
```

**样式值：** `Normal`、`Heading1`–`Heading9`、`Title`、`Subtitle`、`Quote`、`IntenseQuote`、`ListParagraph`、`Caption`、`BodyText`、`NoSpacing` 等。

**对齐值：** `left`、`center`、`right`、`justify`、`distribute`

**行距规则：** `auto`、`exact`、`atLeast`

**Run 字段：**

| 字段 | 类型 | 说明 |
|-------|------|-------------|
| `text` | String | 文本内容 |
| `bold` | Boolean | 加粗 |
| `italic` | Boolean | 斜体 |
| `strike` | Boolean | 删除线 |
| `underline` | `"single"` \| `"double"` \| `"thick"` \| `"dotted"` \| `"dashed"` \| `"wave"` \| `"none"` | 下划线样式 |
| `color` | String | 文本颜色（十六进制） |
| `fontSize` | Number | 字号（**磅**） |
| `font` | String | 字体名（如 `"Arial"`） |
| `highlight` | String | 高亮色（见下文） |
| `hyperlink` | `{ "url": "...", "displayText": "..." }` | 行内超链接 |
| `field` | Object | 文档域（见下文） |
| `image` | Object | 图片（见下文） |
| `break` | `"page"` \| `"column"` \| `"line"` | 插入分隔符 |

**高亮色：** `yellow`、`green`、`cyan`、`magenta`、`blue`、`red`、`darkBlue`、`darkCyan`、`darkGreen`、`darkMagenta`、`darkRed`、`darkYellow`、`darkGray`、`lightGray`、`none`

**域对象：**

```json
{
  "type": "pageNumber"
}
```

域类型：`pageNumber`、`pageCount`、`toc`、`hyperlink`、`styleRef`

带选项的目录域：
```json
{
  "type": "toc",
  "options": { "levels": "1-3", "hyperlinks": "true" }
}
```

超链接域：
```json
{
  "type": "hyperlink",
  "url": "https://example.com",
  "display": "Click here"
}
```

样式引用域（用于页眉）：
```json
{
  "type": "styleRef",
  "style": "Heading 1"
}
```

`url`、`style` 以及 `options.levels`（目录域的标题范围开关）如果包含双引号会被拒绝——它们被嵌入所生成域代码的带引号参数中，否则字面的 `"` 会跳出该参数。`url`、`style`、`options.levels` 如果包含双引号会被拒绝——它们被嵌入所生成域代码的带引号参数中，否则字面的 `"` 会跳出该参数。请求会因校验错误而失败，而非产生损坏或可被利用的域。

**图片对象：**

```json
{
  "path": "/path/to/image.png",
  "widthPx": 400,
  "heightPx": 300
}
```

或使用 base64 数据：
```json
{
  "base64": "<base64-encoded image data>",
  "format": "png",
  "widthPx": 400,
  "heightPx": 300
}
```

---

### 表格

```json
{
  "type": "table",
  "style": "TableGrid",
  "alignment": "left",
  "width": { "type": "dxa", "value": 9000 },
  "rows": [
    {
      "height": 400,
      "cells": [
        {
          "width": 3000,
          "verticalAlignment": "top",
          "shading": "#F0F0F0",
          "gridSpan": 2,
          "borders": {
            "top": { "style": "single", "width": 6, "color": "#000000" }
          },
          "paragraphs": [
            {
              "type": "paragraph",
              "runs": [{ "text": "Cell content", "bold": true }]
            }
          ]
        }
      ]
    }
  ]
}
```

**表格样式：** `TableNormal`、`TableGrid`、`PlainTable1`、`MediumShading1`、`LightShading`、`ColorfulList`

**宽度类型：** `auto`、`dxa`（twips）、`pct`（百分比 × 50）

**垂直对齐：** `top`、`center`、`bottom`

**边框样式：** `single`、`dotted`、`dashed`、`double`、`triple`、`thick`、`none`

---

### 节

```json
{
  "type": "section",
  "breakType": "nextPage",
  "pageSize": "A4",
  "margins": { "top": 1440, "bottom": 1440, "left": 1440, "right": 1440 },
  "orientation": "portrait",
  "columns": 2,
  "headers": {
    "default": [
      {
        "type": "paragraph",
        "alignment": "right",
        "runs": [{ "text": "My Document" }]
      }
    ]
  },
  "footers": {
    "default": [
      {
        "type": "paragraph",
        "alignment": "center",
        "runs": [{ "field": { "type": "pageNumber" } }]
      }
    ]
  }
}
```

**分隔符类型：** `nextPage`（默认）、`continuous`、`evenPage`、`oddPage`

**页眉/页脚键：** `default`、`first`、`even`

---

### 分页符

```json
{ "type": "pageBreak" }
```

---

## Shell 示例

### 创建一个简单文档

```bash
echo '{
  "id": 1,
  "method": "document.create",
  "params": {
    "options": { "title": "Hello World" },
    "content": [
      {
        "type": "paragraph",
        "runs": [{ "text": "Hello, World!", "bold": true }]
      }
    ],
    "output": "file",
    "filePath": "/tmp/hello.docx"
  }
}' | docxgo exec
```

### 检查已有文档

```bash
echo '{
  "id": 1,
  "method": "document.open",
  "params": { "filePath": "/path/to/existing.docx" }
}' | docxgo exec
# → {"id":1,"result":{"documentId":"doc-1"}}
```

### 获取文档信息（exec 模式下两步，或 RPC 下单会话）

多步工作流请使用 RPC 模式：

```bash
printf '{"id":1,"method":"document.open","params":{"filePath":"/path/to/doc.docx"}}\n{"id":2,"method":"document.inspect","params":{"documentId":"doc-1"}}\n' \
  | docxgo rpc 2>/dev/null
```

---

## Node.js 集成

### 一次性模式（exec）

```javascript
const { execFile } = require('child_process');
const path = require('path');

function createDocument(params) {
  return new Promise((resolve, reject) => {
    const request = JSON.stringify({ id: 1, method: 'document.create', params });
    execFile('docxgo', ['exec', '--request', request], (err, stdout, stderr) => {
      if (err) return reject(err);
      const response = JSON.parse(stdout);
      if (response.error) return reject(new Error(response.error.message));
      resolve(response.result);
    });
  });
}

// 用法：
createDocument({
  options: { title: 'My Doc', pageSize: 'A4' },
  content: [
    { type: 'paragraph', runs: [{ text: 'Hello!', bold: true }] }
  ],
  output: 'buffer'
}).then(result => {
  const buf = Buffer.from(result.data, 'base64');
  require('fs').writeFileSync('output.docx', buf);
  console.log('Created:', buf.length, 'bytes');
});
```

### RPC 模式（spawn）

```javascript
const { spawn } = require('child_process');
const readline = require('readline');

class DocxgoRPC {
  constructor(binaryPath = 'docxgo') {
    this._proc = spawn(binaryPath, ['rpc']);
    this._rl = readline.createInterface({ input: this._proc.stdout });
    this._pending = new Map();
    this._rl.on('line', line => {
      const resp = JSON.parse(line);
      const resolve = this._pending.get(resp.id);
      if (resolve) {
        this._pending.delete(resp.id);
        resolve(resp);
      }
    });
    this._seq = 0;
  }

  call(method, params) {
    return new Promise((resolve, reject) => {
      const id = ++this._seq;
      this._pending.set(id, resp => {
        if (resp.error) reject(new Error(resp.error.message));
        else resolve(resp.result);
      });
      this._proc.stdin.write(JSON.stringify({ id, method, params }) + '\n');
    });
  }

  close() {
    this._proc.stdin.end();
  }
}

// 用法：
async function main() {
  const rpc = new DocxgoRPC();

  const result = await rpc.call('document.create', {
    options: { title: 'Batch Document', pageSize: 'Letter' },
    content: [
      { type: 'paragraph', style: 'Heading1', runs: [{ text: 'Report' }] },
      { type: 'paragraph', runs: [{ text: 'Generated via RPC.' }] }
    ],
    output: 'buffer'
  });

  const buf = Buffer.from(result.data, 'base64');
  require('fs').writeFileSync('report.docx', buf);
  console.log('Saved report.docx:', buf.length, 'bytes');

  rpc.close();
}

main().catch(console.error);
```

---

## 错误码参考

| 错误码 | 说明 |
|------|-------------|
| `VALIDATION_ERROR` | 输入参数或文档结构无效 |
| `NOT_FOUND` | 会话中引用的文档 ID 不存在 |
| `IO_ERROR` | 文件读取/写入失败 |
| `INTERNAL_ERROR` | 意外的内部错误 |
| `METHOD_NOT_FOUND` | 未知的 RPC 方法 |
| `PARSE_ERROR` | 请求中 JSON 格式错误 |
| `TEMPLATE_ERROR` | 模板合并/校验失败 |
