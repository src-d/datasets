package ld

import (
	"regexp"
	"strings"
)

var (
	lineEndingsRe = regexp.MustCompile("\\r\\n?")
	// 3.1.1 All whitespace should be treated as a single blank space.
	whitespaceRe         = regexp.MustCompile("[\\t\\f\\r              　​]+")
	trailingWhitespaceRe = regexp.MustCompile("(?m)[\\t\\f\\r              　​]$")
	leadingWhitespaceRe  = regexp.MustCompile("(?m)^[\\t\\f\\r              　​]")
	// 5.1.2 Hyphens, Dashes  Any hyphen, dash, en dash, em dash, or other variation should be
	// considered equivalent.
	punctuationRe = regexp.MustCompile("[-‒–—―⁓⸺⸻~˗‐‑⁃⁻₋−∼⎯⏤─➖𐆑֊﹘﹣－]+")
	// 5.1.3 Quotes  Any variation of quotations (single, double, curly, etc.) should be considered
	// equivalent.
	quotesRe = regexp.MustCompile("[\"'“”‘’„‚«»‹›❛❜❝❞`]+")
	// 7.1.1 Where a line starts with a bullet, number, letter, or some form of a list item
	// (determined where list item is followed by a space, then the text of the sentence), ignore
	// the list item for matching purposes.
	bulletRe = regexp.MustCompile("(?m)^(([-*✱﹡•●⚫⏺🞄∙⋅])|([(\\[{]?\\d+[.)\\]}] ?)|([(\\[{]?[a-z][.)\\]}] ?)|([(\\[{]?i+[.)\\]} ] ?))")
	// 8.1.1 The words in the following columns are considered equivalent and interchangeable.
	wordReplacer = strings.NewReplacer(
		"acknowledgment", "acknowledgement",
		"analogue", "analog",
		"analyse", "analyze",
		"artefact", "artifact",
		"authorisation", "authorization",
		"authorised", "authorized",
		"calibre", "caliber",
		"cancelled", "canceled",
		"capitalisations", "capitalizations",
		"catalogue", "catalog",
		"categorise", "categorize",
		"centre", "center",
		"emphasised", "emphasized",
		"favour", "favor",
		"favourite", "favorite",
		"fulfil", "fulfill",
		"fulfilment", "fulfillment",
		"initialise", "initialize",
		"judgment", "judgement",
		"labelling", "labeling",
		"labour", "labor",
		"licence", "license",
		"maximise", "maximize",
		"modelled", "modeled",
		"modelling", "modeling",
		"offence", "offense",
		"optimise", "optimize",
		"organisation", "organization",
		"organise", "organize",
		"practise", "practice",
		"programme", "program",
		"realise", "realize",
		"recognise", "recognize",
		"signalling", "signaling",
		"sub-license", "sublicense",
		"sub license", "sub-license",
		"utilisation", "utilization",
		"whilst", "while",
		"wilful", "wilfull",
		"non-commercial", "noncommercial",
		"per cent", "percent",
		"copyright owner", "copyright",
	)

	// 9.1.1 "©", "(c)", or "Copyright" should be considered equivalent and interchangeable.
	copyrightRe = regexp.MustCompile("©|\\(c\\)|copyright")
	trademarkRe = regexp.MustCompile("™|\\(tm\\)|trademark")

	// extra cleanup
	brokenLinkRe = regexp.MustCompile("http s ://")
	urlCleanupRe = regexp.MustCompile("[<(](http(s?)://[^\\s]+)[)>]")
)

// NormalizeLicenseText makes a license text ready for analysis.
// It follows SPDX guidelines at
// https://spdx.org/spdx-license-list/matching-guidelines
func NormalizeLicenseText(text string, strict bool) string {
	// Line endings
	text = lineEndingsRe.ReplaceAllString(text, "\n")

	// 3. Whitespace
	text = whitespaceRe.ReplaceAllString(text, " ")
	text = trailingWhitespaceRe.ReplaceAllString(text, "")
	text = leadingWhitespaceRe.ReplaceAllString(text, "")

	// 4. Capitalization
	text = strings.ToLower(text)

	// 5. Punctuation
	text = punctuationRe.ReplaceAllString(text, "-")
	text = quotesRe.ReplaceAllString(text, "\"")

	// 7. Bullets and Numbering
	text = bulletRe.ReplaceAllString(text, "")

	// 8. Varietal Word Spelling
	text = wordReplacer.Replace(text)

	// 9. Copyright Symbol
	text = copyrightRe.ReplaceAllString(text, "©")
	text = trademarkRe.ReplaceAllString(text, "™")

	// fix broken URLs in SPDX source texts
	text = brokenLinkRe.ReplaceAllString(text, "https://")

	// fix URLs in <> - erase the decoration
	text = urlCleanupRe.ReplaceAllString(text, "$1")

	if !strict {
		// there are common mismatches because of trailing dots
		text = strings.Replace(text, ".", "", -1)
	}

	return text
}
