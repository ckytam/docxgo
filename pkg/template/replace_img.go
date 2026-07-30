// SPDX-License-Identifier: MIT
// Copyright (c) 2024-2026 Misael Monterroca
//
// See LICENSE for the full copyright notice, including predecessor authors,
// and CREDITS.md for the project genealogy.

package template

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/mmonterroca/docxgo/v2/domain"
)

// ReplaceImageResult reports the outcome of a ReplaceImage call.
type ReplaceImageResult struct {
	// Replaced is the number of images whose data was replaced.
	Replaced int
	// Skipped is the number of matching images that were found but left
	// untouched because replacing them would not survive serialization
	// (images in preserved headers/footers of a round-trip document).
	Skipped int
}

// ReplaceImage replaces the binary data of every image whose alt-text
// description contains find, across body paragraphs, table cells, headers,
// and footers. The document is modified in place.
//
// Template workflow: in Word, right-click the placeholder image, choose
// "View Alt Text", and put a marker in the description box, e.g.:
//
//	{{IMAGE .PHOTO}}
//
// Then call:
//
//	template.ReplaceImage(doc, "{{IMAGE .PHOTO}}", photoBytes)
//
// Matching is literal and case-sensitive; the description only has to
// contain find, not equal it. After a successful replacement the marker is
// removed from the image's alt text, so the saved document does not ship
// the template tag.
//
// The replacement keeps the image's media path, relationship, displayed
// size, and position, so the surrounding layout is preserved: the new data
// is rendered at the placeholder's size regardless of its own dimensions.
// The new data must be PNG, JPEG, or GIF. Supplying data in the same format
// as the placeholder image is recommended for maximum compatibility, since
// the media part keeps its original file extension.
//
// If several images share the same media file, replacing one changes what
// all of them display. On a document opened via OpenDocument* whose headers
// or footers were preserved for round-trip fidelity, matches in headers and
// footers are skipped and counted in ReplaceImageResult.Skipped — WriteTo
// writes those parts verbatim, so an in-memory edit there would never reach
// the saved file.
func ReplaceImage(doc domain.Document, find string, imageData []byte) (ReplaceImageResult, error) {
	if find == "" {
		return ReplaceImageResult{}, fmt.Errorf("template: find text must not be empty")
	}
	if len(imageData) == 0 {
		return ReplaceImageResult{}, fmt.Errorf("template: image data must not be empty")
	}

	skipHeaderFooter := hasPreservedHeadersOrFooters(doc)

	var result ReplaceImageResult
	err := walkParagraphs(doc, func(para domain.Paragraph, ctx paragraphContext) error {
		for _, r := range para.Runs() {
			img := r.Image()
			if img == nil || !strings.Contains(img.Description(), find) {
				continue
			}

			if skipHeaderFooter && (ctx.locationType == LocationHeader || ctx.locationType == LocationFooter) {
				result.Skipped++
				continue
			}

			if err := doc.ReplaceImageData(img, imageData); err != nil {
				return err
			}
			// Strip the template marker from the alt text so the saved
			// document does not carry it.
			desc := strings.TrimSpace(strings.ReplaceAll(img.Description(), find, ""))
			if err := img.SetDescription(desc); err != nil {
				return err
			}
			result.Replaced++
		}
		return nil
	})
	return result, err
}

// ReplaceImageFromFile is a convenience wrapper around ReplaceImage that
// reads the replacement image from a file on disk.
func ReplaceImageFromFile(doc domain.Document, find, imagePath string) (ReplaceImageResult, error) {
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return ReplaceImageResult{}, fmt.Errorf("template: read image file: %w", err)
	}
	return ReplaceImage(doc, find, data)
}

// ReplaceImages replaces multiple images in one call. The map key is the
// alt-text marker to search for (see ReplaceImage) and the value is the
// replacement image data. Markers are processed in sorted order so results
// are deterministic. The returned result aggregates all replacements.
//
// Processing stops at the first error; replacements already applied are
// kept, and the partial result is returned alongside the error.
func ReplaceImages(doc domain.Document, images map[string][]byte) (ReplaceImageResult, error) {
	keys := make([]string, 0, len(images))
	for k := range images {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var total ReplaceImageResult
	for _, find := range keys {
		res, err := ReplaceImage(doc, find, images[find])
		total.Replaced += res.Replaced
		total.Skipped += res.Skipped
		if err != nil {
			return total, err
		}
	}
	return total, nil
}
