// SPDX-License-Identifier: MIT
// Copyright (c) 2024-2026 Misael Monterroca
//
// See LICENSE for the full copyright notice, including predecessor authors,
// and CREDITS.md for the project genealogy.

package template

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mmonterroca/docxgo/v2/domain"
)

// MergeTemplate replaces all placeholders in doc with values from data.
// The document is modified in place.
//
// By default, placeholders use {{key}} syntax. Unmatched placeholders are left
// as-is unless StrictMode is enabled in options.
//
// On a document opened via OpenDocument/OpenDocumentFromBytes/
// OpenDocumentFromReader whose headers or footers were preserved for
// round-trip fidelity, header and footer placeholders are never written:
// WriteTo writes those parts verbatim, so an in-memory replacement there
// would never reach the saved file. Their keys are reported through
// missingKeys instead (and so trigger StrictMode's error), since silently
// succeeding while discarding the write on save would be worse.
func MergeTemplate(doc domain.Document, data MergeData, opts ...MergeOptions) error {
	opt := DefaultMergeOptions()
	if len(opts) > 0 {
		opt = opts[0]
	}

	pattern := BuildPattern(opt)
	skipHeaderFooter := hasPreservedHeadersOrFooters(doc)
	var missingKeys []string

	err := walkParagraphs(doc, func(para domain.Paragraph, ctx paragraphContext) error {
		if skipHeaderFooter && (ctx.locationType == LocationHeader || ctx.locationType == LocationFooter) {
			for _, ph := range scanParagraph(para, pattern, ctx) {
				missingKeys = append(missingKeys, ph.Name)
			}
			return nil
		}
		if err := ConsolidateRuns(para); err != nil {
			return err
		}
		missing, err := replaceParagraph(para, data, pattern, opt)
		if err != nil {
			return err
		}
		missingKeys = append(missingKeys, missing...)
		return nil
	})
	if err != nil {
		return err
	}

	if opt.StrictMode && len(missingKeys) > 0 {
		// Deduplicate missing keys
		seen := make(map[string]struct{})
		var unique []string
		for _, k := range missingKeys {
			if _, ok := seen[k]; !ok {
				seen[k] = struct{}{}
				unique = append(unique, k)
			}
		}
		return fmt.Errorf("template: missing keys: %s", strings.Join(unique, ", "))
	}

	return nil
}

// ValidateTemplate checks that all placeholders have corresponding keys in data
// and reports unused data keys.
func ValidateTemplate(doc domain.Document, data MergeData, opts ...MergeOptions) []ValidationError {
	opt := DefaultMergeOptions()
	if len(opts) > 0 {
		opt = opts[0]
	}

	names := placeholderNamesWithOptions(doc, opt)
	var errors []ValidationError

	// Check for missing keys (placeholders without data)
	for _, name := range names {
		if _, ok := data[name]; !ok {
			errors = append(errors, ValidationError{
				Severity: SeverityError,
				Key:      name,
				Message:  "placeholder has no corresponding data key",
			})
		}
	}

	// Check for unused keys (data without placeholders)
	nameSet := make(map[string]struct{})
	for _, n := range names {
		nameSet[n] = struct{}{}
	}
	for k := range data {
		if _, ok := nameSet[k]; !ok {
			errors = append(errors, ValidationError{
				Severity: SeverityWarning,
				Key:      k,
				Message:  "data key has no corresponding placeholder in template",
			})
		}
	}

	return errors
}

// replaceParagraph replaces all placeholders in a single paragraph.
// Returns the placeholder keys for which no data was found, and an error if a
// run's text could not be written.
//
// Matching runs over the paragraph's concatenated text (after run consolidation)
// rather than one run at a time, so a placeholder that Word has fragmented
// across several runs — which happens whenever the delimiters or the key are
// typed or formatted separately — is still matched and replaced. A match that
// cannot be rewritten safely (it spans a field, a break, or an image) is left
// untouched; see replaceSpan for those rules.
func replaceParagraph(para domain.Paragraph, data MergeData, pattern *regexp.Regexp, opt MergeOptions) ([]string, error) {
	var missing []string

	if err := ConsolidateRuns(para); err != nil {
		return missing, err
	}

	cursor := 0
	for {
		spans, full := paragraphSpans(para)
		if cursor >= len(full) {
			break
		}
		loc := pattern.FindStringIndex(full[cursor:])
		if loc == nil {
			break
		}
		start := cursor + loc[0]
		end := cursor + loc[1]
		key := extractKey(full[start:end], opt)

		val, ok := data[key]
		if !ok {
			missing = append(missing, key)
			cursor = end
			continue
		}

		okWrite, err := replaceSpan(spans, start, end, val)
		if err != nil {
			return missing, err
		}
		if !okWrite {
			// Unsafe to rewrite (spanning a field/break/image); leave as-is.
			cursor = end
			continue
		}
		cursor = start + len(val)
	}
	return missing, nil
}

// BuildPattern creates a regex pattern from the configured delimiters.
func BuildPattern(opt MergeOptions) *regexp.Regexp {
	open := regexp.QuoteMeta(opt.OpenDelimiter)
	close := regexp.QuoteMeta(opt.CloseDelimiter)
	return regexp.MustCompile(open + `\.?(\w+(?:\.\w+)*)` + close)
}

// extractKey extracts the placeholder key from a full match string.
func extractKey(match string, opt MergeOptions) string {
	// Remove delimiters
	key := match
	key = strings.TrimPrefix(key, opt.OpenDelimiter)
	key = strings.TrimSuffix(key, opt.CloseDelimiter)
	// Remove optional leading dot
	key = strings.TrimPrefix(key, ".")
	return key
}

// placeholderNamesWithOptions returns deduplicated placeholder names using custom options.
func placeholderNamesWithOptions(doc domain.Document, opt MergeOptions) []string {
	pattern := BuildPattern(opt)
	placeholders := findPlaceholdersWithPattern(doc, pattern)

	seen := make(map[string]struct{})
	var names []string
	for _, p := range placeholders {
		if _, ok := seen[p.Name]; !ok {
			seen[p.Name] = struct{}{}
			names = append(names, p.Name)
		}
	}
	return names
}
