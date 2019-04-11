package indexer

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/erizocosmico/gocloc"
	"github.com/sirupsen/logrus"
	"gopkg.in/src-d/core-retrieval.v0/model"
	"gopkg.in/src-d/core-retrieval.v0/repository"
	"gopkg.in/src-d/enry.v1"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-license-detector.v2/licensedb"
	"gopkg.in/src-d/go-license-detector.v2/licensedb/filer"
)

type repositoryData struct {
	URL         string
	SivaFiles   []string
	Size        int64
	Files       int
	Languages   map[string]language
	HEADCommits int64
	Commits     int64
	Branches    int
	Forks       int
	License     map[string]float32
	Stars       uint32
}

func (r repositoryData) toRecord() []string {
	var (
		langs            []string
		langBytes        = make([]string, len(r.Languages))
		langLines        = make([]string, len(r.Languages))
		langFiles        = make([]string, len(r.Languages))
		langEmptyLines   = make([]string, len(r.Languages))
		langCodeLines    = make([]string, len(r.Languages))
		langCommentLines = make([]string, len(r.Languages))
	)

	for lang := range r.Languages {
		langs = append(langs, lang)
	}
	sort.Strings(langs)

	for i, lang := range langs {
		l := r.Languages[lang]
		langBytes[i] = fmt.Sprint(l.Usage.Bytes)
		langFiles[i] = fmt.Sprint(l.Usage.Files)
		langLines[i] = fmt.Sprint(l.Usage.Lines)
		langEmptyLines[i] = fmt.Sprint(l.Lines.Blank)
		langCodeLines[i] = fmt.Sprint(l.Lines.Code)
		langCommentLines[i] = fmt.Sprint(l.Lines.Comments)
	}

	var licenseNames []string
	for lic := range r.License {
		licenseNames = append(licenseNames, lic)
	}
	sort.Strings(licenseNames)

	var licenses = make([]string, len(licenseNames))
	for i, name := range licenseNames {
		licenses[i] = fmt.Sprintf("%s:%.3f", name, r.License[name])
	}

	return []string{
		r.URL,                     // "URL"
		join(r.SivaFiles),         // "SIVA_FILENAMES"
		fmt.Sprint(r.Files),       // "FILE_COUNT"
		join(langs),               // "LANGS"
		join(langBytes),           // "LANGS_BYTE_COUNT"
		join(langLines),           // "LANGS_LINES_COUNT"
		join(langFiles),           // "LANGS_FILES_COUNT"
		fmt.Sprint(r.HEADCommits), // "COMMITS_COUNT"
		fmt.Sprint(r.Branches),    // "BRANCHES_COUNT"
		fmt.Sprint(r.Forks),       // "FORK_COUNT"
		join(langEmptyLines),      // "EMPTY_LINES_COUNT"
		join(langCodeLines),       // "CODE_LINES_COUNT"
		join(langCommentLines),    // "COMMENT_LINES_COUNT"
		join(licenses),            // "LICENSE"
		fmt.Sprint(r.Stars),       // "STARS"
		fmt.Sprint(r.Size),        // "SIZE"
	}
}

func join(strs []string) string {
	return strings.Join(strs, ",")
}

var csvHeader = []string{
	"URL",
	"SIVA_FILENAMES",
	"FILE_COUNT",
	"LANGS",
	"LANGS_BYTE_COUNT",
	"LANGS_LINES_COUNT",
	"LANGS_FILES_COUNT",
	"COMMITS_COUNT",
	"BRANCHES_COUNT",
	"FORK_COUNT",
	"EMPTY_LINES_COUNT",
	"CODE_LINES_COUNT",
	"COMMENT_LINES_COUNT",
	"LICENSE",
	"STARS",
	"SIZE",
}

type language struct {
	Lines lineCounts
	Usage languageUsage
}

type lineCounts struct {
	Blank    int64
	Code     int64
	Comments int64
}

func processRepos(
	workers int,
	txer repository.RootedTransactioner,
	rs *model.RepositoryResultSet,
	stars map[string]uint32,
) <-chan *repositoryData {
	logrus.WithField("workers", runtime.NumCPU()).Info("start processing repos")
	start := time.Now()
	defer func() {
		logrus.WithField("elapsed", time.Since(start)).Debug("finished processing repos")
	}()

	ws := newWorkerSet(workers)
	ch := make(chan *repositoryData)
	locker := newLocker()

	go func() {
		var wg sync.WaitGroup
		logrus.Debug("start processing")

		for rs.Next() {
			repo, err := rs.Get()
			if err != nil {
				logrus.WithField("err", err).Error("unable to get next repository")
				continue
			}

			wg.Add(1)
			ws.do(func() {
				defer wg.Done()
				log := logrus.WithField("repo", repo.ID)
				log.Debug("starting worker")
				defer log.Debug("stopping worker")

				data, err := newProcessor(repo, txer, locker, stars).process()
				if err == errNoHEAD {
					log.WithField("repo", repo.ID).Warn("empty repository")
					ch <- &repositoryData{
						URL:       getRepoURL(repo),
						License:   make(map[string]float32),
						Languages: make(map[string]language),
					}
				} else if err != nil {
					log.WithField("err", err).Error("unable to process repository")
				} else {
					ch <- data
				}
			})
		}

		wg.Wait()
		close(ch)
		logrus.Debug("finished processing")
	}()

	return ch
}

type processor struct {
	repo   *git.Repository
	dbRepo *model.Repository
	txer   repository.RootedTransactioner
	locker *locker
	stars  map[string]uint32
}

func newProcessor(
	dbRepo *model.Repository,
	txer repository.RootedTransactioner,
	locker *locker,
	stars map[string]uint32,
) *processor {
	return &processor{
		dbRepo: dbRepo,
		txer:   txer,
		locker: locker,
		stars:  stars,
	}
}

var errNoHEAD = errors.New("repository has no HEAD")

func (p *processor) process() (*repositoryData, error) {
	log := logrus.WithField("repo", p.dbRepo.ID)
	log.Debug("start processing repository")
	start := time.Now()
	defer func() {
		log.WithField("elapsed", time.Since(start)).Debug("finished processing repository")
	}()

	var inits = make(map[model.SHA1]struct{})
	var empty model.SHA1
	var head model.SHA1
	for _, ref := range p.dbRepo.References {
		if ref.Name == "refs/heads/HEAD" {
			head = ref.Init
		}

		inits[ref.Init] = struct{}{}
	}

	if head == empty {
		return nil, errNoHEAD
	}

	mut := p.locker.lock(head.String())
	mut.Lock()
	tx, err := p.txer.Begin(context.TODO(), plumbing.NewHash(head.String()))
	if err != nil {
		mut.Unlock()
		return nil, fmt.Errorf("can't start transaction: %s", err)
	}

	p.repo, err = git.Open(tx.Storer(), nil)
	if err != nil {
		mut.Unlock()
		return nil, fmt.Errorf("can't open git repo: %s", err)
	}

	data, err := p.data()
	if err != nil {
		mut.Unlock()
		return nil, fmt.Errorf("unable to get repo data: %s", err)
	}

	mut.Unlock()
	_ = tx.Rollback()

	log = log.WithField("url", data.URL)
	var sumOfSivaSizes int64
	for init := range inits {
		log.WithField("init", init.String()).Debug("processing init")
		mut := p.locker.lock(init.String())
		mut.Lock()
		err := func() error {
			defer mut.Unlock()
			tx, err := p.txer.Begin(context.TODO(), plumbing.NewHash(init.String()))
			if err != nil {
				return fmt.Errorf("can't get root transaction: %s", err)
			}
			defer tx.Rollback()

			r, err := git.Open(tx.Storer(), nil)
			if err != nil {
				return fmt.Errorf("can't open root repo: %s", err)
			}

			size, err := sivaSize(init.String())
			if err != nil {
				panic(err)
			}
			sumOfSivaSizes += size

			iter, err := r.CommitObjects()
			if err != nil {
				return fmt.Errorf("can't get root commits: %s", err)
			}

			n, err := countCommits(iter)
			if err != nil {
				return fmt.Errorf("can't count root commits: %s", err)
			}

			id, err := p.repoID()
			if err != nil {
				return err
			}

			refs, err := r.References()
			if err != nil {
				return fmt.Errorf("can't get references: %s", err)
			}

			var refCount int
			err = refs.ForEach(func(ref *plumbing.Reference) error {
				if strings.HasSuffix(string(ref.Name()), "/"+id) {
					refCount++
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("unable to count refs: %s", err)
			}

			data.Branches += refCount
			data.Commits += n

			return nil
		}()

		log.WithField("init", init.String()).Debug("finished processing init")

		if err != nil {
			return nil, err
		}
	}

	data.SivaFiles = sivaFiles(inits)
	data.Size = sumOfSivaSizes

	return data, nil
}

func getRepoURL(repo *model.Repository) string {
	if len(repo.Endpoints) == 0 {
		return ""
	}

	url := repo.Endpoints[0]
	// initialize to first github url, if any
	for _, e := range repo.Endpoints {
		if strings.Contains(e, "github.com") {
			url = e
			break
		}
	}

	return url
}

func (p *processor) data() (*repositoryData, error) {
	log := logrus.WithField("repo", p.dbRepo.ID)
	log.Debug("start building repo data")
	start := time.Now()
	defer func() {
		log.WithField("elapsed", time.Since(start)).Debug("finished building repo data")
	}()

	var data repositoryData
	data.URL = getRepoURL(p.dbRepo)

	head, err := p.head()
	if err != nil {
		return nil, fmt.Errorf("unable to get HEAD ref: %s", err)
	}

	files, err := p.headFiles(head)
	if err != nil {
		return nil, fmt.Errorf("unable to get head files: %s", err)
	}
	data.Files = len(files)

	usage, err := p.languageUsage(files)
	if err != nil {
		return nil, fmt.Errorf("unable to get lang usage: %s", err)
	}

	path, err := writeToTempDir(files)
	if err != nil {
		return nil, fmt.Errorf("unable to write files to temp dir: %s", err)
	}

	defer func() {
		if err := os.RemoveAll(path); err != nil {
			logrus.WithField("dir", path).Error("unable to remove temp dir")
		}
	}()

	lines, err := p.lineCounts(path, files)
	if err != nil {
		return nil, err
	}

	data.Languages = mergeLanguageData(usage, lines)

	data.HEADCommits, err = p.headCommits(head)
	if err != nil {
		return nil, fmt.Errorf("unable to get head commits: %s", err)
	}

	loader, err := filer.FromDirectory(path)
	if err != nil {
		return nil, fmt.Errorf("unable to read %s: %s", path, err)
	}
	data.License, err = licensedb.Detect(loader)
	if err != nil {
		data.License = make(map[string]float32)
		logrus.WithField("repo", data.URL).WithField("err", err).
			Warn("unable to get license for repository")
	}

	repo := trimRepoURL(data.URL)
	data.Stars = p.stars[repo]

	return &data, nil
}

func trimRepoURL(url string) string {
	const (
		HTTPprefix = "https://github.com/"
		SSHprefix  = "git://github.com/"
		suffix     = ".git"
	)

	var repo string
	if strings.HasPrefix(url, HTTPprefix) {
		repo = strings.TrimPrefix(url, HTTPprefix)
	} else if strings.HasPrefix(url, SSHprefix) {
		repo = strings.TrimPrefix(url, SSHprefix)
		repo = strings.TrimSuffix(repo, suffix)
	}

	return repo
}

func (p *processor) head() (*plumbing.Reference, error) {
	id, err := p.repoID()
	if err != nil {
		return nil, err
	}

	return p.repo.Reference(plumbing.ReferenceName("refs/heads/HEAD/"+id), true)
}

func (p *processor) repoID() (string, error) {
	cfg, err := p.repo.Config()
	if err != nil {
		return "", fmt.Errorf("unable to get config: %s", err)
	}

	var target string
Outer:
	for id, r := range cfg.Remotes {
		for _, u := range r.URLs {
			for _, e := range p.dbRepo.Endpoints {
				if u == e {
					target = id
					break Outer
				}
			}
		}
	}

	if target == "" {
		return "", fmt.Errorf("unable to guess the repository from config for repo: %s", p.dbRepo.ID)
	}

	return target, nil
}

func sivaFiles(inits map[model.SHA1]struct{}) []string {
	var files []string
	for init := range inits {
		files = append(files, fmt.Sprintf("%s.siva", init))
	}
	sort.Strings(files)
	return files
}

// regex to match directory names composed by sha1_timestamp as in
// 7a80dfe1684664cefd2923bdbb329dcb9a48dc4f_1551878586555302343
var regSivaDir = regexp.MustCompile(`\b([0-9a-f]{40})_[0-9]{19}\b`)

func sivaSize(init string) (int64, error) {
	info, err := ioutil.ReadDir("/tmp/sourced")
	if err != nil {
		return -1, err
	}

	if len(info) != 1 || !info[0].IsDir() {
		return -1, fmt.Errorf("/tmp/sourced directory wasn't in a clean status")
	}

	tmp := filepath.Join("/tmp/sourced", info[0].Name(), "transactioner")
	info, err = ioutil.ReadDir(tmp)
	if err != nil {
		return -1, err
	}

	var paths []string
	for _, fi := range info {
		if !fi.IsDir() {
			continue
		}

		matches := regSivaDir.FindStringSubmatch(fi.Name())
		if len(matches) != 2 {
			continue
		}

		if init == matches[1] {
			path := filepath.Join(tmp, fi.Name(), "siva")
			// the same siva file could be duplicated under
			// different directories if there are several
			// repositories being analyzed concurrently
			paths = append(paths, path)
		}
	}

	if len(paths) == 0 {
		return -1, fmt.Errorf(
			"couldn't find siva file for SHA1 %s in %s",
			init, tmp)

	}

	var size int64 = -1
	for _, path := range paths {
		fi, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				// a previous found siva file generated from
				// another repository analysis could have been
				// already removed
				continue
			}

			return -1, err
		}

		if size < fi.Size() {
			size = fi.Size()
		}
	}

	if size < 0 {
		return -1, fmt.Errorf(
			"couldn't find siva file for SHA1 %s in %s",
			init, tmp)
	}

	return size, nil
}

func mergeLanguageData(
	usage map[string]languageUsage,
	counts map[string]lineCounts,
) map[string]language {
	var result = make(map[string]language)

	for lang, usage := range usage {
		count := counts[lang]
		result[lang] = language{Lines: count, Usage: usage}
	}

	return result
}

func (p *processor) lineCounts(path string, files []*object.File) (map[string]lineCounts, error) {
	logrus.Debug("start building line counts")
	start := time.Now()
	defer func() {
		logrus.WithField("elapsed", time.Since(start)).Debug("finished building line counts")
	}()

	proc := gocloc.NewProcessor(gocloc.NewDefinedLanguages(), gocloc.NewClocOptions())

	var paths = make([]string, len(files))
	for i, f := range files {
		paths[i] = filepath.Join(path, f.Name)
	}

	result, err := proc.Analyze(paths)
	if err != nil {
		return nil, fmt.Errorf("can't analyze files: %s", err)
	}

	lcounts := make(map[string]lineCounts)
	for lang, counts := range result.Languages {
		lcounts[lang] = lineCounts{
			Blank:    int64(counts.Blanks),
			Code:     int64(counts.Code),
			Comments: int64(counts.Comments),
		}
	}

	return lcounts, nil
}

func (p *processor) headCommits(head *plumbing.Reference) (int64, error) {
	logrus.Debug("start counting HEAD commits")
	start := time.Now()
	defer func() {
		logrus.WithField("elapsed", time.Since(start)).Debug("finished counting HEAD commits")
	}()

	ci, err := p.repo.Log(&git.LogOptions{From: head.Hash()})
	if err != nil {
		return -1, fmt.Errorf("can't get HEAD log (head is %s): %s", head.Hash(), err)
	}

	return countCommits(ci)
}

func countCommits(iter object.CommitIter) (count int64, err error) {
	err = iter.ForEach(func(_ *object.Commit) error {
		count++
		return nil
	})
	return
}

func (p *processor) branches() ([]string, error) {
	logrus.Debug("start counting branches")
	start := time.Now()
	defer func() {
		logrus.WithField("elapsed", time.Since(start)).Debug("finished counting branches")
	}()

	ri, err := p.repo.References()
	if err != nil {
		return nil, fmt.Errorf("can't get repo references: %s", err)
	}

	var refs []string
	err = ri.ForEach(func(ref *plumbing.Reference) error {
		if !ref.Name().IsTag() {
			refs = append(refs, ref.Name().String())
		}
		return nil
	})
	return refs, err
}

type languageUsage struct {
	Files int64
	Bytes int64
	Lines int64
}

func (p *processor) languageUsage(files []*object.File) (map[string]languageUsage, error) {
	logrus.Debug("start building language report")
	start := time.Now()
	defer func() {
		logrus.WithField("elapsed", time.Since(start)).Debug("finished building language report")
	}()

	var idx = make(map[string]languageUsage)

	for _, f := range files {
		content, err := f.Contents()
		if err != nil {
			return nil, fmt.Errorf("can't get file contents: %s", err)
		}

		lang := enry.GetLanguage(f.Name, []byte(content))
		if lang == "" {
			continue
		}

		bytes := len(content)
		lines := len(strings.Split(content, "\n"))

		report := idx[lang]
		report.Files++
		report.Bytes += int64(bytes)
		report.Lines += int64(lines)
		idx[lang] = report
	}

	return idx, nil
}

func (p *processor) headFiles(head *plumbing.Reference) ([]*object.File, error) {
	logrus.Debug("start getting files of HEAD")
	start := time.Now()
	defer func() {
		logrus.WithField("elapsed", time.Since(start)).Debug("finished getting files of HEAD")
	}()

	ci, err := p.repo.Log(&git.LogOptions{From: head.Hash()})
	if err != nil {
		return nil, fmt.Errorf("unable to get HEAD log (head is %s): %s", head.Hash(), err)
	}

	commit, err := ci.Next()
	if err != nil {
		return nil, fmt.Errorf("can't get first commit in HEAD: %s", err)
	}
	ci.Close()

	fi, err := commit.Files()
	if err != nil {
		return nil, fmt.Errorf("can't get commit files: %s", err)
	}

	var files []*object.File
	err = fi.ForEach(func(f *object.File) error {
		files = append(files, f)
		return nil
	})
	return files, err
}

func writeToTempDir(files []*object.File) (base string, err error) {
	base, err = ioutil.TempDir(os.TempDir(), "borges-indexer")
	if err != nil {
		return "", fmt.Errorf("unable to create temp dir: %s", err)
	}

	defer func() {
		if err != nil {
			if removeErr := os.RemoveAll(base); removeErr != nil {
				logrus.Errorf("unable to remove temp dir after error (%s): %s", removeErr, err)
			}
		}
	}()

	for _, f := range files {
		path := filepath.Join(base, f.Name)
		if err = os.MkdirAll(filepath.Dir(path), 0744); err != nil {
			return "", err
		}

		var content string
		content, err = f.Contents()
		if err != nil {
			return "", err
		}

		err = ioutil.WriteFile(path, []byte(content), 0644)
		if err != nil {
			return "", err
		}
	}

	return base, nil
}
