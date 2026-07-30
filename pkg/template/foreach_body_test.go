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

// --- Position-based paragraph insert / delete on domain.Document ---

func TestInsertParagraph_Positioning(t *testing.T) {
	doc := core.NewDocument()

	p0, _ := doc.AddParagraph()
	r0, _ := p0.AddRun()
	_ = r0.SetText("P0")

	// A table between the two paragraphs exercises block-index bookkeeping.
	tbl, _ := doc.AddTable(1, 1)
	row, _ := tbl.Row(0)
	cell, _ := row.Cell(0)
	_, _ = cell.AddParagraph()

	p1, _ := doc.AddParagraph()
	r1, _ := p1.AddRun()
	_ = r1.SetText("P1")

	// Insert at 0 -> first paragraph, before P0 and the table.
	ins, err := doc.InsertParagraph(0)
	if err != nil {
		t.Fatalf("InsertParagraph(0): %v", err)
	}
	ir, _ := ins.AddRun()
	_ = ir.SetText("NEW0")

	paras := doc.Paragraphs()
	if len(paras) != 3 {
		t.Fatalf("expected 3 paragraphs, got %d", len(paras))
	}
	if paras[0].Text() != "NEW0" || paras[1].Text() != "P0" || paras[2].Text() != "P1" {
		t.Errorf("after head insert: %q %q %q", paras[0].Text(), paras[1].Text(), paras[2].Text())
	}

	// Insert at the end (index == len) -> appended last.
	end, err := doc.InsertParagraph(len(doc.Paragraphs()))
	if err != nil {
		t.Fatalf("InsertParagraph(end): %v", err)
	}
	er, _ := end.AddRun()
	_ = er.SetText("NEWEND")

	paras = doc.Paragraphs()
	if paras[len(paras)-1].Text() != "NEWEND" {
		t.Errorf("tail insert not last: %q", paras[len(paras)-1].Text())
	}
}

func TestDeleteParagraph_Positioning(t *testing.T) {
	doc := core.NewDocument()

	p0, _ := doc.AddParagraph()
	r0, _ := p0.AddRun()
	_ = r0.SetText("P0")
	_, _ = doc.AddTable(1, 1)
	p1, _ := doc.AddParagraph()
	r1, _ := p1.AddRun()
	_ = r1.SetText("P1")
	p2, _ := doc.AddParagraph()
	r2, _ := p2.AddRun()
	_ = r2.SetText("P2")

	// Delete the middle paragraph (P1, index 1).
	if err := doc.DeleteParagraph(1); err != nil {
		t.Fatalf("DeleteParagraph(1): %v", err)
	}
	paras := doc.Paragraphs()
	if len(paras) != 2 {
		t.Fatalf("expected 2 paragraphs, got %d", len(paras))
	}
	if paras[0].Text() != "P0" || paras[1].Text() != "P2" {
		t.Errorf("after middle delete: %q %q", paras[0].Text(), paras[1].Text())
	}
}

func TestInsertDeleteParagraph_Errors(t *testing.T) {
	doc := core.NewDocument()
	_, _ = doc.AddParagraph()

	if _, err := doc.InsertParagraph(-1); err == nil {
		t.Error("InsertParagraph(-1) should error")
	}
	if _, err := doc.InsertParagraph(2); err == nil {
		t.Error("InsertParagraph(2) should error (out of range)")
	}
	if err := doc.DeleteParagraph(-1); err == nil {
		t.Error("DeleteParagraph(-1) should error")
	}
	if err := doc.DeleteParagraph(1); err == nil {
		t.Error("DeleteParagraph(1) should error (out of range)")
	}
}

// --- Body paragraph-level foreach ---

func TestMergeTemplateWithBodyLoops_Basic(t *testing.T) {
	doc := core.NewDocument()

	title, _ := doc.AddParagraph()
	tr, _ := title.AddRun()
	_ = tr.SetText("{{Title}}")

	openP, _ := doc.AddParagraph()
	or, _ := openP.AddRun()
	_ = or.SetText("{{#foreach items}}")
	bodyP, _ := doc.AddParagraph()
	br, _ := bodyP.AddRun()
	_ = br.SetText("{{items.Name}}")
	closeP, _ := doc.AddParagraph()
	cr, _ := closeP.AddRun()
	_ = cr.SetText("{{/each}}")

	loops := map[string][]ForeachItem{
		"items": {{"Name": "Alice"}, {"Name": "Bob"}, {"Name": "Carol"}},
	}

	if err := MergeTemplateWithBodyLoops(doc, MergeData{"Title": "Report"}, loops); err != nil {
		t.Fatalf("MergeTemplateWithBodyLoops: %v", err)
	}

	paras := doc.Paragraphs()
	// title + 3 item clones (marker-only paragraphs skipped).
	if len(paras) != 4 {
		t.Fatalf("expected 4 paragraphs, got %d", len(paras))
	}
	if paras[0].Runs()[0].Text() != "Report" {
		t.Errorf("title not replaced: %q", paras[0].Runs()[0].Text())
	}
	for i, want := range []string{"Alice", "Bob", "Carol"} {
		got := paras[i+1].Text()
		if got != want {
			t.Errorf("paragraph %d = %q, want %q", i+1, got, want)
		}
		if strings.Contains(got, "each") || strings.Contains(got, "foreach") {
			t.Errorf("paragraph %d has leftover marker: %q", i+1, got)
		}
	}
}

func TestMergeTemplateWithBodyLoops_MultiParagraphBlock(t *testing.T) {
	doc := core.NewDocument()

	openP, _ := doc.AddParagraph()
	or, _ := openP.AddRun()
	_ = or.SetText("{{#foreach people}}")
	p1, _ := doc.AddParagraph()
	r1, _ := p1.AddRun()
	_ = r1.SetText("Name: {{people.Name}}")
	p2, _ := doc.AddParagraph()
	r2, _ := p2.AddRun()
	_ = r2.SetText("Role: {{people.Role}}")
	closeP, _ := doc.AddParagraph()
	cr, _ := closeP.AddRun()
	_ = cr.SetText("{{/each}}")

	loops := map[string][]ForeachItem{
		"people": {{"Name": "Ann", "Role": "Dev"}, {"Name": "Ben", "Role": "Ops"}},
	}

	if err := MergeTemplateWithBodyLoops(doc, MergeData{}, loops); err != nil {
		t.Fatalf("MergeTemplateWithBodyLoops: %v", err)
	}

	paras := doc.Paragraphs()
	// 2 items * 2 content paragraphs = 4.
	if len(paras) != 4 {
		t.Fatalf("expected 4 paragraphs, got %d", len(paras))
	}
	if paras[0].Text() != "Name: Ann" || paras[1].Text() != "Role: Dev" {
		t.Errorf("item0 wrong: %q / %q", paras[0].Text(), paras[1].Text())
	}
	if paras[2].Text() != "Name: Ben" || paras[3].Text() != "Role: Ops" {
		t.Errorf("item1 wrong: %q / %q", paras[2].Text(), paras[3].Text())
	}
}

func TestMergeTemplateWithBodyLoops_EmptyRemovesBlock(t *testing.T) {
	doc := core.NewDocument()

	title, _ := doc.AddParagraph()
	tr, _ := title.AddRun()
	_ = tr.SetText("Keep")

	openP, _ := doc.AddParagraph()
	or, _ := openP.AddRun()
	_ = or.SetText("{{#foreach items}}")
	bodyP, _ := doc.AddParagraph()
	br, _ := bodyP.AddRun()
	_ = br.SetText("{{items.Name}}")
	closeP, _ := doc.AddParagraph()
	cr, _ := closeP.AddRun()
	_ = cr.SetText("{{/each}}")

	if err := MergeTemplateWithBodyLoops(doc, MergeData{}, map[string][]ForeachItem{"items": {}}); err != nil {
		t.Fatalf("MergeTemplateWithBodyLoops: %v", err)
	}

	paras := doc.Paragraphs()
	if len(paras) != 1 {
		t.Fatalf("expected 1 paragraph after empty loop, got %d", len(paras))
	}
	if paras[0].Text() != "Keep" {
		t.Errorf("title not preserved: %q", paras[0].Text())
	}
}

func TestMergeTemplateWithBodyLoops_TwoLoops(t *testing.T) {
	doc := core.NewDocument()

	add := func(text string) {
		p, _ := doc.AddParagraph()
		r, _ := p.AddRun()
		_ = r.SetText(text)
	}
	add("{{#foreach a}}")
	add("{{a.X}}")
	add("{{/each}}")
	add("{{#foreach b}}")
	add("{{b.Y}}")
	add("{{/each}}")

	loops := map[string][]ForeachItem{
		"a": {{"X": "A1"}, {"X": "A2"}},
		"b": {{"Y": "B1"}, {"Y": "B2"}},
	}
	if err := MergeTemplateWithBodyLoops(doc, MergeData{}, loops); err != nil {
		t.Fatalf("MergeTemplateWithBodyLoops: %v", err)
	}

	paras := doc.Paragraphs()
	want := []string{"A1", "A2", "B1", "B2"}
	if len(paras) != len(want) {
		t.Fatalf("expected %d paragraphs, got %d", len(want), len(paras))
	}
	for i, w := range want {
		if paras[i].Text() != w {
			t.Errorf("paragraph %d = %q, want %q", i, paras[i].Text(), w)
		}
	}
}

func TestMergeTemplateWithBodyLoops_RoundTrip(t *testing.T) {
	doc := core.NewDocument()

	openP, _ := doc.AddParagraph()
	or, _ := openP.AddRun()
	_ = or.SetText("{{#foreach items}}")
	bodyP, _ := doc.AddParagraph()
	br, _ := bodyP.AddRun()
	_ = br.SetText("{{items.Name}}")
	closeP, _ := doc.AddParagraph()
	cr, _ := closeP.AddRun()
	_ = cr.SetText("{{/each}}")

	loops := map[string][]ForeachItem{
		"items": {{"Name": "X"}, {"Name": "Y"}},
	}
	if err := MergeTemplateWithBodyLoops(doc, MergeData{}, loops); err != nil {
		t.Fatalf("MergeTemplateWithBodyLoops: %v", err)
	}

	path := t.TempDir() + "/body_output.docx"
	if err := doc.SaveAs(path); err != nil {
		t.Fatalf("SaveAs: %v", err)
	}
	reopened, err := docx.OpenDocument(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	rp := reopened.Paragraphs()
	if len(rp) != 2 {
		t.Fatalf("reopened expected 2 paragraphs, got %d", len(rp))
	}
	if rp[0].Text() != "X" {
		t.Errorf("reopened[0] = %q, want X", rp[0].Text())
	}
	if rp[1].Text() != "Y" {
		t.Errorf("reopened[1] = %q, want Y", rp[1].Text())
	}
}
