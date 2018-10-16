package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/blang/semver"
	"github.com/bzumhagen/gitchanges/version"
	"github.com/spf13/viper"
	"gopkg.in/src-d/go-billy.v4/osfs"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/storage/filesystem"
	"log"
	"os"
	"regexp"
	"strings"
)

type Change struct {
	Description string
	Reference   string
	Version     semver.Version
	Tag         string
}

func main() {
	args := os.Args[1:]
	repoPtr := flag.String("repo", ".", "Path to git repository")
	configPtr := flag.String("config", "changelog.yaml", "Path to configuration file")
	//startVersionPtr := flag.String("start", "", "Start version. If not specified, will use all versions")

	flag.Parse()
	viper.SetConfigFile(*configPtr)
	err := viper.ReadInConfig()

	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

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

	start := semver.MustParse("0.0.0")
	changes := getChanges(start, itr)

	fmt.Printf("%v", changes)
	return nil
}

func getChanges(start semver.Version, iter object.CommitIter) []Change {
	var changes []Change
	vr := regexp.MustCompile("[\\s\\S]*\nversion: (.+)[\\s\\S]*")
	tr := regexp.MustCompile("[\\s\\S]*\ntag: (.+)[\\s\\S]*")
	rr := regexp.MustCompile("[\\s\\S]*\nreference: (.+)[\\s\\S]*")

	iter.ForEach(func(c *object.Commit) error {
		r := bufio.NewReader(strings.NewReader(c.Message))
		desc, err := r.ReadString('\n')
		if err != nil {
			return err
		}

		desc = strings.TrimRight(desc, "\n")
		versionMatch := vr.FindStringSubmatch(c.Message)
		tagMatch := tr.FindStringSubmatch(c.Message)

		if versionMatch != nil && tagMatch != nil {
			var reference, tag string
			parsedVersion, err := semver.Parse(versionMatch[1])
			if err != nil {
				return err
			}

			if parsedVersion.GTE(start) {
				tag = tagMatch[1]

				referenceMatch := rr.FindStringSubmatch(c.Message)
				if referenceMatch != nil {
					reference = referenceMatch[1]
				}

				change := Change{
					Description: desc,
					Reference:   reference,
					Version:     parsedVersion,
					Tag:         tag,
				}
				changes = append(changes, change)
			}
		}
		return nil
	})

	return changes
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
