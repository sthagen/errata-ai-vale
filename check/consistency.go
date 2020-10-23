package check

import (
	"fmt"
	"strings"

	"github.com/errata-ai/vale/config"
	"github.com/errata-ai/vale/core"
	"github.com/jdkato/regexp"
	"github.com/mitchellh/mapstructure"
)

type step struct {
	pattern *regexp.Regexp
	subs    []string
}

// Consistency ensures that the keys and values of Either don't both exist.
type Consistency struct {
	Definition `mapstructure:",squash"`
	// `nonword` (`bool`): Removes the default word boundaries (`\b`).
	Nonword bool
	// `ignorecase` (`bool`): Makes all matches case-insensitive.
	Ignorecase bool
	// `either` (`map`): A map of `option 1: option 2` pairs, of which only one
	// may appear.
	Either map[string]string

	steps []step
}

// NewConsistency ...
func NewConsistency(cfg *config.Config, generic baseCheck) (Consistency, error) {
	var chkRE string

	rule := Consistency{}
	path := generic["path"].(string)
	name := generic["name"].(string)

	err := mapstructure.Decode(generic, &rule)
	if err != nil {
		return rule, readStructureError(err, path)
	}

	regex := makeRegexp(
		cfg.WordTemplate,
		rule.Ignorecase,
		func() bool { return !rule.Nonword },
		func() string { return "" }, true)

	chkKey := strings.Split(name, ".")[1]
	count := 0
	for v1, v2 := range rule.Either {
		count += 2
		subs := []string{
			fmt.Sprintf("%s%d", chkKey, count), fmt.Sprintf("%s%d", chkKey, count+1)}

		chkRE = fmt.Sprintf("(?P<%s>%s)|(?P<%s>%s)", subs[0], v1, subs[1], v2)
		chkRE = fmt.Sprintf(regex, chkRE)

		re, err := regexp.Compile(chkRE)
		if err != nil {
			return rule, core.NewE201FromPosition(err.Error(), path, 1)
		}

		rule.Extends = name
		rule.Name = fmt.Sprintf("%s.%s", name, v1)
		rule.steps = append(rule.steps, step{pattern: re, subs: subs})
	}

	return rule, nil
}

// Run ...
func (o Consistency) Run(txt string, f *core.File) []core.Alert {
	alerts := []core.Alert{}
	loc := []int{}

	for _, s := range o.steps {
		matches := s.pattern.FindAllStringSubmatchIndex(txt, -1)
		for _, submat := range matches {
			for idx, mat := range submat {
				if mat != -1 && idx > 0 && idx%2 == 0 {
					loc = []int{mat, submat[idx+1]}
					f.Sequences = append(f.Sequences, s.pattern.SubexpNames()[idx/2])
				}
			}
		}

		if matches != nil && core.AllStringsInSlice(s.subs, f.Sequences) {
			o.Name = o.Extends
			alerts = append(alerts, makeAlert(o.Definition, loc, txt))
		}
	}

	return alerts
}

// Fields ...
func (o Consistency) Fields() Definition {
	return o.Definition
}

// Pattern ...
func (o Consistency) Pattern() string {
	return ""
}