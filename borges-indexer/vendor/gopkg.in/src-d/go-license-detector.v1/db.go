package ld

import (
	"archive/tar"
	"bytes"
	"io"
	"math"
	"sort"
	"strings"

	"github.com/ekzhu/minhash-lsh"
	"github.com/sergi/go-diff/diffmatchpatch"
)

type LicenseDatabase struct {
	Debug bool

	licenseTexts map[string]string
	tokens       map[string]int
	docfreqs     []int
	lsh          *minhashlsh.MinhashLSH
	hasher       *WeightedMinHasher
}

const (
	numHashes              = 100
	lshSimilarityThreshold = 0.75
)

func (db LicenseDatabase) Length() int {
	return len(db.licenseTexts)
}

func (db LicenseDatabase) VocabularySize() int {
	return len(db.tokens)
}

func (db *LicenseDatabase) Load() {
	tarBytes, err := Asset("licenses.tar")
	if err != nil {
		panic("failed to load licenses.tar from the assets: " + err.Error())
	}
	tarStream := bytes.NewBuffer(tarBytes)
	archive := tar.NewReader(tarStream)
	db.licenseTexts = map[string]string{}
	for header, err := archive.Next(); err != io.EOF; header, err = archive.Next() {
		if len(header.Name) <= 6 {
			continue
		}
		key := header.Name[2 : len(header.Name)-4]
		text := make([]byte, header.Size)
		readSize, readErr := archive.Read(text)
		if readErr != nil {
			panic("failed to load licenses.tar from the assets: " + header.Name + ": " + readErr.Error())
		}
		if int64(readSize) != header.Size {
			panic("failed to load licenses.tar from the assets: " + header.Name + ": incomplete read")
		}
		db.licenseTexts[key] = NormalizeLicenseText(string(text), false)
	}
	tokenFreqs := map[string]map[string]int{}
	for key, text := range db.licenseTexts {
		lines := strings.Split(text, "\n")
		myUniqueTokens := map[string]int{}
		tokenFreqs[key] = myUniqueTokens
		for _, line := range lines {
			tokens := strings.Split(line, " ")
			for _, token := range tokens {
				myUniqueTokens[token]++
			}
		}
	}
	docfreqs := map[string]int{}
	for _, tokens := range tokenFreqs {
		for token := range tokens {
			docfreqs[token]++
		}
	}
	uniqueTokens := make([]string, len(docfreqs))
	{
		i := 0
		for token := range docfreqs {
			uniqueTokens[i] = token
			i++
		}
	}
	sort.Strings(uniqueTokens)
	db.tokens = map[string]int{}
	db.docfreqs = make([]int, len(uniqueTokens))
	for i, token := range uniqueTokens {
		db.tokens[token] = i
		db.docfreqs[i] = docfreqs[token]
	}
	db.lsh = minhashlsh.NewMinhashLSH64(numHashes, lshSimilarityThreshold)
	if db.Debug {
		k, l := db.lsh.Params()
		println("LSH:", k, l)
	}
	db.hasher = NewWeightedMinHasher(len(uniqueTokens), numHashes, 7)
	for key, tokens := range tokenFreqs {
		indices := make([]int, len(tokens))
		values := make([]float32, len(tokens))
		{
			i := 0
			for t, freq := range tokens {
				indices[i] = db.tokens[t]
				values[i] = tfidf(freq, db.docfreqs[indices[i]], len(db.licenseTexts))
				i++
			}
		}
		db.lsh.Add(key, db.hasher.Hash(values, indices))
	}
	db.lsh.Index()
}

func (db *LicenseDatabase) Query(text string) (options []string, similarities []float32) {
	normalized := NormalizeLicenseText(text, false)
	if db.Debug {
		println(normalized)
	}
	tokens := map[int]int{}
	myRunes := make([]rune, 0, len(normalized)/6)
	oovRune := rune(len(db.tokens))
	for _, line := range strings.Split(normalized, "\n") {
		for _, token := range strings.Split(line, " ") {
			if index, exists := db.tokens[token]; exists {
				tokens[index]++
				myRunes = append(myRunes, rune(index))
			} else if len(myRunes) == 0 || myRunes[len(myRunes)-1] != oovRune {
				myRunes = append(myRunes, oovRune)
			}
		}
	}
	indices := make([]int, len(tokens))
	values := make([]float32, len(tokens))
	{
		i := 0
		for key, val := range tokens {
			indices[i] = key
			values[i] = tfidf(val, db.docfreqs[key], len(db.licenseTexts))
			i++
		}
	}
	found := db.lsh.Query(db.hasher.Hash(values, indices))
	options = make([]string, 0, len(found))
	similarities = make([]float32, 0, len(found))
	if len(found) == 0 {
		return
	}
	for _, keyint := range found {
		key := keyint.(string)
		text := db.licenseTexts[key]
		yourRunes := make([]rune, 0, len(text)/6)
		for _, line := range strings.Split(text, "\n") {
			for _, token := range strings.Split(line, " ") {
				yourRunes = append(yourRunes, rune(db.tokens[token]))
			}
		}
		dmp := diffmatchpatch.New()
		diff := dmp.DiffMainRunes(myRunes, yourRunes, false)

		if db.Debug {
			tokarr := make([]string, len(db.tokens)+1)
			for key, val := range db.tokens {
				tokarr[val] = key
			}
			tokarr[len(db.tokens)] = "!"
			println(dmp.DiffPrettyText(dmp.DiffCharsToLines(diff, tokarr)))
		}

		distance := dmp.DiffLevenshtein(diff)
		options = append(options, key)
		similarities = append(similarities, float32(1)-float32(distance)/float32(len(myRunes)))
	}
	return
}

func tfidf(freq int, docfreq int, ndocs int) float32 {
	return float32(math.Log(1+float64(freq)) * math.Log(float64(ndocs)/float64(docfreq)))
}
