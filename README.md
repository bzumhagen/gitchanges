# gitchanges

A tool to generate a markdown changelog for your git repo using tags.

```
Usage of gitchanges:
  -force
        Force output and overwrite existing file
  -groupBy string
        Groups changes by a regex pattern. Defaults to no grouping.
  -name string
        Project name, if different from repository root directory name
  -output string
        File to output changelog to. Defaults to <path>/CHANGELOG.md
  -path string
        Path to a repository to generate a changelog for (default ".")
  -sinceTag string
        Filters results to changes made after this tag. Non-inclusive
  -skip string
        Skips changes by a regex pattern. Defaults to no skipping.
  -untilTag string
        Filters results to changes made at or before this tag. Inclusive
```
