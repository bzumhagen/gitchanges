package main

import (
	"errors"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/bzumhagen/gitchanges/internal/changelog"
	"github.com/bzumhagen/gitchanges/internal/repository"
)

func main() {
	pathFlag := flag.String("path", ".", "Path to a repository to generate a changelog for")
	nameFlag := flag.String("name", "", "Project name, if different from repository root directory name")
	sinceTagFlag := flag.String("sinceTag", "", "Filters results to changes made after this tag. Non-inclusive")
	untilTagFlag := flag.String("untilTag", "", "Filters results to changes made at or before this tag. Inclusive")
	groupByPatternFlag := flag.String("groupBy", "", "Groups changes by a regex pattern. Defaults to no grouping.")
	skipPatternFlag := flag.String("skip", "", "Skips changes by a regex pattern. Defaults to no skipping.")
	outputFilePathFlag := flag.String("output", "", "File to output changelog to. Defaults to <path>/CHANGELOG.md")
	forceWriteFlag := flag.Bool("force", false, "Force output and overwrite existing file")
	flag.Parse()

	var path string
	var err error
	if pathFlag == nil || *pathFlag == "." {
		path, err = os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
	} else {
		path = *pathFlag
	}
	if !strings.HasSuffix(path, ".git") {
		path = filepath.Join(path, ".git")
	}

	repo := repository.NewGitRepository(path, nameFlag)
	clog := changelog.NewChangelogGenerator()
	generatedBytes, err := clog.Generate(repo, changelog.GenerateConfig{
		SinceTag:       *sinceTagFlag,
		UntilTag:       *untilTagFlag,
		GroupByPattern: *groupByPatternFlag,
		SkipPattern:    *skipPatternFlag,
	})
	if err != nil {
		log.Fatal(err)
	}
	outputFilePath := filepath.Join(filepath.Dir(path), "CHANGELOG.md")
	if *outputFilePathFlag != "" {
		outputFilePath = *outputFilePathFlag
	}

	fileExists := true
	_, err = os.Stat(outputFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fileExists = false
		} else {
			log.Fatal(err)
		}
	}
	if !fileExists || *forceWriteFlag {
		os.WriteFile(outputFilePath, generatedBytes, 0666)
	} else {
		log.Fatalf("file %s already exists and force flag was not true", outputFilePath)
	}
	os.Exit(0)
}
