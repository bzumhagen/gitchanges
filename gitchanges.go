package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/blang/semver"
	"github.com/bzumhagen/gitchanges/version"
	"github.com/hoisie/mustache"
	"github.com/spf13/viper"
	"gopkg.in/src-d/go-billy.v4/osfs"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/storage/filesystem"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const ungroupedTemplate = `
# {{name}} Change Log
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).

{{#changeGroups}}
## [{{Version}}] - {{Date}}
{{#TaggedChanges}}
### {{Tag}}
{{#Changes}}
- {{#showReference}}{{FmtReference}}{{/showReference}}{{Description}}
{{/Changes}}
{{/TaggedChanges}}
***
{{/changeGroups}}
`

type Change struct {
	Description string
	Reference   string
	Version     semver.Version
	Tag         string
	When        time.Time
}

type TaggedChanges struct {
	Tag string
	Changes []Change
}
type ChangeGroup struct {
	Version semver.Version
	TaggedChanges []TaggedChanges
	When        time.Time
}

func (c ChangeGroup) Date() string {
	year, month, day := c.When.Date()
	return strconv.Itoa(year) + "-" + strconv.Itoa(int(month)) + "-" + strconv.Itoa(day)
}

func (c Change) FmtReference() string {
	if len(c.Reference) > 0 {
		return c.Reference
	} else {
		return ""
	}
}

func main() {
	args := os.Args[1:]
	repoPtr := flag.String("repo", ".", "Path to git repository")
	configPtr := flag.String("config", "changelog.yaml", "Path to configuration file")
	//startVersionPtr := flag.String("start", "", "Start version. If not specified, will use all versions")

	flag.Parse()
	readConfig(configPtr)

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

func readConfig(path *string) {
	viper.SetConfigFile(*path)
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}
	viper.SetDefault("sct.name", "Project")
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
	changes := buildChanges(start, itr)
	groups := groupChanges(&changes)
	viper.GetString("sct.name")

	data := mustache.Render(
		ungroupedTemplate,
		map[string][]ChangeGroup{"changeGroups": groups},
		map[string]string{"name": viper.GetString("sct.name")},
		map[string]bool{"showReference": false},
	)

	ioutil.WriteFile("changelog.md", []byte(data), 444)
	return nil
}

func groupChanges(changes *[]Change) []ChangeGroup {
	var groups []ChangeGroup
	versionToChanges := make(map[string][]Change)

	for _, change := range *changes {
		v := change.Version.String()
		versionToChanges[v] = append(versionToChanges[v], change)
	}
	for v, changes := range versionToChanges {
		var taggedChanges []TaggedChanges
		tagToChanges := make(map[string][]Change)

		for _, c := range changes {
			tagToChanges[c.Tag] = append(tagToChanges[c.Tag], c)
		}
		for t, changes := range tagToChanges {
			tc := TaggedChanges{
				Tag: t,
				Changes: changes,
			}
			taggedChanges = append(taggedChanges, tc)
		}
		sort.Slice(taggedChanges, func(i, j int) bool {
			return taggedChanges[i].Tag < taggedChanges[j].Tag
		})
		cg := ChangeGroup{
			Version: semver.MustParse(v),
			TaggedChanges: taggedChanges,
			When: changes[0].When,
		}
		groups = append(groups, cg)
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Version.GT(groups[j].Version)
	})
	return groups
}

func buildChanges(start semver.Version, iter object.CommitIter) []Change {
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
					When:        c.Author.When,
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
