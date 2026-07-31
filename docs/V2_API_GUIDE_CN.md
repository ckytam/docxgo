# docxgo v2 API 指南

本文档是 docxgo v2 公共 API 的综合参考。它涵盖了构建器模式 API 与直接调用 domain 接口两种用法，并附使用示例。

## 目录

- [安装](#安装)
- [快速入门](#快速入门)
- [构建器模式（推荐）](#构建器模式推荐)
  - [DocumentBuilder](#documentbuilder)
  - [ParagraphBuilder](#paragraphbuilder)
  - [TableBuilder](#tablebuilder)
- [直接调用 Domain API](#直接调用-domain-api)
- [段落与文本](#段落与文本)
  - [格式化属性](#格式化属性)
  - [Run 格式化](#run-格式化)
- [表格](#表格)
  - [合并单元格](#合并单元格)
  - [单元格样式](#单元格样式)
  - [嵌套表格](#嵌套表格)
- [图片](#图片)
  - [内联图片](#内联图片)
  - [浮动图片](#浮动图片)
  - [图片尺寸](#图片尺寸)
- [域（Fields）](#域fields)
  - [域类型](#域类型)
  - [超链接域](#超链接域)
  - [Run 上的域](#run-上的域)
- [节与页面布局](#节与页面布局)
  - [默认节](#默认节)
  - [页眉与页脚](#页眉与页脚)
  - [多节](#多节)
  - [分栏](#分栏)
- [样式](#样式)
- [主题](#主题)
- [模板 / 邮件合并](#模板--邮件合并)
- [读取与修改已有文档](#读取与修改已有文档)
- [错误处理](#错误处理)
- [最佳实践](#最佳实践)

---

## 安装

```bash
go get github.com/mmonterroca/docxgo/v2
```

```go
import (
    docx "github.com/mmonterroca/docxgo/v2"
    "github.com/mmonterroca/docxgo/v2/domain"
)
```

**要求**：Go 1.23+

---

## 快速入门

```go
package main

import (
    "fmt"
    "log"

    docx "github.com/mmonterroca/docxgo/v2"
)

func main() {
    // 创建一个新文档
    doc, err := docx.NewDocument(
        docx.WithTitle("My First Document"),
        docx.WithAuthor("Jane Doe"),
    )
    if err != nil {
        log.Fatal(err)
    }

    // 添加标题段落
    h1, err := doc.AddParagraph()
    if err != nil {
        log.Fatal(err)
    }
    h1.SetStyle(domain.StyleIDHeading1)
    h1.SetAlignment(domain.AlignCenter)
    if _, err := h1.AddText("Hello, docxgo!"); err != nil {
        log.Fatal(err)
    }

    // 添加正文段落
    p, err := doc.AddParagraph()
    if err != nil {
        log.Fatal(err)
    }
    if _, err := p.AddText("This document was created with docxgo v2."); err != nil {
        log.Fatal(err)
    }

    // 保存
    if err := doc.SaveAs("hello.docx"); err != nil {
        log.Fatal(err)
    }
    fmt.Println("Document saved!")
}
```

---

## 构建器模式（推荐）

构建器提供流式 API，并具备错误累积能力，是创建文档最便捷的方式。

### DocumentBuilder

```go
builder := docx.NewDocumentBuilder(
    docx.WithTitle("Report"),
    docx.WithAuthor("Jane Doe"),
)

doc, err := builder.
    AddHeading("Executive Summary", 1).
    AddParagraph("This is the summary.").
    AddHeading("Details", 2).
    AddParagraph("Detailed content here.").
    AddTable(3, 2, func(row, col int) string {
        return fmt.Sprintf("R%dC%d", row+1, col+1)
    }).
    Build()

if err != nil {
    log.Fatal(err)
}
```

**可用的构建器方法：**
- `AddHeading(text, level)`
- `AddParagraph(text)`
- `AddRichParagraph(runs)`
- `AddParagraphWithStyle(text, styleID)`
- `AddTable(rows, cols, fill)`
- `AddStyledTable(rows, cols, style, fill)`
- `AddBulletList(items)`
- `AddPageBreak()`
- `AddImage(path)`
- `AddImageWithSize(path, w, h)`
- `Build()` —— 返回 `(Document, error)`

### ParagraphBuilder

```go
p, _ := doc.AddParagraph()
pb := docx.NewParagraphBuilder(p)
pb.
    SetStyle(domain.StyleIDHeading1).
    SetAlignment(domain.AlignCenter).
    AddRun("Bold ", docx.RunBold()).
    AddRun("and ", docx.RunItalic()).
    AddRun("colored", docx.RunColor(domain.RGB(255, 0, 0)))
```

**Run 选项：**
- `docx.RunBold()`
- `docx.RunItalic()`
- `docx.RunUnderline(style)`
- `docx.RunColor(color)`
- `docx.RunFontSize(halfPts)`
- `docx.RunFont(name)`

### TableBuilder

```go
table, err := doc.AddTable(3, 2)
if err != nil {
    log.Fatal(err)
}

tb := docx.NewTableBuilder(table)
tb.
    SetStyle(domain.TableStyleGrid).
    SetAlignment(domain.AlignCenter).
    SetWidth(9000, domain.TableWidthDXA)

tb.Cell(0, 0).SetText("Name").Bold()
tb.Cell(0, 1).SetText("Score").Bold()
tb.Cell(1, 0).SetText("Alice")
tb.Cell(1, 1).SetText("95")
```

---

## 直接调用 Domain API

对于精细控制，可直接使用 `domain` 接口：

```go
doc, _ := docx.NewDocument()

// 添加段落
para, _ := doc.AddParagraph()

// 添加 run
run, _ := para.AddRun()
run.SetText("Hello")
run.SetBold(true)

// 应用样式
para.SetStyle(domain.StyleIDHeading1)
para.SetAlignment(domain.AlignLeft)
```

---

## 段落与文本

### 格式化属性

```go
p, _ := doc.AddParagraph()

// 对齐
p.SetAlignment(domain.AlignCenter)

// 间距（半磅单位）
p.SetSpacingBefore(240)  // 12 磅
p.SetSpacingAfter(120)   // 6 磅

// 行距
p.SetLineSpacing(domain.LineSpacingAuto, 360)  // 1.5 倍

// 缩进（twips，1/1440 英寸）
p.SetIndent(720, 0, 0, 0)  // 左缩进 0.5 英寸

// 样式
p.SetStyle(domain.StyleIDHeading1)
```

**对齐常量：**
- `domain.AlignLeft`
- `domain.AlignCenter`
- `domain.AlignRight`
- `domain.AlignJustify`
- `domain.AlignDistribute`

### Run 格式化

```go
run, _ := para.AddRun()

run.SetText("Formatted text")
run.SetBold(true)
run.SetItalic(true)
run.SetUnderline(domain.UnderlineSingle)
run.SetColor(domain.RGB(0, 122, 204))
run.SetFontSize(24)        // 12 磅（半磅）
run.SetFont("Arial")
run.SetStrike(true)
run.SetSubscript(true)     // 或 SetSuperscript(true)
```

**下划线样式：** `UnderlineSingle`、`UnderlineDouble`、`UnderlineThick`、`UnderlineDotted`、`UnderlineDashed`、`UnderlineWave`、`UnderlineNone`

---

## 表格

```go
table, err := doc.AddTable(3, 2)
if err != nil {
    log.Fatal(err)
}

// 访问单元格
cell, err := table.Cell(0, 0)
if err != nil {
    log.Fatal(err)
}

// 设置单元格文本
para, _ := cell.AddParagraph()
run, _ := para.AddRun()
run.SetText("Cell content")

// 或快捷设置
cell.SetText("Quick text")
```

### 合并单元格

```go
// 水平合并（跨列）
cell.SetGridSpan(2)  // 横跨 2 列

// 垂直合并（跨行）
cell.SetVMerge(true)  // 与上方单元格合并
```

### 单元格样式

```go
cell, _ := table.Cell(0, 0)

// 宽度（twips）
cell.SetWidth(3000, domain.TableWidthDXA)

// 垂直对齐
cell.SetVerticalAlignment(domain.CellAlignCenter)

// 背景色（底纹）
cell.SetShading(domain.RGB(240, 240, 240))

// 边框
cell.SetBorders(domain.CellBorder{
    Top: domain.BorderStyle{
        Style: domain.BorderSingle,
        Width: 4,
        Color: domain.RGB(0, 0, 0),
    },
})
```

### 嵌套表格

```go
cell, _ := table.Cell(1, 1)
nested, _ := cell.AddTable(2, 2)
nested.Cell(0, 0).SetText("Nested")
```

---

## 图片

### 内联图片

```go
img, err := doc.AddImage("photo.png")
if err != nil {
    log.Fatal(err)
}
```

### 浮动图片

```go
img, err := doc.AddImageWithPosition("logo.png", domain.ImagePosition{
    Type:     domain.ImageFloat,
    HAlign:   domain.HAlignCenter,
    VAlign:   domain.VAlignTop,
    OffsetX:  0,
    OffsetY:  0,
    WrapText: domain.WrapSquare,
    ZOrder:   1,
})
if err != nil {
    log.Fatal(err)
}
```

### 图片尺寸

```go
// 指定尺寸（像素）
img, err := doc.AddImageWithSize("chart.png", 400, 300)
if err != nil {
    log.Fatal(err)
}
```

**支持的格式：** PNG、JPEG、GIF

---

## 域（Fields）

```go
run, _ := para.AddRun()
field := domain.Field{
    Type: domain.FieldPageNumber,
}
run.AddField(field)
```

### 域类型

| 类型 | 说明 |
|------|------|
| `FieldPageNumber` | 当前页码 |
| `FieldNumPages` | 总页数 |
| `FieldTOC` | 目录 |
| `FieldHyperlink` | 超链接 |
| `FieldStyleRef` | 样式引用 |
| `FieldSeq` | 序号 |
| `FieldRef` | 书签引用 |
| `FieldPageRef` | 页码引用 |
| `FieldDate` | 日期域 |

### 超链接域

```go
run, _ := para.AddRun()
field := domain.Field{
    Type:  domain.FieldHyperlink,
    URL:   "https://github.com/mmonterroca/docxgo",
    Text:  "docxgo",
}
run.AddField(field)
```

### Run 上的域

```go
run, _ := para.AddRun()
field := domain.Field{
    Type: domain.FieldTOC,
    Options: map[string]string{
        "levels":  "1-3",
        "hyperlinks": "true",
    },
}
run.AddField(field)
```

---

## 节与页面布局

### 默认节

```go
section := doc.DefaultSection()

// 页面尺寸
section.SetPageSize(domain.PageSizeA4)

// 方向
section.SetOrientation(domain.OrientationLandscape)

// 边距
section.SetMargins(domain.MarginNormal)
```

**页面尺寸：** `PageSizeA4`、`PageSizeLetter`、`PageSizeLegal`、`PageSizeA3`、`PageSizeTabloid`

### 页眉与页脚

```go
section := doc.DefaultSection()

// 默认页眉
header, _ := section.GetHeader(domain.HeaderDefault)
headerPara, _ := header.AddParagraph()
headerPara.AddText("Document Title")

// 默认页脚（带页码）
footer, _ := section.GetFooter(domain.FooterDefault)
footerPara, _ := footer.AddParagraph()
footerPara.SetAlignment(domain.AlignCenter)
footerRun, _ := footerPara.AddRun()
footerRun.AddField(domain.Field{Type: domain.FieldPageNumber})
```

**页眉/页脚类型：**
- `HeaderDefault` / `FooterDefault`
- `HeaderFirst` / `FooterFirst`
- `HeaderEven` / `FooterEven`

### 多节

```go
// 添加带分节符的新节
newSection, err := doc.AddSectionWithBreak(domain.SectionBreakNextPage)
if err != nil {
    log.Fatal(err)
}

newSection.SetPageSize(domain.PageSizeA4)
newSection.SetOrientation(domain.OrientationPortrait)
```

### 分栏

```go
section := doc.DefaultSection()
section.SetColumns(2)  // 两栏布局
```

---

## 样式

docxgo 内置 40+ 段落样式：

```go
p, _ := doc.AddParagraph()
p.SetStyle(domain.StyleIDHeading1)  // 标题 1
p.SetStyle(domain.StyleIDQuote)     // 引用
p.SetStyle(domain.StyleIDListParagraph)  // 列表段落
```

**常用样式 ID：**
- `StyleIDNormal`
- `StyleIDHeading1` 到 `StyleIDHeading9`
- `StyleIDTitle`、`StyleIDSubtitle`
- `StyleIDQuote`、`StyleIDIntenseQuote`
- `StyleIDListParagraph`、`StyleIDListBullet`、`StyleIDListNumber`
- `StyleIDCaption`

---

## 主题

```go
builder := docx.NewDocumentBuilder(
    docx.WithTheme(domain.ThemeCorporate),
)

// 或应用自定义主题
customTheme := domain.Theme{
    Colors: domain.ThemeColors{
        Primary:   domain.RGB(0, 122, 204),
        Secondary: domain.RGB(68, 114, 196),
        Accent1:   domain.RGB(255, 192, 0),
    },
    Fonts: domain.ThemeFonts{
        Heading: "Calibri",
        Body:    "Calibri",
    },
}
```

**预设主题：** `ThemeCorporate`、`ThemeStartup`、`ThemeModern`、`ThemeFintech`、`ThemeAcademic`、`ThemeTechPresentation`、`ThemeTechDarkMode`

---

## 模板 / 邮件合并

```go
import "github.com/mmonterroca/docxgo/v2/pkg/template"

doc, _ := docx.OpenDocument("template.docx")

data := template.MergeData{
    "customer_name": "Acme Corp",
    "invoice_total": "$1,234.56",
    "date":          "2025-01-15",
}

missing, err := template.MergeTemplate(doc, data)
if err != nil {
    log.Fatal(err)
}

// 校验所有占位符都被替换
if len(missing) > 0 {
    log.Printf("Missing keys: %v", missing)
}

doc.SaveAs("filled.docx")
```

**占位符语法：** `{{key}}`

**循环（表格行）：**

```go
loops := template.ForeachLoops{
    "items": []template.ForeachItem{
        {"name": "Widget", "price": "$10"},
        {"name": "Gadget", "price": "$20"},
    },
}
template.MergeTemplateWithLoops(doc, data, loops)
```

---

## 读取与修改已有文档

```go
doc, err := docx.OpenDocument("existing.docx")
if err != nil {
    log.Fatal(err)
}

// 添加内容
p, _ := doc.AddParagraph()
p.AddText("Appended paragraph")

// 修改样式
for i := 0; i < doc.ParagraphCount(); i++ {
    para, _ := doc.Paragraph(i)
    // 检查并修改
}

doc.SaveAs("modified.docx")
```

**注意：**
- 保留原始样式与格式
- 超链接关系在往返中保留
- 自定义样式不丢失
- 图片正常保留

---

## 错误处理

docxgo 使用结构化的错误类型：

```go
doc, err := docx.OpenDocument("missing.docx")
if err != nil {
    var docxErr *errors.DocxError
    if errors.As(err, &docxErr) {
        fmt.Printf("操作: %s, 错误码: %s\n", docxErr.Op, docxErr.Code)
    }
}
```

**常见错误码：**
- `VALIDATION_ERROR` —— 非法输入或文档结构
- `NOT_FOUND` —— 资源不存在
- `INVALID_STATE` —— 对象处于非法状态
- `IO_ERROR` —— 文件读写失败
- `INTERNAL_ERROR` —— 库内部错误

---

## 最佳实践

1. **使用构建器** —— 对于大多数文档生成场景，构建器模式配合错误累积更便捷
2. **检查错误** —— 直接 API 返回 `(T, error)`，务必处理
3. **样式优先** —— 优先使用内置样式 ID 而非手动格式化
4. **复用文档对象** —— 在批量生成中复用 `Document` 实例（需注意并发安全）
5. **保存前校验** —— 对生成的关键文档使用 `doc.Validate()`
6. **图片格式** —— 偏好 PNG（无损）；大图考虑 JPEG
7. **模板校验** —— 邮件合并后用 `missing` 列表校验占位符覆盖

---

更多示例见 `examples/` 目录。

---

*文档创建：2025 年 10 月*
*最后更新：2026 年 7 月*
*作者：mmonterroca*
