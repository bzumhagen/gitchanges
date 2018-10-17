package main

import (
	"github.com/blang/semver"
	"github.com/magiconair/properties/assert"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"testing"
	"time"
)

func TestGetRepository(t *testing.T) {
	_, err := getRepository(".")
	if err != nil {
		t.Fail()
	}

	_, err = getRepository("some-path")
	if err == nil {
		t.Fail()
	}
}

type mockCommitIter struct {
	commits      []object.Commit
	currentIndex int
}

func (m mockCommitIter) Next() (*object.Commit, error) {
	if m.currentIndex < len(m.commits) {
		m.currentIndex++
		return &m.commits[m.currentIndex], nil
	} else {
		return nil, nil
	}
}

func (m mockCommitIter) ForEach(fn func(*object.Commit) error) error {
	var err error
	for i := 0; i < len(m.commits); i++ {
		err = fn(&m.commits[i])
	}
	return err
}

func (m mockCommitIter) Close() {
	m.currentIndex = 0
}

func TestGroupChanges(t *testing.T) {
	version012 := semver.MustParse("0.1.2")
	version011 := semver.MustParse("0.1.1")
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	lastWeek := now.AddDate(0, 0, -7)
	tests := []struct {
		changes         []Change
		expectedResults []ChangeGroup
	}{
		{
			changes: []Change{
				{
					Description: "Commit with version, tag, and reference",
					Version:     version012,
					Reference:   "XYZ-123",
					Tag:         "Added",
					When:        now,
				},
				{
					Description: "Another commit with version, tag, and reference",
					Version:     version012,
					Reference:   "XYZ-123",
					Tag:         "Added",
					When:        yesterday,
				},
				{
					Description: "One more commit with version, tag, and reference",
					Version:     version012,
					Reference:   "XYZ-123",
					Tag:         "Removed",
					When:        lastWeek,
				},
				{
					Description: "Commit with version, tag, but no reference",
					Version:     version011,
					Reference:   "",
					Tag:         "Changed",
					When:        lastWeek,
				},
			},
			expectedResults: []ChangeGroup{
				{
					Version: version012,
					TaggedChanges: []TaggedChanges{
						{
							Tag: "Added",
							Changes: []Change{
								{
									Description: "Commit with version, tag, and reference",
									Version:     version012,
									Reference:   "XYZ-123",
									Tag:         "Added",
									When:        now,
								},
								{
									Description: "Another commit with version, tag, and reference",
									Version:     version012,
									Reference:   "XYZ-123",
									Tag:         "Added",
									When:        yesterday,
								},
							},
						},
						{
							Tag: "Removed",
							Changes: []Change{
								{
									Description: "One more commit with version, tag, and reference",
									Version:     version012,
									Reference:   "XYZ-123",
									Tag:         "Removed",
									When:        lastWeek,
								},
							},
						},
					},
					When: now,
				},
				{
					Version: version011,
					TaggedChanges: []TaggedChanges{
						{
							Tag: "Changed",
							Changes: []Change{
								{
									Description: "Commit with version, tag, but no reference",
									Version:     version011,
									Reference:   "",
									Tag:         "Changed",
									When:        lastWeek,
								},
							},
						},
					},
					When: lastWeek,
				},
			},
		},
		{
			changes:         []Change{},
			expectedResults: []ChangeGroup{},
		},
	}

	for _, test := range tests {
		results := groupChanges(&test.changes)
		if len(test.expectedResults) > 0 {
			assert.Equal(t, test.expectedResults, results)
		} else {
			assert.Equal(t, 0, len(results))
		}
	}
}

func TestBuildChanges(t *testing.T) {
	version000 := semver.MustParse("0.0.0")
	version012 := semver.MustParse("0.1.2")
	version011 := semver.MustParse("0.1.1")
	testAuthor := object.Signature{
		Name:  "Testy Testerson",
		Email: "test@test.com",
		When:  time.Now(),
	}
	tests := []struct {
		start           semver.Version
		commitItr       object.CommitIter
		expectedResults []Change
	}{
		{
			start: version000,
			commitItr: mockCommitIter{
				commits: []object.Commit{
					{
						Message: "Commit with version, tag, and reference\n" +
							"\n" +
							"version: 0.1.2\n" +
							"tag: Added\n" +
							"reference: XYZ-123",
						Author: testAuthor,
					},
					{
						Message: "Commit with version, tag, but no reference\n" +
							"\n" +
							"version: 0.1.1\n" +
							"tag: Changed",
						Author: testAuthor,
					},
					{
						Message: "Commit with version, but no tag or reference\n" +
							"\n" +
							"version: 0.1.0",
						Author: testAuthor,
					},
					{
						Message: "Commit with no version, tag or reference\n" +
							"\n",
						Author: testAuthor,
					},
				},
				currentIndex: 0,
			},
			expectedResults: []Change{
				{
					Description: "Commit with version, tag, and reference",
					Version:     version012,
					Reference:   "XYZ-123",
					Tag:         "Added",
					When:        testAuthor.When,
				},
				{
					Description: "Commit with version, tag, but no reference",
					Version:     version011,
					Reference:   "",
					Tag:         "Changed",
					When:        testAuthor.When,
				},
			},
		},
		{
			start: version012,
			commitItr: mockCommitIter{
				commits: []object.Commit{
					{
						Message: "Commit with version, tag, and reference\n" +
							"\n" +
							"version: 0.1.2\n" +
							"tag: Added\n" +
							"reference: XYZ-123",
						Author: testAuthor,
					},
					{
						Message: "Commit with version, tag, but no reference\n" +
							"\n" +
							"version: 0.1.1\n" +
							"tag: Changed",
						Author: testAuthor,
					},
					{
						Message: "Commit with version, but no tag or reference\n" +
							"\n" +
							"version: 0.1.0",
						Author: testAuthor,
					},
					{
						Message: "Commit with no version, tag or reference\n" +
							"\n",
						Author: testAuthor,
					},
				},
				currentIndex: 0,
			},
			expectedResults: []Change{
				{
					Description: "Commit with version, tag, and reference",
					Version:     version012,
					Reference:   "XYZ-123",
					Tag:         "Added",
					When:        testAuthor.When,
				},
			},
		},
		{
			start: version000,
			commitItr: mockCommitIter{
				commits: []object.Commit{
					{
						Message: "Commit with no version, tag or reference\n" +
							"\n",
					},
				},
				currentIndex: 0,
			},
			expectedResults: []Change{},
		},
		{
			start: version000,
			commitItr: mockCommitIter{
				commits:      []object.Commit{},
				currentIndex: 0,
			},
			expectedResults: []Change{},
		},
	}

	for _, test := range tests {
		results := buildChanges(test.start, test.commitItr)
		if len(test.expectedResults) > 0 {
			assert.Equal(t, test.expectedResults, results)
		} else {
			assert.Equal(t, 0, len(results))
		}
	}
}
