// SPDX-License-Identifier: MIT
// Copyright (c) 2024-2026 Misael Monterroca
//
// See LICENSE for the full copyright notice, including predecessor authors,
// and CREDITS.md for the project genealogy.

package template

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	docx "github.com/mmonterroca/docxgo/v2"
	"github.com/mmonterroca/docxgo/v2/domain"
	"github.com/mmonterroca/docxgo/v2/internal/core"
)

// makePNG generates a solid-color PNG of the given size.
func makePNG(t *testing.T, w, h int, c color.Color) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func TestReplaceImage_Basic(t *testing.T) {
	oldData := makePNG(t, 4, 4, color.RGBA{R: 255, A: 255})
	newData := makePNG(t, 8, 8, color.RGBA{B: 255, A: 255})

	doc := core.NewDocument()
	para, _ := doc.AddParagraph()
	img, err := para.AddImageFromBytes(oldData, domain.ImageFormatPNG)
	if err != nil {
		t.Fatalf("AddImageFromBytes: %v", err)
	}
	if err := img.SetDescription("Team photo {{IMAGE .PHOTO}}"); err != nil {
		t.Fatalf("SetDescription: %v", err)
	}
	displayed := img.Size()

	result, err := ReplaceImage(doc, "{{IMAGE .PHOTO}}", newData)
	if err != nil {
		t.Fatalf("ReplaceImage: %v", err)
	}
	if result.Replaced != 1 || result.Skipped != 0 {
		t.Errorf("result = %+v, want 1 replaced / 0 skipped", result)
	}
	if !bytes.Equal(img.Data(), newData) {
		t.Error("image data was not replaced")
	}
	if img.Size() != displayed {
		t.Errorf("displayed size changed: %+v -> %+v", displayed, img.Size())
	}
	if got := img.Description(); got != "Team photo" {
		t.Errorf("description = %q, want marker stripped", got)
	}
}

func TestReplaceImage_NoMatch(t *testing.T) {
	data := makePNG(t, 4, 4, color.RGBA{G: 255, A: 255})

	doc := core.NewDocument()
	para, _ := doc.AddParagraph()
	img, _ := para.AddImageFromBytes(data, domain.ImageFormatPNG)
	_ = img.SetDescription("just a picture")

	result, err := ReplaceImage(doc, "{{IMAGE .PHOTO}}", data)
	if err != nil {
		t.Fatalf("ReplaceImage: %v", err)
	}
	if result.Replaced != 0 || result.Skipped != 0 {
		t.Errorf("result = %+v, want 0 replaced / 0 skipped", result)
	}
}

func TestReplaceImage_InvalidArgs(t *testing.T) {
	doc := core.NewDocument()
	data := makePNG(t, 2, 2, color.White)

	if _, err := ReplaceImage(doc, "", data); err == nil {
		t.Error("expected error for empty find")
	}
	if _, err := ReplaceImage(doc, "{{IMAGE .X}}", nil); err == nil {
		t.Error("expected error for empty image data")
	}
}

func TestReplaceImage_InTableCell(t *testing.T) {
	oldData := makePNG(t, 4, 4, color.RGBA{R: 255, A: 255})
	newData := makePNG(t, 4, 4, color.RGBA{B: 255, A: 255})

	doc := core.NewDocument()
	table, _ := doc.AddTable(1, 1)
	para, _ := table.Rows()[0].Cells()[0].AddParagraph()
	img, err := para.AddImageFromBytes(oldData, domain.ImageFormatPNG)
	if err != nil {
		t.Fatalf("AddImageFromBytes: %v", err)
	}
	_ = img.SetDescription("{{IMAGE .LOGO}}")

	result, err := ReplaceImage(doc, "{{IMAGE .LOGO}}", newData)
	if err != nil {
		t.Fatalf("ReplaceImage: %v", err)
	}
	if result.Replaced != 1 {
		t.Errorf("replaced = %d, want 1", result.Replaced)
	}
	if !bytes.Equal(img.Data(), newData) {
		t.Error("image data in table cell was not replaced")
	}
}

func TestReplaceImages_Batch(t *testing.T) {
	oldA := makePNG(t, 4, 4, color.RGBA{R: 255, A: 255})
	oldB := makePNG(t, 4, 4, color.RGBA{G: 255, A: 255})
	newA := makePNG(t, 4, 4, color.RGBA{B: 255, A: 255})
	newB := makePNG(t, 4, 4, color.RGBA{R: 255, G: 255, A: 255})

	doc := core.NewDocument()
	p1, _ := doc.AddParagraph()
	imgA, _ := p1.AddImageFromBytes(oldA, domain.ImageFormatPNG)
	_ = imgA.SetDescription("{{IMAGE .PHOTO}}")

	p2, _ := doc.AddParagraph()
	imgB, _ := p2.AddImageFromBytes(oldB, domain.ImageFormatPNG)
	_ = imgB.SetDescription("{{IMAGE .LOGO}}")

	result, err := ReplaceImages(doc, map[string][]byte{
		"{{IMAGE .PHOTO}}": newA,
		"{{IMAGE .LOGO}}":  newB,
	})
	if err != nil {
		t.Fatalf("ReplaceImages: %v", err)
	}
	if result.Replaced != 2 {
		t.Errorf("replaced = %d, want 2", result.Replaced)
	}
	if !bytes.Equal(imgA.Data(), newA) {
		t.Error("imgA data not replaced")
	}
	if !bytes.Equal(imgB.Data(), newB) {
		t.Error("imgB data not replaced")
	}
}

// TestReplaceImage_RoundTrip verifies the replaced bytes actually reach the
// saved .docx: build a template with a marked image, save it, reopen it,
// replace the image, save again, reopen and inspect the embedded data.
func TestReplaceImage_RoundTrip(t *testing.T) {
	oldData := makePNG(t, 4, 4, color.RGBA{R: 255, A: 255})
	newData := makePNG(t, 6, 6, color.RGBA{B: 255, A: 255})

	// Build and save the template.
	doc := core.NewDocument()
	para, _ := doc.AddParagraph()
	img, err := para.AddImageFromBytes(oldData, domain.ImageFormatPNG)
	if err != nil {
		t.Fatalf("AddImageFromBytes: %v", err)
	}
	if err := img.SetDescription("{{IMAGE .PHOTO}}"); err != nil {
		t.Fatalf("SetDescription: %v", err)
	}

	templatePath := t.TempDir() + "/img_template.docx"
	if err := doc.SaveAs(templatePath); err != nil {
		t.Fatalf("save template: %v", err)
	}

	// Reopen, replace, save output.
	doc2, err := docx.OpenDocument(templatePath)
	if err != nil {
		t.Fatalf("open template: %v", err)
	}
	result, err := ReplaceImage(doc2, "{{IMAGE .PHOTO}}", newData)
	if err != nil {
		t.Fatalf("ReplaceImage: %v", err)
	}
	if result.Replaced != 1 {
		t.Fatalf("replaced = %d, want 1", result.Replaced)
	}

	outputPath := t.TempDir() + "/img_output.docx"
	if err := doc2.SaveAs(outputPath); err != nil {
		t.Fatalf("save output: %v", err)
	}

	// Reopen the output and verify the embedded image data.
	doc3, err := docx.OpenDocument(outputPath)
	if err != nil {
		t.Fatalf("reopen output: %v", err)
	}

	var found bool
	err = walkParagraphs(doc3, func(p domain.Paragraph, ctx paragraphContext) error {
		for _, r := range p.Runs() {
			if ri := r.Image(); ri != nil {
				found = true
				if !bytes.Equal(ri.Data(), newData) {
					t.Error("saved image data does not match replacement")
				}
				if ri.Description() != "" {
					t.Errorf("saved description = %q, want empty", ri.Description())
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk output: %v", err)
	}
	if !found {
		t.Fatal("no image found in saved output")
	}
}
