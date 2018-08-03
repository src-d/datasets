package ld

import (
	"regexp"

	"github.com/jdkato/prose/chunk"
	"github.com/jdkato/prose/tag"
	"github.com/jdkato/prose/tokenize"
	"strings"
)

var (
	licenseMarkReadmeRe = regexp.MustCompile("([Cc]opy(right|ing))|\\(c\\)|©|([Ll]icen[cs][ei])|released under")
	garbageReadmeRe     = regexp.MustCompile("([Cc]opy(right|ing))|\\(c\\)|©")
	licenseReadmeRe     = regexp.MustCompile("\\s*[Ll]icen[cs]e\\s*")
	licenseNamePartRe   = regexp.MustCompile("([a-z]+)|([0-9]+)")
	digitsRe            = regexp.MustCompile("[0-9]+")
)

// investigateReadmeFile is the implementation of InvestigateReadmeText.
// It takes two additional arguments: licenseNameParts and licenseNameSizes.
// The idea is to map substrings to real licenses, and the confidence is
// <the number of matches> / <overall number of substrings>.
func investigateReadmeFile(
	text string, licenseNameParts map[string][]substring,
	licenseNameSizes map[string]int) map[string]float32 {
	matches := licenseMarkReadmeRe.FindAllStringIndex(text, -1)
	if len(matches) == 0 {
		return map[string]float32{}
	}
	beginIndex := matches[0][0]
	endIndex := beginIndex + 50
	if len(matches) > 1 {
		endIndex = matches[len(matches)-1][1]
	} else {
		beginIndex -= 50
		if beginIndex < 0 {
			beginIndex = 0
		} else {
			for ; text[beginIndex] != ' ' && text[beginIndex] != '\t' &&
				text[beginIndex] != '\n' && beginIndex < matches[0][0]; beginIndex++ {
			}
		}
		for ; endIndex < len(text) && text[endIndex] != ' ' && text[endIndex] != '\t' &&
			text[endIndex] != '\n'; endIndex++ {
		}
	}
	if endIndex > len(text) {
		endIndex = len(text)
	}
	suspectedText := text[beginIndex:endIndex]
	suspectedWords := tokenize.TextToWords(suspectedText)
	tagger := tag.NewPerceptronTagger()
	candidates := map[string]float32{}
	for _, entity := range chunk.Chunk(tagger.Tag(suspectedWords), chunk.TreebankNamedEntities) {
		if garbageReadmeRe.MatchString(entity) {
			continue
		}
		scores := map[string]int{}
		entity = licenseReadmeRe.ReplaceAllString(entity, "")
		substrs := splitLicenseName(entity)
		for _, substr := range substrs {
			for _, match := range licenseNameParts[substr.Value] {
				common := match.Count
				if substr.Count < common {
					common = substr.Count
				}
				scores[match.Value] += common
			}
		}
		// if the only reason a license matched is a single digit, drop it
		for _, substr := range substrs {
			if digitsRe.MatchString(substr.Value) && len(substr.Value) == 1 {
				for _, match := range licenseNameParts[substr.Value] {
					if scores[match.Value] == 1 {
						delete(scores, match.Value)
					}
				}
			}
		}
		for key, val := range scores {
			confidence := float32(val) / float32(licenseNameSizes[key])
			if candidates[key] < confidence {
				candidates[key] = confidence
			}
		}
	}
	return candidates
}

func splitLicenseName(name string) []substring {
	counts := map[string]int{}
	parts := licenseNamePartRe.FindAllString(strings.ToLower(name), -1)
	for i, part := range parts {
		if part[len(part)-1] == 'v' && i < len(parts)-1 && digitsRe.MatchString(parts[i+1]) {
			part = part[:len(part)-1]
			if len(part) == 0 {
				continue
			}
		}
		if part == "clause" {
			continue
		}
		// BSD hack
		if part == "simplified" {
			part = "2"
		}
		counts[part]++
	}
	result := make([]substring, len(counts))
	i := 0
	for key, val := range counts {
		result[i] = substring{Value: key, Count: val}
		i++
	}
	return result
}
