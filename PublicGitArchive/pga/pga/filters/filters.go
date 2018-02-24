// Package filters provides a set of filters useful to narrow the list of
// repositories in Public Git Archive.
package filters

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/src-d/datasets/PublicGitArchive/pga/pga"
)

// And returns a filter that matches a repository only when all of the filters given match.
func And(filters ...pga.Filter) pga.Filter {
	return func(r *pga.Repository) bool {
		for _, filter := range filters {
			if !filter(r) {
				return false
			}
		}
		return true
	}
}

// Or returns a filter that matches a repository only when at least one of the filters given matches.
func Or(filters ...pga.Filter) pga.Filter {
	return func(r *pga.Repository) bool {
		for _, filter := range filters {
			if filter(r) {
				return true
			}
		}
		return false
	}
}

// HasLanguage returns a Filter that matches all repositories with at least the given language.
func HasLanguage(lang string) pga.Filter {
	lang = strings.ToLower(lang)
	return func(r *pga.Repository) bool {
		for _, l := range r.Languages {
			if strings.ToLower(l) == lang {
				return true
			}
		}
		return false
	}
}

// URLRegexp returns a Filter that checks whether the URL of a repo matches the given regular expression.
func URLRegexp(s string) (pga.Filter, error) {
	re, err := regexp.Compile(s)
	if err != nil {
		return nil, fmt.Errorf("could not compile regular expression %q: %v", s, err)
	}
	return func(r *pga.Repository) bool { return re.MatchString(r.URL) }, nil
}
