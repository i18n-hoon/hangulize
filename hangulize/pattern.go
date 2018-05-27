package hangulize

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

// Pattern represents an HRE (Hangulize-specific Regular Expression) pattern.
// It is used for the rewrite of Hangulize.
//
// Some expressions in Pattern have special meaning:
//
// - "^" - start of chunk
// - "^^" - start of string
// - "$" - end of chunk
// - "$$" - end of string
// - "{...}" - zero-width match
// - "{~...}" - zero-width negative match
// - "<var>" - one of var values (defined in spec)
//
type Pattern struct {
	expr string

	re  *regexp.Regexp // positive regexp
	neg *regexp.Regexp // negative regexp

	// Letters used in the positive/negative regexps.
	letters []string

	// References to expanded vars.
	usedVars [][]string
}

func (p *Pattern) String() string {
	return fmt.Sprintf(`/%s/`, p.expr)
}

// NewPattern compiles an HRE pattern from an expression.
func NewPattern(
	expr string,

	macros map[string]string,
	vars map[string][]string,

) (*Pattern, error) {

	reExpr := expr

	reExpr = expandMacros(reExpr, macros)

	reExpr, usedVars := expandVars(reExpr, vars)

	reExpr, negExpr, err := expandLookaround(reExpr)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile pattern: %#v", expr)
	}

	reExpr = expandEdges(reExpr)

	// Collect letters in the regexps.
	letters := make([]string, 0)
	for _, ch := range regexpLetters(reExpr + negExpr) {
		letters = append(letters, string(ch))
	}
	letters = set(letters)

	// Compile regexp.
	re, err := regexp.Compile(reExpr)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile pattern: %#v", expr)
	}

	neg, err := regexp.Compile(negExpr)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile pattern: %#v", expr)
	}

	p := &Pattern{expr, re, neg, letters, usedVars}
	return p, nil
}

// Explain shows the HRE expression with
// the underlying standard regexp patterns.
func (p *Pattern) Explain() string {
	if p == nil {
		return fmt.Sprintf("%#v", nil)
	}
	return fmt.Sprintf("expr:/%s/, re:/%s/, neg:/%s/", p.expr, p.re, p.neg)
}

// -----------------------------------------------------------------------------

// Find searches up to n matches in the word.
func (p *Pattern) Find(word string, n int) [][]int {
	matches := make([][]int, 0)
	offset := 0

	for n < 0 || len(matches) < n {
		// Erase visited characters on the word with "\x00".  Because of
		// lookaround, the search cursor should be calculated manually.
		erased := strings.Repeat(".", offset) + word[offset:]

		m := p.re.FindStringSubmatchIndex(erased)

		if len(m) == 0 || m[1]-m[0] == 0 {
			// No more match.
			break
		}

		// p.re looks like (edge)(look)abc(look)(edge).
		// Hold only non-zero-width matches.
		if len(m) < 10 {
			panic(fmt.Errorf("unexpected submatches: %v", m))
		}

		start := m[5]
		if start == -1 {
			start = m[0]
		}
		stop := m[len(m)-4]
		if stop == -1 {
			stop = m[1]
		}

		// Pick matched word.  Call it "highlight".
		highlight := erased[m[0]:m[1]]

		// Test highlight with p.neg to determine whether skip or not.
		negM := p.neg.FindStringSubmatchIndex(highlight)

		// If no negative match, this match is successful.
		if len(negM) == 0 {
			match := []int{start, stop}

			// Keep content ()...
			match = append(match, m[6:len(m)-4]...)

			matches = append(matches, match)
		}

		// Shift the cursor.
		if len(negM) == 0 {
			offset = stop
		} else {
			offset = m[0] + negM[1]
		}
	}

	return matches
}

// Replace searches up to n matches in the word and replaces them with the
// RPattern list.
func (p *Pattern) Replace(word string, rpatterns []*RPattern, n int) []string {
	var buf strings.Builder
	offset := 0

	for _, m := range p.Find(word, n) {
		start, stop := m[0], m[1]

		buf.WriteString(word[offset:start])

		// TODO(sublee): Support multiple targets.
		rp := rpatterns[0]

		fmt.Println(start, stop, rp)

		// Write replacement instead of the match.
		buf.WriteString(rp.Interpolate(p, word, m))

		offset = stop
	}

	buf.WriteString(word[offset:])

	return []string{buf.String()}
}
