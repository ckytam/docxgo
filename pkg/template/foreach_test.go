// SPDX-License-Identifier: MIT
// Copyright (c) 2024-2026 Misael Monterroca
//
// See LICENSE for the full copyright notice, including predecessor authors,
// and CREDITS.md for the project genealogy.

package template

import (
	"strings"
	"testing"

	"github.com/mmonterroca/docxgo/v2"
	"github.com/mmonterroca/docxgo/v2/internal/core"
)

func TestMergeTemplateWithLoops_Basic(t *testing.T) {
	doc := core.NewDocument()

	// Scalar content outside the loop.
	title, err := doc.AddParagraph()
	if err != nil {
		t.Fatalf("AddParagraph: %v", err)
	}
	tr, err := title.AddRun()
	if err != nil {
		t.Fatalf("AddRun: %v", err)
	}
	if err := tr.SetText("{{Title}}"); err != nil {
		t.Fatalf("SetText: %v", err)
	}

	// A 2x2 table: header row + a foreach template row.
	table, err := doc.AddTable(2, 2)
	if err != nil {
		t.Fatalf("AddTable: %v", err)
	}

	rows := table.Rows()
	// Header.
	hp, _ := rows[0].Cells()[0].AddParagraph()
	hr, _ := hp.AddRun()
	_ = hr.SetText("Name")
	hp2, _ := rows[0].Cells()[1].AddParagraph()
	hr2, _ := hp2.AddRun()
	_ = hr2.SetText("Price")

	// Foreach template row: open tag in column 0, body + close in column 1.
	frow := rows[1]
	cp, _ := frow.Cells()[0].AddParagraph()
	cr, _ := cp.AddRun()
	_ = cr.SetText("{{#foreach items}}")

	c1p, _ := frow.Cells()[1].AddParagraph()
	c1r, _ := c1p.AddRun()
	_ = c1r.SetText("{{items.Name}}")
	c1r2, _ := c1p.AddRun()
	_ = c1r2.SetText("{{/each}}")

	loops := map[string][]ForeachItem{
		"items": {
			{"Name": "Alice"},
			{"Name": "Bob"},
			{"Name": "Carol"},
		},
	}

	if err := MergeTemplateWithLoops(doc, MergeData{"Title": "Report"}, loops); err != nil {
		t.Fatalf("MergeTemplateWithLoops: %v", err)
	}

	got := doc.Tables()[0].Rows()
	if len(got) != 4 { // header + 3 items
		t.Fatalf("expected 4 rows, got %d", len(got))
	}

	// Header intact.
	if got[0].Cells()[0].Paragraphs()[0].Runs()[0].Text() != "Name" {
		t.Errorf("header cell 0 not preserved: %q", got[0].Cells()[0].Paragraphs()[0].Runs()[0].Text())
	}

	for i, want := range []string{"Alice", "Bob", "Carol"} {
		cellText := got[i+1].Cells()[1].Paragraphs()[0].Text()
		if cellText != want {
			t.Errorf("row %d cell 1 = %q, want %q", i+1, cellText, want)
		}
		// No leftover markers anywhere in the row.
		for _, cell := range got[i+1].Cells() {
			for _, p := range cell.Paragraphs() {
				if strings.Contains(p.Text(), "foreach") || strings.Contains(p.Text(), "/each") {
					t.Errorf("row %d has leftover loop marker: %q", i+1, p.Text())
				}
			}
		}
	}

	// Scalar replaced.
	if title.Runs()[0].Text() != "Report" {
		t.Errorf("scalar Title not replaced: %q", title.Runs()[0].Text())
	}
}

func TestMergeTemplateWithLoops_EmptyRemovesRow(t *testing.T) {
	doc := core.NewDocument()

	table, err := doc.AddTable(2, 1)
	if err != nil {
		t.Fatalf("AddTable: %v", err)
	}
	rows := table.Rows()
	hp, _ := rows[0].Cells()[0].AddParagraph()
	hr, _ := hp.AddRun()
	_ = hr.SetText("Header")

	fp, _ := rows[1].Cells()[0].AddParagraph()
	fr, _ := fp.AddRun()
	_ = fr.SetText("{{#foreach items}}")
	fr2, _ := fp.AddRun()
	_ = fr2.SetText("{{/each}}")

	// items empty -> template row removed.
	if err := MergeTemplateWithLoops(doc, MergeData{}, map[string][]ForeachItem{"items": {}}); err != nil {
		t.Fatalf("MergeTemplateWithLoops: %v", err)
	}

	got := doc.Tables()[0].Rows()
	if len(got) != 1 {
		t.Fatalf("expected 1 row after empty loop, got %d", len(got))
	}
	if got[0].Cells()[0].Paragraphs()[0].Runs()[0].Text() != "Header" {
		t.Errorf("header not preserved: %q", got[0].Cells()[0].Paragraphs()[0].Runs()[0].Text())
	}
}

func TestMergeTemplateWithLoops_RoundTrip(t *testing.T) {
	doc := core.NewDocument()

	table, err := doc.AddTable(2, 1)
	if err != nil {
		t.Fatalf("AddTable: %v", err)
	}
	rows := table.Rows()
	hp, _ := rows[0].Cells()[0].AddParagraph()
	hr, _ := hp.AddRun()
	_ = hr.SetText("Items")

	fp, _ := rows[1].Cells()[0].AddParagraph()
	fr, _ := fp.AddRun()
	_ = fr.SetText("{{#foreach items}}")
	fr2, _ := fp.AddRun()
	_ = fr2.SetText("{{items.Name}}")
	fr3, _ := fp.AddRun()
	_ = fr3.SetText("{{/each}}")

	loops := map[string][]ForeachItem{
		"items": {{"Name": "X"}, {"Name": "Y"}},
	}
	if err := MergeTemplateWithLoops(doc, MergeData{}, loops); err != nil {
		t.Fatalf("MergeTemplateWithLoops: %v", err)
	}

	reopenPath := t.TempDir() + "/foreach_output.docx"
	if err := doc.SaveAs(reopenPath); err != nil {
		t.Fatalf("SaveAs: %v", err)
	}

	reopened, err := docx.OpenDocument(reopenPath)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}

	rt := reopened.Tables()[0].Rows()
	if len(rt) != 3 {
		t.Fatalf("reopened expected 3 rows, got %d", len(rt))
	}
	if rt[1].Cells()[0].Paragraphs()[0].Text() != "X" {
		t.Errorf("reopened row1 = %q, want X", rt[1].Cells()[0].Paragraphs()[0].Text())
	}
	if rt[2].Cells()[0].Paragraphs()[0].Text() != "Y" {
		t.Errorf("reopened row2 = %q, want Y", rt[2].Cells()[0].Paragraphs()[0].Text())
	}
}
