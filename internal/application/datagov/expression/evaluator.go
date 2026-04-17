package expression

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Evaluate 对表达式求值
func Evaluate(ctx *Context, expr string) (bool, error) {
	p := &parser{input: expr, pos: 0}
	node, err := p.parseExpr()
	if err != nil {
		return false, err
	}
	p.skipSpaces()
	if p.pos < len(p.input) {
		return false, fmt.Errorf("unexpected character at position %d", p.pos)
	}
	return node.eval(ctx)
}

// --- AST ---

type node interface {
	eval(ctx *Context) (bool, error)
}

type orNode struct{ left, right node }

func (n *orNode) eval(ctx *Context) (bool, error) {
	l, err := n.left.eval(ctx)
	if err != nil {
		return false, err
	}
	if l {
		return true, nil
	}
	return n.right.eval(ctx)
}

type andNode struct{ left, right node }

func (n *andNode) eval(ctx *Context) (bool, error) {
	l, err := n.left.eval(ctx)
	if err != nil || !l {
		return false, err
	}
	return n.right.eval(ctx)
}

type cmpNode struct {
	path  []string
	op    string
	value any
}

func (n *cmpNode) eval(ctx *Context) (bool, error) {
	lv, err := resolvePath(ctx, n.path)
	if err != nil {
		return false, err
	}
	rv := n.value
	if rv == sentinelEmpty {
		rv = ""
	}
	return compare(lv, n.op, rv)
}

func compare(lv any, op string, rv any) (bool, error) {
	switch op {
	case "==":
		return lv == rv, nil
	case "!=":
		return lv != rv, nil
	case ">", ">=", "<", "<=":
		lf, lok := toFloat(lv)
		rf, rok := toFloat(rv)
		if !lok || !rok {
			return false, fmt.Errorf("cannot compare non-numbers: %v %s %v", lv, op, rv)
		}
		switch op {
		case ">":
			return lf > rf, nil
		case ">=":
			return lf >= rf, nil
		case "<":
			return lf < rf, nil
		case "<=":
			return lf <= rf, nil
		}
	case "in":
		arr, ok := rv.([]any)
		if !ok {
			return false, fmt.Errorf("right side of 'in' must be array")
		}
		for _, v := range arr {
			if lv == v {
				return true, nil
			}
		}
		return false, nil
	case "not_in":
		arr, ok := rv.([]any)
		if !ok {
			return false, fmt.Errorf("right side of 'not_in' must be array")
		}
		for _, v := range arr {
			if lv == v {
				return false, nil
			}
		}
		return true, nil
	case "contains":
		ls, ok := lv.(string)
		rs, ok2 := rv.(string)
		if !ok || !ok2 {
			return false, fmt.Errorf("'contains' requires strings")
		}
		return strings.Contains(ls, rs), nil
	case "matches":
		ls, ok := lv.(string)
		rs, ok2 := rv.(string)
		if !ok || !ok2 {
			return false, fmt.Errorf("'matches' requires strings")
		}
		matched, err := regexp.MatchString(rs, ls)
		return matched, err
	}
	return false, fmt.Errorf("unknown operator: %s", op)
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case string:
		f, err := strconv.ParseFloat(n, 64)
		return f, err == nil
	}
	return 0, false
}

var sentinelEmpty = struct{}{}

func resolvePath(ctx *Context, path []string) (any, error) {
	if len(path) == 0 {
		return nil, fmt.Errorf("empty path")
	}
	var v any
	switch path[0] {
	case "tool":
		v = resolveStructFields(ctx.Tool, path[1:])
	case "user":
		v = resolveStructFields(ctx.User, path[1:])
	case "ctx":
		v = resolveStructFields(ctx.Ctx, path[1:])
	default:
		return nil, fmt.Errorf("unknown root: %s", path[0])
	}
	return v, nil
}

func resolveStructFields(s any, fields []string) any {
	v := s
	for _, f := range fields {
		switch val := v.(type) {
		case map[string]any:
			v = val[f]
		default:
			v = getStructField(val, f)
		}
	}
	return v
}

func getStructField(s any, field string) any {
	switch v := s.(type) {
	case ToolContext:
		switch field {
		case "name":
			return v.Name
		case "type":
			return v.Type
		case "category":
			return v.Category
		case "description":
			return v.Description
		case "endpoint":
			return v.Endpoint
		case "status":
			return v.Status
		case "connector_id":
			return v.ConnectorID
		case "config":
			return v.Config
		}
	case UserContext:
		switch field {
		case "id":
			return v.ID
		case "role":
			return v.Role
		case "username":
			return v.Username
		}
	case RequestContext:
		switch field {
		case "tenant_id":
			return v.TenantID
		case "action":
			return v.Action
		}
	}
	return nil
}

// --- Parser ---

type parser struct {
	input string
	pos   int
}

func (p *parser) skipSpaces() {
	for p.pos < len(p.input) && p.input[p.pos] == ' ' {
		p.pos++
	}
}

func (p *parser) peek() byte {
	if p.pos >= len(p.input) {
		return 0
	}
	return p.input[p.pos]
}

func (p *parser) parseExpr() (node, error) {
	return p.parseOr()
}

func (p *parser) parseOr() (node, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for {
		p.skipSpaces()
		if p.pos+1 < len(p.input) && p.input[p.pos] == '|' && p.input[p.pos+1] == '|' {
			p.pos += 2
			right, err := p.parseAnd()
			if err != nil {
				return nil, err
			}
			left = &orNode{left, right}
		} else {
			break
		}
	}
	return left, nil
}

func (p *parser) parseAnd() (node, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	for {
		p.skipSpaces()
		if p.pos+1 < len(p.input) && p.input[p.pos] == '&' && p.input[p.pos+1] == '&' {
			p.pos += 2
			right, err := p.parsePrimary()
			if err != nil {
				return nil, err
			}
			left = &andNode{left, right}
		} else {
			break
		}
	}
	return left, nil
}

func (p *parser) parsePrimary() (node, error) {
	p.skipSpaces()
	if p.peek() == '(' {
		p.pos++
		n, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		p.skipSpaces()
		if p.peek() != ')' {
			return nil, fmt.Errorf("expected ')' at position %d", p.pos)
		}
		p.pos++
		return n, nil
	}
	return p.parseComparison()
}

func (p *parser) parseComparison() (node, error) {
	path, err := p.parsePath()
	if err != nil {
		return nil, err
	}
	p.skipSpaces()
	op, err := p.parseOp()
	if err != nil {
		return nil, err
	}
	p.skipSpaces()
	val, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	return &cmpNode{path: path, op: op, value: val}, nil
}

func (p *parser) parsePath() ([]string, error) {
	p.skipSpaces()
	ident, err := p.parseIdent()
	if err != nil {
		return nil, err
	}
	parts := []string{ident}
	for {
		p.skipSpaces()
		if p.peek() == '.' {
			p.pos++
			ident, err := p.parseIdent()
			if err != nil {
				return nil, err
			}
			parts = append(parts, ident)
		} else {
			break
		}
	}
	return parts, nil
}

func (p *parser) parseIdent() (string, error) {
	p.skipSpaces()
	start := p.pos
	for p.pos < len(p.input) && (isAlphaNum(p.input[p.pos]) || p.input[p.pos] == '_') {
		p.pos++
	}
	if p.pos == start {
		return "", fmt.Errorf("expected identifier at position %d", p.pos)
	}
	return p.input[start:p.pos], nil
}

func isAlphaNum(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

func (p *parser) parseOp() (string, error) {
	p.skipSpaces()
	rest := p.input[p.pos:]
	for _, op := range []string{"==", "!=", ">=", "<=", "not_in", "contains", "matches", "in", ">", "<"} {
		if strings.HasPrefix(rest, op) {
			p.pos += len(op)
			return op, nil
		}
	}
	return "", fmt.Errorf("expected operator at position %d", p.pos)
}

func (p *parser) parseValue() (any, error) {
	p.skipSpaces()
	c := p.peek()
	if c == '"' {
		return p.parseString()
	}
	if c == '[' {
		return p.parseArray()
	}
	if c == '-' || (c >= '0' && c <= '9') {
		return p.parseNumber()
	}
	// keywords
	start := p.pos
	for p.pos < len(p.input) && (isAlphaNum(p.input[p.pos]) || p.input[p.pos] == '_') {
		p.pos++
	}
	word := p.input[start:p.pos]
	if word == "" {
		return nil, fmt.Errorf("expected value at position %d", p.pos)
	}
	switch word {
	case "true":
		return true, nil
	case "false":
		return false, nil
	case "null":
		return nil, nil
	case "empty":
		return sentinelEmpty, nil
	}
	return nil, fmt.Errorf("unknown value: %s", word)
}

func (p *parser) parseString() (string, error) {
	if p.peek() != '"' {
		return "", fmt.Errorf("expected '\"' at position %d", p.pos)
	}
	p.pos++
	start := p.pos
	for p.pos < len(p.input) && p.input[p.pos] != '"' {
		p.pos++
	}
	if p.pos >= len(p.input) {
		return "", fmt.Errorf("unterminated string")
	}
	s := p.input[start:p.pos]
	p.pos++
	return s, nil
}

func (p *parser) parseNumber() (any, error) {
	start := p.pos
	if p.peek() == '-' {
		p.pos++
	}
	for p.pos < len(p.input) && p.input[p.pos] >= '0' && p.input[p.pos] <= '9' {
		p.pos++
	}
	if p.pos < len(p.input) && p.input[p.pos] == '.' {
		p.pos++
		for p.pos < len(p.input) && p.input[p.pos] >= '0' && p.input[p.pos] <= '9' {
			p.pos++
		}
		f, err := strconv.ParseFloat(p.input[start:p.pos], 64)
		return f, err
	}
	i, err := strconv.Atoi(p.input[start:p.pos])
	return float64(i), err
}

func (p *parser) parseArray() ([]any, error) {
	if p.peek() != '[' {
		return nil, fmt.Errorf("expected '[' at position %d", p.pos)
	}
	p.pos++
	var items []any
	for {
		p.skipSpaces()
		if p.peek() == ']' {
			p.pos++
			return items, nil
		}
		val, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		items = append(items, val)
		p.skipSpaces()
		if p.peek() == ',' {
			p.pos++
		}
	}
}
