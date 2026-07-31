# DOCX 校验错误排查

本文档描述了 docxgo v2 开发过程中遇到的 OOXML 校验问题，以及所实施的解决方案。它可作为未来诊断和解决类似问题的指南。

## 目录

1. [诊断工具](#诊断工具)
2. [问题 1：非法的超链接 RelationshipID](#问题-1非法的超链接-relationshipid)
3. [问题 2：空的 wp:align 值](#问题-2空的-wpalign-值)
4. [已知且可容忍的错误](#已知且可容忍的错误)
5. [通用诊断工作流](#通用诊断工作流)

---

## 诊断工具

### DocxValidator（C#）

位置：`DocxValidator/`

```powershell
cd DocxValidator
dotnet run -- "path/to/document.docx"
```

该校验器使用 OpenXML SDK 检测架构错误，提供：
- 错误总数
- 每条错误的描述
- 受影响的文档部件（document.xml、header1.xml 等）
- 问题元素的精确 XPath

### 手动解包 DOCX

`.docx` 文件是一个 ZIP 压缩包。要检查内部 XML：

```powershell
# 解压内容
Expand-Archive -Path "document.docx" -DestinationPath "document_debug" -Force

# 搜索特定模式
Select-String -Path "document_debug\word\document.xml" -Pattern "wp:positionV" -Context 2,2
```

### 文档对比

要识别往返过程中发生的变化：

```powershell
# 解压原始文件与生成文件
Expand-Archive -Path "original.docx" -DestinationPath "original_debug" -Force
Expand-Archive -Path "generated.docx" -DestinationPath "generated_debug" -Force

# 对比特定文件
Compare-Object (Get-Content "original_debug\word\document.xml") `
               (Get-Content "generated_debug\word\document.xml")
```

---

## 问题 1：非法的超链接 RelationshipID

### 症状

```
The relationship 'rId37' was not found in 'word/document.xml'.
```

Word 无法打开文档，因为它引用了不存在的关系 ID。

### 根本原因

读取已有文档时，外部超链接会保留其原始的 `relationshipID`（例如 `rId25`）。但在序列化时，`run.go` 中的 `AddField()` 方法**总是**调用 `AddHyperlink()`，这会生成一个**新的**关系 ID（例如 `rId37`），从而覆盖了被保留的那个。

这导致了：
- XML 引用了 `rId37`
- 但 `document.xml.rels` 中只有原始的 `rId25`

### 诊断

1. 在生成文档中搜索超链接 ID：
   ```powershell
   Select-String -Path "generated_debug\word\document.xml" -Pattern 'w:hyperlink r:id="rId'
   ```

2. 检查实际存在的关系：
   ```powershell
   Get-Content "generated_debug\word\_rels\document.xml.rels"
   ```

3. 如果 ID 不匹配，问题出在序列化环节。

### 解决方案

**文件**：`internal/core/run.go` - `AddField()` 方法

在生成新关系之前，先检查是否已存在被保留的关系：

```go
// Check if this hyperlink already has a relationship ID (preserved from read)
// If so, skip creating a new one to preserve original document references
existingRelID, hasExistingRelID := accessor.GetProperty("relationshipID")
if hasExistingRelID && existingRelID != "" {
    // Already has a relationship ID, skip creating new one
} else {
    // No existing relationship ID, create a new one
    if r.relManager == nil {
        return errors.InvalidState("Run.AddField", "hyperlink relationship manager not initialized")
    }

    url, ok := accessor.GetProperty("url")
    if !ok || url == "" {
        return errors.InvalidArgument("Run.AddField", "url", url, "hyperlink URL cannot be empty")
    }

    relID, err := r.relManager.AddHyperlink(url)
    if err != nil {
        return errors.Wrap(err, "Run.AddField")
    }

    accessor.SetProperty("relationshipID", relID)
}
```

### 相关文件

- `internal/reader/reconstruct.go` - `hydrateHyperlink()`：通过 `SetProperty` 保留 `originalRelID`
- `internal/serializer/serializer.go` - `expandRunWithFields()`：使用被保留的 relationshipID
- `internal/core/run.go` - `AddField()`：曾在此处生成重复 ID

---

## 问题 2：空的 wp:align 值

### 症状

```
The element 'wp:align' has invalid value ''
Part: document.xml
Path: /w:document[1]/w:body[1]/w:p[4]/w:r[1]/w:drawing[1]/wp:anchor[1]/wp:positionV[1]/wp:align[1]
```

### 根本原因

在 OOXML 中，`wp:positionH` 与 `wp:positionV` 元素可以包含：
- `<wp:posOffset>` - 以 EMU 为单位的数值偏移（可以为 0）
- `<wp:align>` - 命名对齐（"left"、"center"、"top" 等）

**但二者只能出现其一，绝不同时出现，也绝不能空。**

问题发生在以下情形：
1. 原始文档有 `<wp:posOffset>0</wp:posOffset>`（显式的 0 偏移）
2. 读取时 `pos.OffsetY = 0`（正好是 Go 的默认值）
3. 序列化时条件 `if pos.OffsetY != 0` 为 FALSE
4. 于是落入 `else` 分支，尝试使用 `pos.VAlign`（此时为空）
5. 结果：`<wp:align></wp:align>` - **非法**

### 诊断

1. 校验原始文档：
   ```powershell
   cd DocxValidator
   dotnet run -- "original.docx"
   ```

2. 如果原始文档没有该错误而生成的文档有，问题出在往返过程。

3. 对比 positionV 的 XML 结构：
   ```powershell
   Select-String -Path "original_debug\word\document.xml" -Pattern "wp:positionV" -Context 0,3
   Select-String -Path "generated_debug\word\document.xml" -Pattern "wp:positionV" -Context 0,3
   ```

### 解决方案

需要两处改动：

#### 1. 为 ImagePosition 增加标志位

**文件**：`domain/image.go`

```go
type ImagePosition struct {
    Type       ImagePositionType
    HAlign     HorizontalAlign
    VAlign     VerticalAlign
    OffsetX    int
    OffsetY    int
    UseOffsetX bool  // 当应使用 OffsetX 时为真（即使为 0）
    UseOffsetY bool  // 当应使用 OffsetY 时为真（即使为 0）
    WrapText   TextWrapType
    ZOrder     int
    BehindText bool
}
```

#### 2. 读取时设置标志位

**文件**：`internal/reader/reconstruct.go` - `buildFloatingPosition()`

```go
if positionH := findChild(elem, "positionH"); positionH != nil {
    if align := findChild(positionH, "align"); align != nil {
        if mapped, ok := mapHorizontalAlignValue(strings.TrimSpace(align.Text)); ok {
            pos.HAlign = mapped
        }
    }
    if offset, ok := parseChildInt(positionH, "posOffset"); ok {
        pos.OffsetX = offset
        pos.UseOffsetX = true  // ← 标记应当使用偏移
    }
}

if positionV := findChild(elem, "positionV"); positionV != nil {
    if align := findChild(positionV, "align"); align != nil {
        if mapped, ok := mapVerticalAlignValue(strings.TrimSpace(align.Text)); ok {
            pos.VAlign = mapped
        }
    }
    if offset, ok := parseChildInt(positionV, "posOffset"); ok {
        pos.OffsetY = offset
        pos.UseOffsetY = true  // ← 标记应当使用偏移
    }
}
```

#### 3. 序列化时使用标志位

**文件**：`internal/xml/drawing_helper.go` - `NewFloatingDrawing()`

```go
// 设置水平位置
anchor.PositionH = &PositionH{
    RelativeFrom: convertHAlign(pos.HAlign),
}
// 显式设置了偏移（即使为 0）则使用偏移，否则使用对齐（若提供）
if pos.UseOffsetX || pos.OffsetX != 0 {
    offset := pos.OffsetX
    anchor.PositionH.PosOffset = &offset
} else if pos.HAlign != "" {
    align := string(pos.HAlign)
    anchor.PositionH.Align = &align
}

// 设置垂直位置
anchor.PositionV = &PositionV{
    RelativeFrom: convertVAlign(pos.VAlign),
}
// 显式设置了偏移（即使为 0）则使用偏移，否则使用对齐（若提供）
if pos.UseOffsetY || pos.OffsetY != 0 {
    offset := pos.OffsetY
    anchor.PositionV.PosOffset = &offset
} else if pos.VAlign != "" {
    align := string(pos.VAlign)
    anchor.PositionV.Align = &align
}
```

### 经验教训

> **在 Go 中，零值（`0`、`""`、`false`）与"未设置"无法区分。**
>
> 当数值 0 在语义上不同于"无值"时，需要额外的布尔标志来跟踪该值是否曾被显式设置。

---

## 已知且可容忍的错误

### 页眉中的 tblLook（通常 60 个错误）

```
The 'http://schemas.openxmlformats.org/wordprocessingml/2006/main:firstRow' attribute is not declared.
Part: /word/headerX.xml
```

这些错误来自 Word 创建的原始文档，且被 **Word 本身所容忍**。它们是 Word 2010+ 的属性，而 OpenXML 校验器因为使用 Word 2007 的架构而无法识别。

**处理**：如果原始文档中本就存在这些错误，可以忽略。

---

## 通用诊断工作流

```
┌──────────────────────────────────────┐
│ 1. 校验原始文档                        │
│    DocxValidator original.docx       │
└──────────────────┬───────────────────┘
                   ↓
┌──────────────────────────────────────┐
│ 2. 校验生成文档                        │
│    DocxValidator generated.docx      │
└──────────────────┬───────────────────┘
                   ↓
┌──────────────────────────────────────┐
│ 3. 对比错误                            │
│    是否出现了新的错误？                │
└──────────────────┬───────────────────┘
                   ↓
        ┌─────────┴─────────┐
        │ 是                │ 否
        ↓                   ↓
┌───────────────────┐  ┌───────────────────┐
│ 4. 解压两个        │  │ 错误来自原文档，   │
│    文档            │  │ 忽略              │
└────────┬──────────┘  └───────────────────┘
         ↓
┌──────────────────────────────────────┐
│ 5. 在两个 XML 中搜索错误 XPath         │
└──────────────────┬───────────────────┘
         ↓
┌──────────────────────────────────────┐
│ 6. 定位差异                            │
│    往返过程中什么被改动了？            │
└──────────────────┬───────────────────┘
         ↓
┌──────────────────────────────────────┐
│ 7. 在代码中定位                        │
│    grep_search 该元素                 │
└──────────────────┬───────────────────┘
         ↓
┌──────────────────────────────────────┐
│ 8. 追踪 读取 → 序列化 流程            │
│    数值在哪里丢失/被改？              │
└──────────────────────────────────────┘
```

---

## 发布前校验检查清单

- [ ] 对所有示例运行 DocxValidator
- [ ] 将错误与原始文档对比
- [ ] 确认未引入新的错误
- [ ] 测试在 Word 中打开
- [ ] 运行全部单元测试

---

## 参考资料

- [ECMA-376 Office Open XML](https://www.ecma-international.org/publications-and-standards/standards/ecma-376/)
- [DrawingML Positioning](https://docs.microsoft.com/en-us/dotnet/api/documentformat.openxml.drawing.wordprocessing)
- [Open XML SDK validation](https://learn.microsoft.com/en-us/office/open-xml/word/how-to-validate-a-word-processing-document)

---

*文档创建：2026 年 1 月*
*最后更新：2026 年 1 月*
*作者：mmonterroca*
