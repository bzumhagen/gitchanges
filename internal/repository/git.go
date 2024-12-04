package repository

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/bzumhagen/gitchanges/internal/changelog"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type GitRepository struct {
	pathToRoot string
	name       string
}

func NewGitRepository(pathToRoot string, maybeName *string) *GitRepository {
	var name string
	if maybeName == nil || *maybeName == "" {
		name = filepath.Base(filepath.Dir(pathToRoot))
		caser := cases.Title(language.AmericanEnglish)
		name = caser.String(name)
	} else {
		name = *maybeName
	}
	return &GitRepository{
		pathToRoot: pathToRoot,
		name:       name,
	}
}

func (r *GitRepository) Name() string {
	return r.name
}

func (r *GitRepository) TraverseHistory(f func(c changelog.Commit) error) error {
	fs := osfs.New(r.pathToRoot)
	c := cache.NewObjectLRUDefault()
	bfs := filesystem.NewStorage(fs, c)
	repo, err := git.Open(bfs, fs)
	if err != nil {
		return fmt.Errorf("failed to open repository at path %s: %w", r.pathToRoot, err)
	}

	tagToRef := make(map[string]string, 0)
	tIter, err := repo.Tags()
	if err != nil {
		return fmt.Errorf("failed to retrieve repository tags: %w", err)
	}

	err = tIter.ForEach(func(r *plumbing.Reference) error {
		var tagName string
		var commitHash string
		obj, iterErr := repo.TagObject(r.Hash())
		switch iterErr {
		case nil:
			// Tag object present (i.e. is annotated tag)
			tagName = r.Name().Short()
			commitHash = obj.Target.String()
		case plumbing.ErrObjectNotFound:
			// Not a tag object (i.e. is lightweight tag)
			tagName = r.Name().Short()
			commitHash = r.Hash().String()
		default:
			return iterErr
		}
		tagToRef[commitHash] = tagName
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to iterate repository tags: %w", err)
	}

	cIter, err := repo.Log(&git.LogOptions{})
	if err != nil {
		return fmt.Errorf("failed to access commit history: %w", err)
	}
	err = cIter.ForEach(func(c *object.Commit) error {
		cc := changelog.Commit{
			Message: c.Message,
			Date:    c.Committer.When.Format(time.DateOnly),
		}
		tag, found := tagToRef[c.Hash.String()]
		if found {
			cc.Tag = &tag
		}
		return f(cc)
	})
	if err != nil {
		return fmt.Errorf("failed to iterate commit history: %w", err)
	}
	return nil
}
