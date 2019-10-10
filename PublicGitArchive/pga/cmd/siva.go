package cmd

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	sivafs "gopkg.in/src-d/go-billy-siva.v4"
	"gopkg.in/src-d/go-billy.v4/memfs"
	"gopkg.in/src-d/go-billy.v4/osfs"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/cache"
	"gopkg.in/src-d/go-git.v4/plumbing/filemode"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/storage/filesystem"
	"gopkg.in/src-d/go-siva.v1/cmd/siva/impl"
)

var actions = map[string]func(flags *pflag.FlagSet) error{
	"unpack": unpack,
	"dump":   dump,
	"list":   list,
}

func unpack(flags *pflag.FlagSet) error {
	var err error
	cmd := &impl.CmdUnpack{Overwrite: true, IgnorePerms: true}
	cmd.Args.File = flags.Arg(1)
	cmd.Output.Path, err = flags.GetString("output")
	if err != nil {
		return err
	}
	cmd.Match, err = flags.GetString("match")
	if err != nil {
		return err
	}
	return cmd.Execute(nil)
}

func dump(flags *pflag.FlagSet) error {
	output, err := flags.GetString("output")
	if err != nil {
		return errors.Wrapf(err, "required command line argument: -o/--output")
	}
	repo, err := loadRepository(flags.Arg(1))
	if err != nil {
		return err
	}
	fmt.Print("Reading the references, this may take some time... ")
	refs, err := repo.References()
	fmt.Println("done.")
	if err != nil {
		return errors.Wrapf(err, "unable to list Git references in %s", flags.Arg(1))
	}
	commits := map[string]*object.Commit{}
	err = refs.ForEach(func(ref *plumbing.Reference) error {
		refname := ref.Name()
		if strings.HasPrefix(refname.String(), "refs/heads/HEAD/") {
			commit, err := repo.CommitObject(ref.Hash())
			if err != nil {
				return errors.Wrapf(err, "failed to load %s in %s", ref.Hash().String(), flags.Arg(1))
			}
			commits[string(refname[len("refs/heads/HEAD/"):])] = commit
		}
		return nil
	})
	for key, commit := range commits {
		err = dumpFiles(commit, filepath.Join(output, key))
		if err != nil {
			return err
		}
	}
	return nil
}

func list(flags *pflag.FlagSet) error {
	repo, err := loadRepository(flags.Arg(1))
	if err != nil {
		return err
	}
	fmt.Print("Reading the references, this may take some time... ")
	refs, err := repo.References()
	fmt.Println("done.")
	if err != nil {
		return errors.Wrapf(err, "unable to list Git references in %s", flags.Arg(1))
	}
	commitIter, err := repo.CommitObjects()
	if err != nil {
		return err
	}
	var commits []*object.Commit
	err = commitIter.ForEach(func(commit *object.Commit) error {
		commits = append(commits, commit)
		return nil
	})
	if err != nil {
		return err
	}
	stdout := os.Stdout
	_, err = fmt.Fprintln(stdout, "{\n\t\"commits\": {")
	if err != nil {
		return err
	}
	for i, c := range commits {
		hashes := make([]string, 0, len(c.ParentHashes))
		for _, h := range c.ParentHashes {
			hashes = append(hashes, "\""+h.String()+"\"")
		}
		_, err = fmt.Fprintf(stdout, "\t\t\"%s\": [%s]", c.Hash.String(), strings.Join(hashes, ", "))
		if err != nil {
			return err
		}
		if i < len(commits)-1 {
			_, err = fmt.Fprintln(stdout, ",")
		} else {
			_, err = fmt.Fprintln(stdout)
		}
		if err != nil {
			return err
		}
	}
	_, err = fmt.Fprintln(stdout, "\t},\n\t\"references\": {")
	if err != nil {
		return err
	}
	var refObjs []*plumbing.Reference
	_ = refs.ForEach(func(ref *plumbing.Reference) error {
		if ref.Hash() != plumbing.ZeroHash {
			refObjs = append(refObjs, ref)
		}
		return nil
	})
	for i, ref := range refObjs {
		_, err = fmt.Fprintf(stdout, "\t\t\"%s\": \"%s\"",
			strings.Replace(ref.Name().String(), "\"", "\\\"", -1), ref.Hash().String())
		if err != nil {
			return err
		}
		if i < len(refObjs)-1 {
			_, err = fmt.Fprintln(stdout, ",")
		} else {
			_, err = fmt.Fprintln(stdout)
		}
		if err != nil {
			return err
		}
	}
	_, err = fmt.Fprintln(stdout, "\t}\n}")
	return err
}

func loadRepository(fileName string) (*git.Repository, error) {
	localFs := osfs.New(filepath.Dir(fileName))
	tmpFs := memfs.New()
	basePath := filepath.Base(fileName)
	fs, err := sivafs.NewFilesystem(localFs, basePath, tmpFs)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to create a siva filesystem from %s", fileName)
	}
	sivaStorage := filesystem.NewStorage(fs, cache.NewObjectLRUDefault())
	if sivaStorage == nil {
		return nil, fmt.Errorf("unable to create a new storage backend from %s", fileName)
	}
	repo, err := git.Open(sivaStorage, tmpFs)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to open the Git repository from %s", fileName)
	}
	return repo, nil
}

func dumpFiles(commit *object.Commit, destDir string) error {
	tree, err := commit.Tree()
	if err != nil {
		return errors.Wrapf(err, "could not read the tree from %s", commit.Hash.String())
	}
	err = tree.Files().ForEach(func(file *object.File) (res error) {
		fmt.Println(path.Join(filepath.Base(destDir), file.Name))
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

// sivaCmd represents the set of commands to work with the siva files: extract revisions, list, dump raw contents.
var sivaCmd = &cobra.Command{
	Use:   "siva",
	Short: "work with siva files",
	Long:  `Unpack, dump specific or HEAD revisions, list revisions.`,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		if cmd.Flags().NArg() != 2 {
			return fmt.Errorf("usage: pga siva <action> /path/to/siva")
		}
		action := cmd.Flags().Arg(0)
		actionFunc, exists := actions[action]
		if !exists {
			knownActions := make([]string, 0, len(actions))
			for k := range actions {
				knownActions = append(knownActions, k)
			}
			sort.Strings(knownActions)
			return fmt.Errorf("unknown action: %s (choose from %s)",
				action, strings.Join(knownActions, ", "))
		}
		return actionFunc(cmd.Flags())
	},
}

func init() {
	RootCmd.AddCommand(sivaCmd)
	flags := sivaCmd.Flags()
	addFilterFlags(flags)
	flags.StringP("match", "m", ".*", "only extract files matching the given regexp")
	flags.StringP("output", "o", ".", "output directory")
}
