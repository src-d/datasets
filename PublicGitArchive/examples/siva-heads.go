package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	sivafs "gopkg.in/src-d/go-billy-siva.v4"
	"gopkg.in/src-d/go-billy.v4/memfs"
	"gopkg.in/src-d/go-billy.v4/osfs"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/cache"
	"gopkg.in/src-d/go-git.v4/plumbing/filemode"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/storage/filesystem"
)

func readSiva(path string) (map[string]*object.Commit, error) {
	localFs := osfs.New(filepath.Dir(path))
	tmpFs := memfs.New()
	basePath := filepath.Base(path)
	fs, err := sivafs.NewFilesystem(localFs, basePath, tmpFs)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to create a Siva filesystem from %s", path)
	}
	sivaStorage := filesystem.NewStorage(fs, cache.NewObjectLRUDefault())
	if sivaStorage == nil {
		return nil, fmt.Errorf("unable to create a new storage backend for Siva file %s", path)
	}
	repo, err := git.Open(sivaStorage, tmpFs)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to open the Git repository from Siva file %s", path)
	}
	log.Print("Reading the references, this make take some time... ")
	refs, err := repo.References()
	log.Println("done.")
	if err != nil {
		return nil, errors.Wrapf(err, "unable to list Git references from Siva file %s", path)
	}
	commits := map[string]*object.Commit{}
	err = refs.ForEach(func(ref *plumbing.Reference) error {
		refname := ref.Name()
		if strings.HasPrefix(refname.String(), "refs/heads/HEAD/") {
			commit, err := repo.CommitObject(ref.Hash())
			if err != nil {
				return errors.Wrapf(err, "failed to load %s in Siva file %s", ref.Hash().String(), path)
			}
			commits[string(refname[len("refs/heads/HEAD/"):])] = commit
		}
		return nil
	})
	return commits, err
}

func dumpFiles(commit *object.Commit, destDir string) error {
	tree, err := commit.Tree()
	if err != nil {
		return errors.Wrapf(err, "could not read the tree from %s", commit.Hash.String())
	}
	err = tree.Files().ForEach(func(file *object.File) (res error) {
		destPath := filepath.Join(destDir, file.Name)
		switch file.Mode {
		case filemode.Dir:
			return os.MkdirAll(destPath, 0777)
		case filemode.Regular:
			baseDir := filepath.Dir(destPath)
			err = os.MkdirAll(baseDir, 0777)
			if err != nil {
				return errors.Wrapf(err, "failed to create directory %s", baseDir)
			}
			writer, err := os.Create(destPath)
			if err != nil {
				return errors.Wrapf(err, "failed to create %s", destPath)
			}
			defer func() {
				err = writer.Close()
				if err != nil {
					res = errors.Wrapf(err, "failed to close %s", destPath)
				}
			}()
			reader, err := file.Reader()
			if err != nil {
				return errors.Wrapf(err, "failed to read %s %s", file.Name, file.Hash.String())
			}
			defer reader.Close()
			written, err := io.Copy(writer, reader)
			if err != nil {
				return errors.Wrapf(err, "failed to write %s - only %d bytes were written from %s",
					destPath, written, file.Hash.String())
			}
			return nil
		default:
			return nil
		}
	})
	return err
}

func main() {
	if len(os.Args) != 3 {
		log.Fatalln("Usage: siva-heads input.siva output/directory")
	}
	commits, err := readSiva(os.Args[1])
	if err != nil {
		log.Fatalln(err)
	}
	for key, commit := range commits {
		err = dumpFiles(commit, filepath.Join(os.Args[2], key))
		if err != nil {
			log.Fatalln(err)
		}
	}
}
