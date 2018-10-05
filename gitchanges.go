package main

import (
    "flag"
    "fmt"
    "github.com/bzumhagen/gitchanges/version"
    "gopkg.in/src-d/go-billy.v4/osfs"
    "gopkg.in/src-d/go-git.v4"
    "gopkg.in/src-d/go-git.v4/plumbing"
    "gopkg.in/src-d/go-git.v4/plumbing/object"
    "gopkg.in/src-d/go-git.v4/storage/filesystem"
    "log"
    "os"
)

func main() {
    args := os.Args[1:]
    repoPtr := flag.String("repo", ".", "Path to git repository")
    //configPtr := flag.String("config", "changelog.conf", "Path to configuration file")
    //startVersionPtr := flag.String("start", "", "Start version. If not specified, will use all versions")

    flag.Parse()

    if len(args) > 0 {
        arg := args[0]
        if arg == "version" {
            fmt.Println(version.VERSION)
        } else {
            err := buildChangelog(*repoPtr)
            if err != nil {
                log.Fatal(err)
            }
        }
    } else {
        err := buildChangelog(*repoPtr)
        if err != nil {
            log.Fatal(err)
        }
    }
}

func buildChangelog(path string) error {
    r, err := getRepository(path)
    if err != nil {
        return err
    }

    itr, err := getCommits(r)
    if err != nil {
        return err
    }

    itr.ForEach(func(c *object.Commit) error {
        fmt.Println(c.Message)
        return nil
    })
    return err
}

func getCommits(repository *git.Repository) (object.CommitIter, error) {
    itr, err := repository.Log(&git.LogOptions{From: plumbing.ZeroHash})
    if err != nil {
        return nil, err
    }
    return itr, nil
}

func getRepository(path string) (*git.Repository, error) {
    fs := osfs.New(path + "/.git")
    st, err := filesystem.NewStorage(fs)
    if err != nil {
        return nil, err
    }
    r, err := git.Open(st, fs)
    if err != nil {
        return nil, err
    }
    return r, nil
}
