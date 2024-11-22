package changelog

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"text/template"
)

//go:embed templates/singleproject.tmpl
var singleProjectTemplate string

type ChangelogGenerator struct {
}

type Commit struct {
	Message string
	Date    string
	Tag     *string
}

type Change struct {
	Description string
}

type ChangeGroup struct {
	Tag            string
	Date           *string
	LabeledChanges map[string][]Change
}

type Repository interface {
	Name() string
	TraverseHistory(func(c Commit) error) error
}

type GenerateConfig struct {
	SinceTag       string
	UntilTag       string
	GroupByPattern string
	SkipPattern    string
}

func NewChangelogGenerator() *ChangelogGenerator {
	return &ChangelogGenerator{}
}

var errEndTraversal = errors.New("end traversal")

func (g *ChangelogGenerator) Generate(sourceRepo Repository, cfg GenerateConfig) ([]byte, error) {
	tmpl, err := template.New("singleProject").Parse(singleProjectTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to load template %s: %w", "singleProject", err)
	}

	changeGroups, err := buildChangeGroups(sourceRepo, cfg)
	if err != nil {
		return nil, err
	}

	generatedBytes := new(bytes.Buffer)
	var filterDeclaration string
	if cfg.SinceTag != "" || cfg.UntilTag != "" {
		sinceText := "earliest"
		untilText := "latest"
		if cfg.SinceTag != "" {
			sinceText = cfg.SinceTag
		}
		if cfg.UntilTag != "" {
			untilText = cfg.UntilTag
		}
		filterDeclaration = fmt.Sprintf("Changes have been filtered from %s to %s.", sinceText, untilText)
	}
	tmpl.Execute(generatedBytes, SingleProjectTemplateData{
		ProjectName:       sourceRepo.Name(),
		ChangeGroups:      changeGroups,
		FilterDeclaration: filterDeclaration,
	})
	return generatedBytes.Bytes(), nil
}

func buildChangeGroups(sourceRepo Repository, cfg GenerateConfig) ([]ChangeGroup, error) {
	changeGroups := make([]ChangeGroup, 0)
	currentChangeGroup := ChangeGroup{
		Tag:            "Unreleased",
		LabeledChanges: make(map[string][]Change, 0),
	}
	traversalUntilTag := cfg.UntilTag
	var defaultChangeLabel string
	var groupByRegex *regexp.Regexp
	var err error
	if cfg.GroupByPattern != "" {
		groupByRegex, err = regexp.Compile(cfg.GroupByPattern)
		if err != nil {
			return nil, fmt.Errorf("failed to compile groupByPattern %s: %w", cfg.GroupByPattern, err)
		}
		defaultChangeLabel = "Misc"
	}
	var skipRegex *regexp.Regexp
	if cfg.SkipPattern != "" {
		skipRegex, err = regexp.Compile(cfg.SkipPattern)
		if err != nil {
			return nil, fmt.Errorf("failed to compile skipPattern %s: %w", cfg.SkipPattern, err)
		}
	}
	err = sourceRepo.TraverseHistory(func(c Commit) error {
		description := strings.Split(c.Message, "\n")[0]
		if c.Tag != nil {
			if traversalUntilTag != "" && *c.Tag == traversalUntilTag {
				// we reached our target until tag and can now stop skipping commits
				traversalUntilTag = ""
			}
			if cfg.SinceTag != "" && *c.Tag == cfg.SinceTag {
				return errEndTraversal
			}
			if len(currentChangeGroup.LabeledChanges) > 0 {
				changeGroups = append(changeGroups, currentChangeGroup)
			}
			currentChangeGroup = ChangeGroup{
				Tag:            *c.Tag,
				Date:           &c.Date,
				LabeledChanges: make(map[string][]Change, 0),
			}
		}
		if traversalUntilTag != "" && currentChangeGroup.Tag != traversalUntilTag {
			// skip until traversalUntilTag matches current tag
			return nil
		}
		if skipRegex != nil && skipRegex.MatchString(c.Message) {
			return nil
		}
		label := defaultChangeLabel
		if groupByRegex != nil {
			matches := groupByRegex.FindStringSubmatch(c.Message)
			if len(matches) > 1 {
				label = matches[1]
			}
		}
		currentChangeGroup.LabeledChanges[label] = append(currentChangeGroup.LabeledChanges[label], Change{
			Description: description,
		})
		return nil
	})
	if err != nil && !errors.Is(err, errEndTraversal) {
		return nil, fmt.Errorf("failed during history traversal: %w", err)
	}
	if len(currentChangeGroup.LabeledChanges) < 1 {
		currentChangeGroup.LabeledChanges[""] = []Change{{Description: "[All changes in this group have been skipped]"}}
	}
	changeGroups = append(changeGroups, currentChangeGroup)
	return changeGroups, nil
}
