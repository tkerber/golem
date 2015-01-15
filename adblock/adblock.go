// Package adblock is in charge of parsing adblock filter fists and deciding
// whether or not to block URIs.
package adblock

import (
	"bufio"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// regexpReplacer is a replacer for strings in the basic filter regexps.
var regexpReplacer = strings.NewReplacer(
	`\*`,
	`.*`,
	`\^`,
	`[^a-zA-Z0-9-_.%]`)

// A Blocker is an instance of adblock.
type Blocker struct {
	// We (more or less) use adblock pluses technique for rule matching.
	blockRuleMap       map[[8]byte][]*BlockRule
	trailingBlockRules []*BlockRule
}

// NewBlocker creates a new ad blocker.
func NewBlocker(dir string) *Blocker {
	b := &Blocker{make(map[[8]byte][]*BlockRule, 1000), make([]*BlockRule, 10)}
	go func() {
		err := filepath.Walk(
			dir,
			func(path string, i os.FileInfo, err error) error {
				if err != nil || i.IsDir() {
					return err
				}
				f, err := os.Open(path)
				if err != nil {
					return err
				}
				defer f.Close()
				reader := bufio.NewReader(f)
				var prefix []byte
				for {
					line, isPref, err := reader.ReadLine()
					if err == io.EOF {
						break
					} else if err != nil {
						return err
					}
					if isPref {
						if len(prefix) == 0 {
							prefix = line
						} else {
							prefix = append(prefix, line...)
						}
					} else {
						if len(prefix) != 0 {
							line = append(prefix, line...)
						}
						b.parseLine(line)
					}
				}
				return nil
			})
		if err != nil {
			log.Printf("Failed to read filterlist: %v", err)
		} else {
			log.Printf("Filterlist parsed.")
		}
	}()
	return b
}

// Blocks checks if a specific uri is blocked or not.
func (b *Blocker) Blocks(uri string) bool {
	uri = strings.ToLower(uri)
	blocked := false
	exception := false
	for _, candidate := range candidateSubstrings([]byte(uri)) {
		for _, rule := range b.blockRuleMap[candidate] {
			if !blocked && rule.RuleType == RuleTypeBlock && rule.MatchString(uri) {
				blocked = true
			}
			if !exception && rule.RuleType == RuleTypeException && rule.MatchString(uri) {
				exception = true
			}
			if blocked && exception {
				return false
			}
		}
	}
	return blocked && !exception
}

// parseLine parses a single line of a filterlist and adds it to the blocker.
func (b *Blocker) parseLine(line []byte) {
	// empty or comment line.
	if len(line) == 0 || line[0] == '!' {
		return
	}
	// For now, we don't handle element hiders.
	// For now, we also completely ignore rules with $ signs.
	if strings.Contains(string(line), "##") {
		return
	}
	split := strings.SplitN(string(line), "$", 2)
	line = []byte(split[0])
	// TODO handle $ options.
	line = []byte(strings.ToLower(string(line)))
	rule, err := NewBlockRule(line)
	if err == nil {
		b.addRule(rule, line)
	}
}

// addRule adds a new rule to the blocker.
func (b *Blocker) addRule(rule Rule, src []byte) {
	switch rule := rule.(type) {
	case *BlockRule:
		if !rule.IsSimple {
			b.trailingBlockRules = append(b.trailingBlockRules, rule)
			return
		}
		// Find the key for our rule.
		candidates := candidateSubstrings(src)
		var key [8]byte
		var competing uint
		competing = ^uint(0)
		for _, candidate := range candidates {
			if strings.ContainsAny(string(candidate[:]), "*^") {
				continue
			}
			if competing == 0 {
				break
			}
			c := len(b.blockRuleMap[candidate])
			if uint(c) < competing {
				key = candidate
				competing = uint(c)
			}
		}
		if competing == ^uint(0) {
			b.trailingBlockRules = append(b.trailingBlockRules, rule)
			return
		}
		// Add the rule under the specified key.
		rules := b.blockRuleMap[key]
		if rules == nil {
			b.blockRuleMap[key] = []*BlockRule{rule}
		} else {
			b.blockRuleMap[key] = append(rules, rule)
		}
	}
}

// candidateSubstrings gets all length 8 substrings of a string.
//
// And by string I mean bytearray.
func candidateSubstrings(str []byte) [][8]byte {
	if len(str) < 8 {
		return nil
	}
	ret := make([][8]byte, len(str)-7)
	for i := 0; i <= len(str)-8; i++ {
		copy(ret[i][:], str[i:i+8])
	}
	return ret
}

// A Rule is either a BlockRule or a HideRule.
type Rule interface {
	isRule()
}

// A BlockRule is a single filter in the filterlist.
type BlockRule struct {
	*regexp.Regexp
	RuleType
	IsSimple bool
}

// isRule adherence to the Rule interface.
func (r *BlockRule) isRule() {}

// NewBlockRule creates a new rule from the corresponding line in the filterlist.
func NewBlockRule(rule []byte) (*BlockRule, error) {
	rt := RuleTypeBlock
	if len(rule) == 0 {
		return nil, errors.New("empty rule")
	}
	if len(rule) >= 2 && string(rule[:2]) == "@@" {
		rt = RuleTypeException
		rule = rule[2:]
		if len(rule) == 0 {
			return nil, errors.New("empty rule")
		}
	}
	if strings.Contains(string(rule), "##") || strings.Contains(string(rule), "$") || strings.Contains(string(rule), "||") {
		return nil, errors.New("currently unsupported ruletype")
	}
	simple := true
	if len(rule) >= 2 && rule[0] == '/' && rule[len(rule)-1] == '/' {
		simple = false
		rule = rule[1 : len(rule)-1]
		if len(rule) == 0 {
			return nil, errors.New("empty rule")
		}
	}
	var r *regexp.Regexp
	var err error
	if simple {
		reg := ``
		if len(rule) >= 2 && string(rule[0:2]) == "||" {
			reg += `^[^:]*://`
			rule = rule[2:]
			if len(rule) == 0 {
				return nil, errors.New("empty rule")
			}
		} else if rule[0] == '|' {
			reg += `^`
			rule = rule[1:]
			if len(rule) == 0 {
				return nil, errors.New("empty rule")
			}
		} else {
			reg += `^.*`
		}
		matchEnd := rule[len(rule)-1] == '|'
		if rule[len(rule)-1] == '|' {
			rule = rule[:len(rule)-1]
			if len(rule) == 0 {
				return nil, errors.New("empty rule")
			}
		}
		quot := regexp.QuoteMeta(string(rule))
		reg += regexpReplacer.Replace(quot)
		if matchEnd {
			reg += `$`
		} else {
			reg += `.*$`
		}
		r, err = regexp.Compile(reg)
	} else {
		r, err = regexp.Compile(string(rule))
	}
	if err != nil {
		return nil, err
	}
	return &BlockRule{r, rt, simple}, nil
}

// The RuleType of a Rule is what the rule does when it matches.
type RuleType uint

const (
	// RuleTypeBlock indicates that matching this rule blocks the URI.
	RuleTypeBlock RuleType = iota
	// RuleTypeException indicates that matching this rule exempts the URI
	// from blocking.
	RuleTypeException
)
