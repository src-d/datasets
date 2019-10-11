package pga

const (
	uastHeaderURL = iota
	uastHeaderFilenames
	uastHeaderFileCount
	uastHeaderSize
	uastHeaderFileExtractionRate
	uastHeaderByteExtractionRate
	uastHeaderLangs
	uastHeaderLangsFileCount
	uastHeaderLangsByteCount
	uastHeaderLangsFileExtractionRate
	uastHeaderLangsByteExtractionRate
)

var uastCSVHeaders = []string{
	uastHeaderURL:                     "URL",
	uastHeaderFilenames:               "PARQUET_FILENAMES",
	uastHeaderFileCount:               "FILE_COUNT",
	uastHeaderSize:                    "SIZE",
	uastHeaderFileExtractionRate:      "FILE_EXTRACT_RATE",
	uastHeaderByteExtractionRate:      "BYTE_EXTRACT_RATE",
	uastHeaderLangs:                   "LANGS",
	uastHeaderLangsFileCount:          "LANGS_FILE_COUNT",
	uastHeaderLangsByteCount:          "LANGS_BYTE_COUNT",
	uastHeaderLangsFileExtractionRate: "LANGS_FILE_EXTRACT_RATE",
	uastHeaderLangsByteExtractionRate: "LANGS_BYTE_EXTRACT_RATE",
}

// UastRepository contains the data from a row of the CSV index
type UastRepository struct {
	URL              string   `json:"url"`              // URL of the repository.
	ParquetFilenames []string `json:"parquetFilenames"` // Parquet filenames.
	Size             int64    `json:"size"`             // Sum of the files sizes.

	// Stats per language
	Languages                   []string  `json:"langs"`                // Languages found in the repository.
	LanguagesFileCount          []int64   `json:"langsFileCount"`       // Number of files in the language in the same index.
	LanguagesByteCount          []int64   `json:"langsByteCount"`       // Number of bytes for the language in same index.
	LanguagesFileExtractionRate []float64 `json:"langsFileExtractRate"` // Ratio of files extracted and converted to UAST in the repository per language.
	LanguagesByteExtractionRate []float64 `json:"langsByteExtractRate"` // Ratio of bytes extracted and converted to UAST in the repository per language.

	// Global stats
	Files              int64   `json:"fileCount"`       // Number of files in the repository.
	FileExtractionRate float64 `json:"fileExtractRate"` // Ratio of files extracted and converted to UAST in the repository.
	ByteExtractionRate float64 `json:"byteExtractRate"` // Ratio of bytes extracted and converted to UAST in the repository.
}

// ToCSV returns a slice of strings corresponding to the CSV representation of the repository.
func (r *UastRepository) ToCSV() []string {
	return []string{
		uastHeaderURL:                     r.URL,
		uastHeaderFilenames:               formatStringList(r.ParquetFilenames),
		uastHeaderFileCount:               formatInt(r.Files),
		uastHeaderSize:                    formatInt(r.Size),
		uastHeaderFileExtractionRate:      formatFloat(r.FileExtractionRate),
		uastHeaderByteExtractionRate:      formatFloat(r.ByteExtractionRate),
		uastHeaderLangs:                   formatStringList(r.Languages),
		uastHeaderLangsFileCount:          formatIntList(r.LanguagesFileCount),
		uastHeaderLangsByteCount:          formatIntList(r.LanguagesByteCount),
		uastHeaderLangsFileExtractionRate: formatFloatList(r.LanguagesFileExtractionRate),
		uastHeaderLangsByteExtractionRate: formatFloatList(r.LanguagesByteExtractionRate),
	}
}

// GetURL returns the string corresponding to the URL of the repository.
func (r *UastRepository) GetURL() string {
	return r.URL
}

// GetLanguages returns a slice of strings corresponding to the languages found in the repository.
func (r *UastRepository) GetLanguages() []string {
	return r.Languages
}

// GetFilenames returns a slice of strings corresponding to the filenames found in the repository.
func (r *UastRepository) GetFilenames() []string {
	return r.ParquetFilenames
}

// UastDataset provides iteration over the SivaRepositories.
type UastDataset struct{}

// Name returns the name of the dataset
func (UastDataset) Name() string {
	return "uast"
}

// ReadHeader reads the header of the CSV index.
func (dataset *UastDataset) ReadHeader(columnNames []string) error {
	length := len(columnNames)
	expected := len(uastCSVHeaders)
	if length != expected {
		return &badHeaderLengthError{
			length:      length,
			expectedMin: expected,
			expectedMax: expected,
		}
	}

	for i, h := range columnNames {
		if h != uastCSVHeaders[i] {
			return &badHeaderColumnError{expected: uastCSVHeaders[i], index: i, col: h}
		}
	}
	return nil
}

// RepositoryFromTuple returns a UastRepository from a slice of strings corresponding to it's CSV representation.
func (dataset *UastDataset) RepositoryFromTuple(cols []string) (repo Repository, err error) {
	p := parser{cols: cols, csvHeaders: &uastCSVHeaders}
	return &UastRepository{
		URL:                         p.readString(uastHeaderURL),
		ParquetFilenames:            p.readStringList(uastHeaderFilenames),
		Files:                       p.readInt(uastHeaderFileCount),
		Size:                        p.readInt(uastHeaderSize),
		FileExtractionRate:          p.readFloat(uastHeaderFileExtractionRate),
		ByteExtractionRate:          p.readFloat(uastHeaderByteExtractionRate),
		Languages:                   p.readStringList(uastHeaderLangs),
		LanguagesFileCount:          p.readIntList(uastHeaderLangsFileCount),
		LanguagesByteCount:          p.readIntList(uastHeaderLangsByteCount),
		LanguagesFileExtractionRate: p.readFloatList(uastHeaderLangsFileExtractionRate),
		LanguagesByteExtractionRate: p.readFloatList(uastHeaderLangsByteExtractionRate),
	}, p.err
}
