// Package adblock is in charge of parsing adblock filter fists and deciding
// whether or not to block URIs.
package adblock

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

const (
	Script uint = 1 << iota
	Image
	StyleSheet
	Object
	XMLHTTPRequest
	ObjectSubrequest
	Subdocument
	Document
	Elemhide
	Other
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

	elemHideRuleMap map[string][]*ElemHideRule
}

// NewBlocker creates a new ad blocker.
func NewBlocker(dir string) *Blocker {
	b := &Blocker{
		make(map[[8]byte][]*BlockRule, 1000),
		make([]*BlockRule, 10),
		make(map[string][]*ElemHideRule, 1000),
	}
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
					runtime.Gosched()
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

// DomainElemHideCSS returns the css string to hide the elements on a given
// domain.
func (b *Blocker) DomainElemHideCSS(domain string) string {
	superdomains := strings.Split(domain, ".")
	for i := range superdomains {
		superdomains[i] = strings.Join(superdomains[i:], ".")
	}
	superdomains = append(superdomains, "")

	var selectors []string
	exemptSelectors := make(map[string]bool)
	for _, superdomain := range superdomains {
		for _, rule := range b.elemHideRuleMap[superdomain] {
			switch rule.RuleType {
			case RuleTypeBlock:
				selectors = append(selectors, rule.cssSelector)
			case RuleTypeException:
				exemptSelectors[rule.cssSelector] = true
			}
		}
	}

	trueSelectors := make([]string, 0, len(selectors))
	for _, selector := range selectors {
		if exemptSelectors[selector] {
			continue
		}
		trueSelectors = append(trueSelectors, selector)
	}

	size := len("{display:none!important;}")
	for _, selector := range trueSelectors {
		size += len(selector)
	}
	size += len(trueSelectors) - 1

	css := make([]byte, 0, size)
	first := true
	for _, selector := range selectors {
		if first {
			first = false
		} else {
			css = append(css, ',')
		}
		css = append(css, []byte(selector)...)
	}
	css = append(css, []byte("{display:none!important;}")...)

	return string(css)
}

// Blocks checks if a specific uri is blocked or not.
func (b *Blocker) Blocks(uri string, flags uint) bool {
	blocked := false
	exception := false
	for _, candidate := range candidateSubstrings([]byte(uri)) {
		for _, rule := range b.blockRuleMap[candidate] {
			if blocked && exception {
				return false
			}
			if !(!blocked && rule.RuleType == RuleTypeBlock) &&
				!(!exception && rule.RuleType == RuleTypeException) {

				continue
			}
			if !rule.MatchString(uri) {
				continue
			}
			if (flags&rule.EnableFlags) == 0 || (flags&rule.DisableFlags) != 0 {
				continue
			}
			if !blocked && rule.RuleType == RuleTypeBlock && rule.MatchString(uri) {
				blocked = true
			}
			if !exception && rule.RuleType == RuleTypeException && rule.MatchString(uri) {
				exception = true
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
	if strings.Contains(string(line), "#@#") ||
		strings.Contains(string(line), "##") {

		rules, err := NewElemHideRules(line)
		if err == nil {
			for _, rule := range rules {
				b.addRule(rule, line)
			}
		}
	} else {
		rule, err := NewBlockRule(line)
		if err == nil {
			b.addRule(rule, line)
		}
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
	case *ElemHideRule:
		rules := b.elemHideRuleMap[rule.domain]
		if rules == nil {
			b.elemHideRuleMap[rule.domain] = []*ElemHideRule{rule}
		} else {
			b.elemHideRuleMap[rule.domain] = append(rules, rule)
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

// A ElemHideRule filters a single element on a single domain.
//
// If domain is empty, all domains are filtered.
type ElemHideRule struct {
	domain      string
	cssSelector string
	RuleType
}

// isRule adherence to the Rule interface.
func (r *ElemHideRule) isRule() {}

// NewElemHideRules creates new rules for hiding elements from a given line.
func NewElemHideRules(rule []byte) ([]*ElemHideRule, error) {
	ruleType := RuleTypeBlock
	var split []string
	if strings.Contains(string(rule), "#@#") {
		ruleType = RuleTypeException
		split = strings.SplitN(string(rule), "#@#", 2)
	} else {
		split = strings.SplitN(string(rule), "##", 2)
	}
	if len(split) != 2 {
		return nil, fmt.Errorf(
			"Failed parsing element hide rule: '%s'",
			string(rule))
	}
	domainsSplit := strings.Split(strings.ToLower(split[0]), ",")
	rules := make([]*ElemHideRule, 0, len(domainsSplit))
	for _, domain := range domainsSplit {
		domainRuleType := ruleType
		// correctly invert.
		if len(domain) != 0 && domain[0] == '~' {
			if ruleType == RuleTypeBlock {
				domainRuleType = RuleTypeException
			} else {
				domainRuleType = RuleTypeBlock
			}
			domain = domain[1:]
		}
		rules = append(rules, &ElemHideRule{
			domain,
			split[1],
			domainRuleType,
		})
	}
	return rules, nil
}

// A BlockRule is a single filter in the filterlist.
type BlockRule struct {
	*regexp.Regexp
	RuleType
	IsSimple     bool
	ThirdParty   *bool
	EnableFlags  uint
	DisableFlags uint
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

	split := strings.SplitN(string(rule), "$", 2)
	rule = []byte(split[0])
	var thirdParty *bool = nil
	matchCase := false
	var enableFlags uint = 0
	var disableFlags uint = 0
	if len(split) == 2 {
		options := split[1]
		split = strings.Split(options, ",")
		for _, option := range split {
			flagPtr := &enableFlags
			if option[0] == '~' {
				flagPtr = &disableFlags
				option = option[1:]
			}
			switch option {
			case "script":
				*flagPtr |= Script
			case "image":
				*flagPtr |= Image
			case "stylesheet":
				*flagPtr |= StyleSheet
			case "object":
				*flagPtr |= Object
			case "xmlhttprequest":
				*flagPtr |= XMLHTTPRequest
			case "object-subrequest":
				*flagPtr |= ObjectSubrequest
			case "subdocument":
				*flagPtr |= Subdocument
			case "document":
				*flagPtr |= Document
			case "elemhide":
				*flagPtr |= Elemhide
			case "other":
				*flagPtr |= Other
			case "third-party":
				thirdParty = new(bool)
				*thirdParty = flagPtr == &enableFlags
			case "match-case":
				matchCase = true
			}
		}
	}
	if enableFlags == 0 {
		enableFlags = ^uint(0)
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
		if !matchCase {
			reg += `(?i)`
		}
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
		if !matchCase {
			r, err = regexp.Compile(`(?i)` + string(rule))
		} else {
			r, err = regexp.Compile(string(rule))
		}
	}
	if err != nil {
		return nil, err
	}
	return &BlockRule{r, rt, simple, thirdParty, enableFlags, disableFlags}, nil
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
