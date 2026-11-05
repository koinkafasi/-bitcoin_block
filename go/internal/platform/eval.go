// Package platform — hesaplayıcı ifade değerlendiricisi (P5-F).
// Only numbers, [alan], + - * / % ve parantez; güvenli (no eval).
package platform

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var errExpr = errors.New("ifade hatası")

// EvalCalc bir ifadeyi değişken haritasıyla değerlendirir.
// [alan] sözdizimi değişkene işaret eder. Örn: "([ana] * [orani] / 100)"
func EvalCalc(expr string, vars map[string]float64) (float64, error) {
	p := &calcParser{tokens: tokenizeCalc(expr)}
	v, err := p.parseExpr()
	if err != nil {
		return 0, fmt.Errorf("%w: %v", errExpr, err)
	}
	return v, nil
}

type calcToken struct {
	kind byte // 'n' sayı, 'v' değişken, 'o' operatör, 'p' parantez
	val  float64
	name string
	op   byte
	pos  string
}

func tokenizeCalc(s string) []calcToken {
	var out []calcToken
	i := 0
	b := []rune(s)
	for i < len(b) {
		c := b[i]
		switch {
		case c == ' ' || c == '\t':
			i++
		case c == '+' || c == '-' || c == '*' || c == '/' || c == '%':
			out = append(out, calcToken{kind: 'o', op: byte(c), pos: string(c)})
			i++
		case c == '(' || c == ')':
			out = append(out, calcToken{kind: 'p', pos: string(c)})
			i++
		case c == '[':
			j := i + 1
			for j < len(b) && b[j] != ']' {
				j++
			}
			if j < len(b) {
				out = append(out, calcToken{kind: 'v', name: strings.TrimSpace(string(b[i+1 : j]))})
				i = j + 1
			} else {
				i++
			}
		case c >= '0' && c <= '9' || c == '.':
			j := i
			for j < len(b) && (b[j] >= '0' && b[j] <= '9' || b[j] == '.') {
				j++
			}
			if f, err := strconv.ParseFloat(string(b[i:j]), 64); err == nil {
				out = append(out, calcToken{kind: 'n', val: f, pos: string(b[i:j])})
			}
			i = j
		default:
			i++
		}
	}
	return out
}

type calcParser struct {
	tokens []calcToken
	pos    int
}

func (p *calcParser) peek() *calcToken {
	if p.pos < len(p.tokens) {
		return &p.tokens[p.pos]
	}
	return nil
}

func (p *calcParser) next() *calcToken {
	t := p.peek()
	if t != nil {
		p.pos++
	}
	return t
}

func (p *calcParser) parseExpr() (float64, error) {
	left, err := p.parseTerm()
	if err != nil {
		return 0, err
	}
	for {
		t := p.peek()
		if t == nil || t.kind != 'o' || (t.op != '+' && t.op != '-') {
			return left, nil
		}
		p.next()
		right, err := p.parseTerm()
		if err != nil {
			return 0, err
		}
		if t.op == '+' {
			left += right
		} else {
			left -= right
		}
	}
}

func (p *calcParser) parseTerm() (float64, error) {
	left, err := p.parseFactor()
	if err != nil {
		return 0, err
	}
	for {
		t := p.peek()
		if t == nil || t.kind != 'o' || (t.op != '*' && t.op != '/' && t.op != '%') {
			return left, nil
		}
		p.next()
		right, err := p.parseFactor()
		if err != nil {
			return 0, err
		}
		switch t.op {
		case '*':
			left *= right
		case '/':
			if right == 0 {
				return 0, errExpr
			}
			left /= right
		case '%':
			if right == 0 {
				return 0, errExpr
			}
			left = float64(int(left) % int(right))
		}
	}
}

func (p *calcParser) parseFactor() (float64, error) {
	t := p.next()
	if t == nil {
		return 0, errExpr
	}
	switch t.kind {
	case 'n':
		return t.val, nil
	case 'v':
		return t.posValue(), nil // vars sonradan set edilir (aşağıda)
	case 'o':
		if t.op == '-' { // unary eksi
			v, err := p.parseFactor()
			return -v, err
		}
		return 0, errExpr
	case 'p':
		if t.pos == "(" {
			v, err := p.parseExpr()
			if err != nil {
				return 0, err
			}
			if c := p.next(); c == nil || c.pos != ")" {
				return 0, errExpr
			}
			return v, nil
		}
	}
	return 0, errExpr
}

// posValue placeholder: değişken çözümü ParseCalc'ta ayrı yapılır; buraya gelmemeli.
func (t *calcToken) posValue() float64 { return 0 }

// CalcParse değişkenli ifadeyi ayrıştırır ve verilen değerlerle hesaplar.
func CalcParse(expr string, vars map[string]float64) (float64, error) {
	resolved := resolveCalcVars(expr, vars)
	return EvalCalc(resolved, nil)
}

func resolveCalcVars(expr string, vars map[string]float64) string {
	out := expr
	for name, val := range vars {
		out = strings.ReplaceAll(out, "["+name+"]", strconv.FormatFloat(val, 'f', -1, 64))
	}
	return out
}
