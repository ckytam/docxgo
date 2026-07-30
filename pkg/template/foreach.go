// SPDX-License-Identifier: MIT
// Copyright (c) 2024-2026 Misael Monterroca
//
// See LICENSE for the full copyright notice, including predecessor authors,
// and CREDITS.md for the project genealogy.

package template

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/mmonterroca/docxgo/v2/domain"
)

// ForeachItem is a single element of a looped collection. Its keys are the
// field names available inside the matching {{#foreach name}}...{{/each}}
// block, referenced as {{name.Field}}.
//
// For example, given the loop `{{#foreach items}}` and an item
// ForeachItem{"Name": "Alice", "Price": "9.99"}, the body uses
// `{{items.Name}}` and `{{items.Price}}`.
type ForeachItem map[string]string

// foreachOpenRe matches the opening tag of a loop block and captures the
// loop variable name, e.g. "items" from "{{#foreach items}}".
var foreachOpenRe = regexp.MustCompile(`\{\{#foreach\s+(\w+)\}\}`)

// foreachCloseRe matches the closing tag of a loop block.
var foreachCloseRe = regexp.MustCompile(`\{\{/each\}\}`)

// loopMarkerRe matches a whole loop tag (open or close) so it can be stripped
// from a rendered copy.
var loopMarkerRe = regexp.MustCompile(`\{\{#foreach\s+\w+\}\}|\{\{/each\}\}`)

// MergeTemplateWithLoops merges scalar {{key}} placeholders (via data) exactly
// like MergeTemplate, and additionally repeats every
// {{#foreach name}}...{{/each}} table row once per entry in loops[name].
//
// The whole row containing the markers is treated as the template: it is
// cloned once per item, the clone's body placeholders ({{name.Field}}) are
// filled with that item's fields, the loop markers are stripped, and the
// original template row is removed. Rows are processed so that the first item
// appears directly below where the template row was.
//
// If loops[name] is missing or empty for a loop, the template row is removed
// (leaving no stray markers). Placeholders outside loops are filled by data
// as usual; placeholders inside a loop body may also reference outer scalar
// data (already substituted before cloning).
//
// Notes / v1 limitations:
//   - Only top-level tables are scanned; loops inside nested (cell) tables are
//     ignored, and a nested table copied as part of a repeated row is not
//     re-expanded.
//   - The opening and closing tags should each sit within a single run (do not
//     partially format "{{#foreach items}}" or "{{/each}}"); after consolidation
//     the tag is removed per-run, so a tag split across runs with differing
//     formatting would not be fully stripped.
func MergeTemplateWithLoops(doc domain.Document, data MergeData, loops map[string][]ForeachItem, opts ...MergeOptions) error {
	opt := DefaultMergeOptions()
	if len(opts) > 0 {
		opt = opts[0]
	}

	// Replace scalar placeholders everywhere first (loop markers are left
	// intact because they do not match the placeholder pattern). Cloning
	// later copies these already-substituted runs.
	if err := MergeTemplate(doc, data, opt); err != nil {
		return err
	}

	pattern := BuildPattern(opt)

	for _, table := range doc.Tables() {
		if err := expandTableForeach(table, loops, pattern, opt); err != nil {
			return err
		}
	}
	if err := expandBodyForeach(doc, loops, pattern, opt); err != nil {
		return err
	}
	return nil
}

// MergeTemplateWithBodyLoops merges scalar {{key}} placeholders (via data) exactly
// like MergeTemplate, and additionally repeats every
// {{#foreach name}}...{{/each}} block of body (top-level) paragraphs once per
// entry in loops[name].
//
// Unlike the table-level loop, the whole block of paragraphs between the markers
// is cloned as a unit, so a loop can span several paragraphs (not just a single
// table row). See expandBodyForeach for the full behavior and v1 limitations.
//
// If loops[name] is missing or empty for a loop, its template block is removed
// (leaving no stray markers).
func MergeTemplateWithBodyLoops(doc domain.Document, data MergeData, loops map[string][]ForeachItem, opts ...MergeOptions) error {
	opt := DefaultMergeOptions()
	if len(opts) > 0 {
		opt = opts[0]
	}

	// Replace scalar placeholders everywhere first so the cloned paragraphs
	// inherit already-substituted outer data.
	if err := MergeTemplate(doc, data, opt); err != nil {
		return err
	}

	pattern := BuildPattern(opt)
	return expandBodyForeach(doc, loops, pattern, opt)
}

// expandBodyForeach finds every {{#foreach name}}...{{/each}} block among the
// document's top-level paragraphs and repeats the whole block of paragraphs once
// per entry in loops[name]. Blocks are processed in descending start position so
// that inserting and deleting paragraphs never shifts an index we have not yet
// handled.
//
// The block is the contiguous run of paragraphs from the opening-marker paragraph
// to the closing-marker paragraph (inclusive). Each item clones the entire block
// — formatting, runs (including images), and per-item placeholders — then the
// markers are stripped and the original block is removed. A loop with no entries
// removes its template block entirely.
//
// v1 limitations:
//   - Only body (top-level) paragraphs are scanned; loops inside table cells or
//     headers/footers are ignored, and a table nested between the markers is not
//     part of the repeated block.
//   - As with table loops, each marker must sit within a single run.
func expandBodyForeach(doc domain.Document, loops map[string][]ForeachItem, pattern *regexp.Regexp, opt MergeOptions) error {
	paras := doc.Paragraphs()
	if len(paras) == 0 {
		return nil
	}

	type loopBlock struct {
		start int
		end   int
		name  string
	}
	var blocks []loopBlock

	openStart := -1
	openName := ""
	for i, p := range paras {
		text := p.Text()
		if m := foreachOpenRe.FindStringSubmatch(text); m != nil {
			if openStart == -1 {
				openStart = i
				openName = m[1]
			}
			// A second open before a close is not yet supported; it is ignored
			// until the pending close resolves the current block.
			continue
		}
		if foreachCloseRe.MatchString(text) {
			if openStart != -1 {
				blocks = append(blocks, loopBlock{openStart, i, openName})
				openStart = -1
			}
		}
	}

	// Process descending by start so later (higher-index) blocks are expanded
	// first, leaving earlier paragraph indices stable.
	sort.Slice(blocks, func(a, b int) bool {
		return blocks[a].start > blocks[b].start
	})

	for _, lb := range blocks {
		k := lb.end - lb.start + 1
		items := loops[lb.name]

		if len(items) == 0 {
			for idx := lb.end; idx >= lb.start; idx-- {
				if err := doc.DeleteParagraph(idx); err != nil {
					return fmt.Errorf("template: remove empty body loop block: %w", err)
				}
			}
			continue
		}

		// Phase A: insert clones before the template block, in order. A running
		// insert position keeps the clones contiguous regardless of how many
		// marker-only paragraphs are skipped below.
		pos := lb.start
		for _, item := range items {
			render := buildLoopData(item, lb.name)
			for r := 0; r < k; r++ {
				// paras still references the original (unmodified) template
				// paragraphs, so copy from the source at relative position r.
				tp := paras[lb.start+r]

				// Skip paragraphs that consist solely of loop markers; after
				// stripping they would leave an empty paragraph (extra blank
				// line) in each generated block.
				if strings.TrimSpace(loopMarkerRe.ReplaceAllString(tp.Text(), "")) == "" {
					continue
				}

				np, err := doc.InsertParagraph(pos)
				if err != nil {
					return fmt.Errorf("template: insert body loop paragraph: %w", err)
				}
				copyParagraphFormat(np, tp)
				if err := copyRuns(np, tp); err != nil {
					return err
				}
				if err := ConsolidateRuns(np); err != nil {
					return err
				}
				if _, err := replaceParagraph(np, render, pattern, opt); err != nil {
					return err
				}
				stripLoopMarkers(np)
				pos++
			}
		}

		// Phase B: remove the original template block, now shifted up by the
		// inserted clones.
		base := pos
		for idx := base + k - 1; idx >= base; idx-- {
			if err := doc.DeleteParagraph(idx); err != nil {
				return fmt.Errorf("template: remove body loop template: %w", err)
			}
		}
	}
	return nil
}

// expandTableForeach finds every foreach row in the table and repeats it.
// Rows are processed in descending index order so that inserting/deleting
// rows never shifts an index we have not yet handled.
func expandTableForeach(table domain.Table, loops map[string][]ForeachItem, pattern *regexp.Regexp, opt MergeOptions) error {
	rows := table.Rows()

	type foreachRow struct {
		index int
		name  string
	}
	var targets []foreachRow
	for i, row := range rows {
		if name, ok := detectForeachRow(row); ok {
			targets = append(targets, foreachRow{i, name})
		}
	}

	for k := len(targets) - 1; k >= 0; k-- {
		tgt := targets[k]
		items := loops[tgt.name]
		templateRow := rows[tgt.index] // live reference; still present until we delete it

		if len(items) == 0 {
			if err := table.DeleteRow(tgt.index); err != nil {
				return fmt.Errorf("template: remove empty foreach row: %w", err)
			}
			continue
		}

		for j, item := range items {
			newRow, err := table.InsertRow(tgt.index + j + 1)
			if err != nil {
				return fmt.Errorf("template: insert foreach row: %w", err)
			}
			render := buildLoopData(item, tgt.name)
			if err := copyRowContent(newRow, templateRow, render, pattern, opt); err != nil {
				return err
			}
		}

		if err := table.DeleteRow(tgt.index); err != nil {
			return fmt.Errorf("template: remove foreach template row: %w", err)
		}
	}
	return nil
}

// detectForeachRow reports the loop variable name if the row's text contains
// both an opening and a closing loop tag.
func detectForeachRow(row domain.TableRow) (string, bool) {
	var sb strings.Builder
	for _, cell := range row.Cells() {
		for _, p := range cell.Paragraphs() {
			sb.WriteString(p.Text())
		}
	}
	text := sb.String()
	m := foreachOpenRe.FindStringSubmatch(text)
	if m == nil || !foreachCloseRe.MatchString(text) {
		return "", false
	}
	return m[1], true
}

// buildLoopData maps an item's fields to keys prefixed with the loop variable
// name, e.g. {"Name": "Alice"} with name "items" -> {"items.Name": "Alice"}.
func buildLoopData(item ForeachItem, name string) MergeData {
	d := make(MergeData, len(item))
	for k, v := range item {
		d[name+"."+k] = v
	}
	return d
}

// copyRowContent clones a template row's cells (formatting + paragraphs + runs,
// including images) into a new row and renders the per-item placeholders,
// then strips the loop markers from the copies.
func copyRowContent(newRow, templateRow domain.TableRow, render MergeData, pattern *regexp.Regexp, opt MergeOptions) error {
	tcells := templateRow.Cells()
	ncells := newRow.Cells()

	for c := 0; c < len(tcells) && c < len(ncells); c++ {
		tcell := tcells[c]
		ncell := ncells[c]

		copyCellFormat(ncell, tcell)

		// Drop the default empty paragraph(s) created with the new row.
		for len(ncell.Paragraphs()) > 0 {
			if err := ncell.RemoveParagraph(0); err != nil {
				return fmt.Errorf("template: clear cell paragraph: %w", err)
			}
		}

		for _, tp := range tcell.Paragraphs() {
			// Skip paragraphs that consist solely of loop markers; after
			// stripping they would leave an empty paragraph (extra blank
			// line) in each generated row.
			if strings.TrimSpace(loopMarkerRe.ReplaceAllString(tp.Text(), "")) == "" {
				continue
			}

			np, err := ncell.AddParagraph()
			if err != nil {
				return fmt.Errorf("template: add cell paragraph: %w", err)
			}
			copyParagraphFormat(np, tp)
			if err := copyRuns(np, tp); err != nil {
				return err
			}
			if err := ConsolidateRuns(np); err != nil {
				return err
			}
			if _, err := replaceParagraph(np, render, pattern, opt); err != nil {
				return err
			}
			stripLoopMarkers(np)
		}
	}
	return nil
}

// copyCellFormat copies cell-level formatting (width, shading, borders,
// vertical alignment, merge spans) from src to dst.
func copyCellFormat(dst, src domain.TableCell) {
	if w := src.Width(); w > 0 {
		_ = dst.SetWidth(w)
	}
	dst.SetShading(src.Shading())
	dst.SetBorders(src.Borders())
	dst.SetVerticalAlignment(src.VerticalAlignment())
	if gs := src.GridSpan(); gs > 1 {
		_ = dst.SetGridSpan(gs)
	}
	if vm := src.VMerge(); vm != domain.VMergeNone {
		_ = dst.SetVMerge(vm)
	}
}

// copyParagraphFormat copies paragraph-level formatting from src to dst.
func copyParagraphFormat(dst, src domain.Paragraph) {
	_ = dst.SetAlignment(src.Alignment())
	if sb := src.SpacingBefore(); sb != 0 {
		_ = dst.SetSpacingBefore(sb)
	}
	if sa := src.SpacingAfter(); sa != 0 {
		_ = dst.SetSpacingAfter(sa)
	}
	dst.SetLineSpacing(src.LineSpacing())
	dst.SetIndent(src.Indent())
	if num, ok := src.Numbering(); ok {
		_ = dst.SetNumbering(num)
	}
	dst.SetBorders(src.Borders())
}

// copyRuns recreates every run of src in dst, preserving text, formatting,
// breaks, and (by re-registering) images.
func copyRuns(dst, src domain.Paragraph) error {
	for _, sr := range src.Runs() {
		if img := sr.Image(); img != nil {
			// Images are added to the paragraph (not a text run); preserve
			// the displayed size so the layout matches the template.
			if _, err := dst.AddImageFromBytesWithSize(img.Data(), img.Format(), img.Size()); err != nil {
				return fmt.Errorf("template: copy run image: %w", err)
			}
			continue
		}

		dr, err := dst.AddRun()
		if err != nil {
			return fmt.Errorf("template: add run: %w", err)
		}
		if err := dr.SetText(sr.Text()); err != nil {
			return fmt.Errorf("template: set run text: %w", err)
		}
		dr.SetFont(sr.Font())
		dr.SetColor(sr.Color())
		dr.SetSize(sr.Size())
		dr.SetBold(sr.Bold())
		dr.SetItalic(sr.Italic())
		dr.SetUnderline(sr.Underline())
		dr.SetStrike(sr.Strike())
		dr.SetHighlight(sr.Highlight())
		for _, b := range sr.Breaks() {
			_ = dr.AddBreak(b)
		}
	}
	return nil
}

// stripLoopMarkers removes any leftover loop tags from the paragraph's runs.
func stripLoopMarkers(para domain.Paragraph) {
	for _, run := range para.Runs() {
		t := run.Text()
		if t == "" || !loopMarkerRe.MatchString(t) {
			continue
		}
		_ = run.SetText(loopMarkerRe.ReplaceAllString(t, ""))
	}
}
