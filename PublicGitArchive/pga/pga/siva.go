package pga

const (
	sivaHeaderURL = iota
	sivaHeaderFilenames
	sivaHeaderFileCount
	sivaHeaderLangs
	sivaHeaderLangsByteCount
	sivaHeaderLangsLinesCount
	sivaHeaderLangsFilesCount
	sivaHeaderCommitsCount
	sivaHeaderBranchesCount
	sivaHeaderForkCount
	sivaHeaderEmptyLinesCount
	sivaHeaderCodeLinesCount
	sivaHeaderCommentLinesCount
	sivaHeaderLicense
	sivaHeaderStars
	sivaHeaderSize
)

var sivaCSVHeaders = []string{
	sivaHeaderURL:               "URL",
	sivaHeaderFilenames:         "SIVA_FILENAMES",
	sivaHeaderFileCount:         "FILE_COUNT",
	sivaHeaderLangs:             "LANGS",
	sivaHeaderLangsByteCount:    "LANGS_BYTE_COUNT",
	sivaHeaderLangsLinesCount:   "LANGS_LINES_COUNT",
	sivaHeaderLangsFilesCount:   "LANGS_FILES_COUNT",
	sivaHeaderCommitsCount:      "COMMITS_COUNT",
	sivaHeaderBranchesCount:     "BRANCHES_COUNT",
	sivaHeaderForkCount:         "FORK_COUNT",
	sivaHeaderEmptyLinesCount:   "EMPTY_LINES_COUNT",
	sivaHeaderCodeLinesCount:    "CODE_LINES_COUNT",
	sivaHeaderCommentLinesCount: "COMMENT_LINES_COUNT",
	sivaHeaderLicense:           "LICENSE",
	sivaHeaderStars:             "STARS",
	sivaHeaderSize:              "SIZE",
}

// SivaRepository contains the data from a row of the CSV index
type SivaRepository struct {
	URL           string   `json:"url"`           // URL of the repository.
	SivaFilenames []string `json:"sivaFilenames"` // Siva filenames.
	Size          int64    `json:"size"`          // Sum of the siva files sizes.
	License       string   `json:"license"`       // Main license name of the repository.

	// Stats per language
	Languages             []string `json:"langs"`             // Languages found in the repository.
	LanguagesByteCount    []int64  `json:"langsByteCount"`    // Number of bytes for the language in same index.
	LanguagesLineCount    []int64  `json:"langsLinesCount"`   // Number of lines of code for the language in same index.
	LanguagesFileCount    []int64  `json:"langsFilesCount"`   // Number of files in the language in the same index.
	LanguagesEmptyLines   []int64  `json:"emptyLinesCount"`   // Number of blank lines for the language in the same index
	LanguagesCodeLines    []int64  `json:"codeLinesCount"`    // Number of lines of code for the language in the same index.
	LanguagesCommentLines []int64  `json:"commentLinesCount"` // Number of comment lines for the language in the same index.

	// Global stats
	Files    int64 `json:"fileCount"`     // Number of files in the repository.
	Commits  int64 `json:"commitsCount"`  // Number of commits in the repository.
	Branches int64 `json:"branchesCount"` // Number of branches in the repository.
	Forks    int64 `json:"forkCount"`     // Number of forks of this repository.
	Stars    int64 `json:"stars"`         // Number of stars of this repository.
}

// ToCSV returns a slice of strings corresponding to the CSV representation of the repository.
func (r *SivaRepository) ToCSV() []string {
	return []string{
		sivaHeaderURL:               r.URL,
		sivaHeaderFilenames:         formatStringList(r.SivaFilenames),
		sivaHeaderFileCount:         formatInt(r.Files),
		sivaHeaderLangs:             formatStringList(r.Languages),
		sivaHeaderLangsByteCount:    formatIntList(r.LanguagesByteCount),
		sivaHeaderLangsLinesCount:   formatIntList(r.LanguagesLineCount),
		sivaHeaderLangsFilesCount:   formatIntList(r.LanguagesFileCount),
		sivaHeaderCommitsCount:      formatInt(r.Commits),
		sivaHeaderBranchesCount:     formatInt(r.Branches),
		sivaHeaderForkCount:         formatInt(r.Forks),
		sivaHeaderEmptyLinesCount:   formatIntList(r.LanguagesEmptyLines),
		sivaHeaderCodeLinesCount:    formatIntList(r.LanguagesCodeLines),
		sivaHeaderCommentLinesCount: formatIntList(r.LanguagesCommentLines),
		sivaHeaderLicense:           r.License,
		sivaHeaderStars:             formatInt(r.Stars),
		sivaHeaderSize:              formatInt(r.Size),
	}
}

// GetURL returns the string corresponding to the URL of the repository.
func (r *SivaRepository) GetURL() string {
	return r.URL
}

// GetLanguages returns a slice of strings corresponding to the languages found in the repository.
func (r *SivaRepository) GetLanguages() []string {
	return r.Languages
}

// GetFilenames returns a slice of strings corresponding to the filenames found in the repository.
func (r *SivaRepository) GetFilenames() []string {
	return r.SivaFilenames
}

// RepositoryFromTuple returns a SivaRepository from a slice of strings corresponding to it's CSV representation.
func (dataset *SivaDataset) RepositoryFromTuple(cols []string) (repo Repository, err error) {
	if !dataset.hasStars {
		cols = append(cols, "-1")
	}
	if !dataset.hasSize {
		cols = append(cols, "-1")
	}
	p := parser{cols: cols, csvHeaders: &sivaCSVHeaders}
	return &SivaRepository{
		URL:                   p.readString(sivaHeaderURL),
		SivaFilenames:         p.readStringList(sivaHeaderFilenames),
		Files:                 p.readInt(sivaHeaderFileCount),
		Languages:             p.readStringList(sivaHeaderLangs),
		LanguagesByteCount:    p.readIntList(sivaHeaderLangsByteCount),
		LanguagesLineCount:    p.readIntList(sivaHeaderLangsLinesCount),
		LanguagesFileCount:    p.readIntList(sivaHeaderLangsFilesCount),
		Commits:               p.readInt(sivaHeaderCommitsCount),
		Branches:              p.readInt(sivaHeaderBranchesCount),
		Forks:                 p.readInt(sivaHeaderForkCount),
		LanguagesEmptyLines:   p.readIntList(sivaHeaderEmptyLinesCount),
		LanguagesCodeLines:    p.readIntList(sivaHeaderCodeLinesCount),
		LanguagesCommentLines: p.readIntList(sivaHeaderCommentLinesCount),
		License:               cols[sivaHeaderLicense],
		Stars:                 p.readInt(sivaHeaderStars),
		Size:                  p.readInt(sivaHeaderSize),
	}, p.err
}

// SivaDataset provides iteration over the SivaRepositories.
type SivaDataset struct {
	hasStars bool
	hasSize  bool
}

// Name returns the name of the dataset.
func (SivaDataset) Name() string {
	return "siva"
}

// ReadHeader reads the header of the CSV index (including legacy indexes from v1).
func (dataset *SivaDataset) ReadHeader(columnNames []string) error {
	length := len(columnNames)
	expected := len(sivaCSVHeaders)
	if length < expected-2 || length > expected {
		return &badHeaderLengthError{
			length:      length,
			expectedMin: expected - 2,
			expectedMax: expected,
		}
	}

	for i, h := range columnNames {
		if h != sivaCSVHeaders[i] {
			return &badHeaderColumnError{expected: sivaCSVHeaders[i], index: i, col: h}
		}

		switch h {
		case sivaCSVHeaders[sivaHeaderStars]:
			dataset.hasStars = true
		case sivaCSVHeaders[sivaHeaderSize]:
			dataset.hasSize = true
		}
	}
	return nil
}
