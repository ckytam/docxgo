# 段落边框功能

## 概述

新增了对段落边框（上、下、左、右）的支持，用于在段落周围添加装饰线或边框框。

## 新增 API 方法

### Paragraph 接口

```go
// 获取所有边框
Borders() ParagraphBorders

// 一次性设置所有边框
SetBorders(borders ParagraphBorders) error

// 设置单个边框
SetBorderTop(border BorderStyle) error
SetBorderBottom(border BorderStyle) error
SetBorderLeft(border BorderStyle) error
SetBorderRight(border BorderStyle) error
```

### 类型

```go
// ParagraphBorders 表示段落的边框
type ParagraphBorders struct {
    Top    BorderStyle
    Bottom BorderStyle
    Left   BorderStyle
    Right  BorderStyle
}

// BorderStyle（表格中已存在）
type BorderStyle struct {
    Style BorderLineStyle
    Width int   // 宽度，以八分之一磅为单位
    Color Color
}

// BorderLineStyle 常量
const (
    BorderNone   BorderLineStyle = iota
    BorderSingle
    BorderDotted
    BorderDashed
    BorderDouble
    BorderTriple
    BorderThick
)
```

## 用法示例

### 下边框（装饰线）

非常适合在标题下方添加视觉分隔：

```go
h1, _ := doc.AddParagraph()
h1.SetStyle(domain.StyleIDHeading1)
h1Run, _ := h1.AddRun()
h1Run.AddText("Section Title")

// 在标题下方添加装饰线
h1.SetBorderBottom(domain.BorderStyle{
    Style: domain.BorderSingle,
    Width: 6,
    Color: domain.Color{R: 0, G: 122, B: 204}, // 蓝色
})
```

### 完整边框框

为重要文本创建边框框：

```go
para, _ := doc.AddParagraph()
run, _ := para.AddRun()
run.AddText("Important note with border on all sides")

para.SetBorders(domain.ParagraphBorders{
    Top: domain.BorderStyle{
        Style: domain.BorderSingle,
        Width: 4,
        Color: domain.Color{R: 128, G: 128, B: 128},
    },
    Bottom: domain.BorderStyle{
        Style: domain.BorderSingle,
        Width: 4,
        Color: domain.Color{R: 128, G: 128, B: 128},
    },
    Left: domain.BorderStyle{
        Style: domain.BorderSingle,
        Width: 4,
        Color: domain.Color{R: 128, G: 128, B: 128},
    },
    Right: domain.BorderStyle{
        Style: domain.BorderSingle,
        Width: 4,
        Color: domain.Color{R: 128, G: 128, B: 128},
    },
})
```

### 不同的边框样式

```go
// 虚线边框
para.SetBorderBottom(domain.BorderStyle{
    Style: domain.BorderDashed,
    Width: 6,
    Color: domain.Color{R: 200, G: 50, B: 50},
})

// 点线边框
para.SetBorderBottom(domain.BorderStyle{
    Style: domain.BorderDotted,
    Width: 6,
    Color: domain.Color{R: 200, G: 50, B: 50},
})

// 双线边框（优雅）
para.SetBorderBottom(domain.BorderStyle{
    Style: domain.BorderDouble,
    Width: 6,
    Color: domain.Color{R: 0, G: 0, B: 0},
})
```

## 实现细节

### 修改的文件

1. **domain/paragraph.go**
   - 新增 `ParagraphBorders` 类型
   - 在 `Paragraph` 接口中新增边框方法

2. **internal/core/paragraph.go**
   - 在段落结构体中新增 `borders` 字段
   - 实现了所有边框方法

3. **internal/xml/paragraph.go**
   - 新增 `ParagraphBorders` XML 结构
   - 复用了 table.go 中已有的 `Border` 类型

4. **internal/serializer/serializer.go**
   - 在 `serializeProperties` 中新增边框序列化
   - 新增 `hasBorders()`、`serializeBorder()`、`borderStyleToString()` 辅助方法

### XML 输出

生成标准的 OOXML 段落边框标记：

```xml
<w:p>
    <w:pPr>
        <w:pBdr>
            <w:bottom w:val="single" w:color="007ACC" w:sz="6"/>
        </w:pBdr>
    </w:pPr>
    <w:r>
        <w:t>Paragraph text</w:t>
    </w:r>
</w:p>
```

## 用例

1. **节分隔符**：在节标题下方添加装饰线
2. **提示框**：为重要提示或警告创建带边框的框
3. **视觉层级**：使用不同的边框样式表示不同类型的内容
4. **技术文档**：为架构文档添加专业样式
5. **目录**：在主要节之间添加分隔线

## 测试示例

完整示例见 `examples/13_themes/test_borders/main.go`，演示了：
- 标题下边框
- 完整边框框
- 不同边框样式（single、dashed、dotted、double、thick、triple）

运行测试：
```bash
cd examples/13_themes/test_borders
go run main.go
```

## 注意事项

- 边框宽度以八分之一磅为单位（与表格边框相同）
- 宽度为 8 = 1 磅
- 设置 `BorderNone` 会移除边框
- 边框兼容所有支持 OOXML 的 Word 版本
- 边框不影响布局间距（如需间距请使用 `SetSpacingBefore/After`）

## 与主题的集成

技术架构示例（`examples/13_themes/04_tech_architecture/`）演示了如何将边框与主题一起使用：

```go
h1.SetBorderBottom(domain.BorderStyle{
    Style: domain.BorderSingle,
    Width: 6,
    Color: colors.Primary, // 使用主题色
})
```

这样能在整篇文档中创建一致、主题感知的装饰线。
