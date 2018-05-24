package hangulize

import (
	"regexp"
	"strings"
)

// <...>
//  └─┴─ (1)
var reVar = regex(`<(.+?)>`)

// expandVars replaces <var> to corresponding content regexp such as (a|b|c).
func expandVars(expr string, vars map[string][]string) string {
	return reVar.ReplaceAllStringFunc(expr, func(varExpr string) string {
		// Retrieve variable name and values.
		name, vals := getVar(varExpr, vars)

		// Build as RegExp like /(a|b|c)/.
		escapedVals := make([]string, len(vals))
		for i, val := range vals {
			escapedVals[i] = regexp.QuoteMeta(val)
		}

		// return `(?P<` + name + `>` + strings.Join(escapedVals, `|`) + `)`
		_ = name
		return `(?:` + strings.Join(escapedVals, `|`) + `)`
	})
}

func getVar(expr string, vars map[string][]string) (string, []string) {
	// expr is the matched string.  It looks like "<var>".
	name := strings.Trim(expr, `<>`)
	vals := vars[name]

	return name, vals
}
