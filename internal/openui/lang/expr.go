package lang

import "strconv"

const (
	precTernary = 1
	precOr      = 2
	precAnd     = 3
	precEq      = 4
	precCmp     = 5
	precAdd     = 6
	precMul     = 7
	precUnary   = 8
	precMember  = 9
)

// exprParser is the Pratt parser of expressions (upstream expressions.ts).
type exprParser struct {
	toks     []token
	pos      int
	depth    int
	maxDepth int
	nodes    int
	maxNodes int
	over     bool
	calls    *int // call ordinal counter for source keys
}

var eofTok = token{k: tEOF}

func (p *exprParser) cur() token {
	if p.pos < len(p.toks) {
		return p.toks[p.pos]
	}
	return eofTok
}

func (p *exprParser) peekKind(off int) tokKind {
	if p.pos+off < len(p.toks) {
		return p.toks[p.pos+off].k
	}
	return tEOF
}

func (p *exprParser) adv() token {
	t := p.cur()
	p.pos++
	return t
}

func (p *exprParser) eat(k tokKind) {
	if p.cur().k == k {
		p.pos++
	}
}

func infixPrec(k tokKind) int {
	switch k {
	case tQuestion:
		return precTernary
	case tOr:
		return precOr
	case tAnd:
		return precAnd
	case tEqEq, tNotEq:
		return precEq
	case tGreater, tLess, tGreaterEq, tLessEq:
		return precCmp
	case tPlus, tMinus:
		return precAdd
	case tStar, tSlash, tPercent:
		return precMul
	case tDot, tLBrack:
		return precMember
	}
	return 0
}

func (p *exprParser) count() {
	p.nodes++
	if p.maxNodes > 0 && p.nodes > p.maxNodes {
		p.over = true
	}
}

func (p *exprParser) parseExpr(minPrec int) Node {
	if p.over {
		p.pos = len(p.toks)
		return nullNode
	}
	p.depth++
	defer func() { p.depth-- }()
	if p.maxDepth > 0 && p.depth > p.maxDepth {
		p.over = true
		p.pos = len(p.toks)
		return nullNode
	}
	left := p.parsePrefix()
	for !p.over && infixPrec(p.cur().k) > minPrec {
		left = p.parseInfix(left)
	}
	return left
}

func (p *exprParser) parsePrefix() Node {
	p.count()
	tok := p.cur()
	switch tok.k {
	case tStr:
		p.adv()
		return &StrNode{V: tok.s}
	case tNum:
		p.adv()
		return &NumNode{V: tok.n}
	case tTrue:
		p.adv()
		return &BoolNode{V: true}
	case tFalse:
		p.adv()
		return &BoolNode{V: false}
	case tNull:
		p.adv()
		return nullNode
	case tLBrack:
		return p.parseArr()
	case tLBrace:
		return p.parseObj()
	case tStateVar:
		p.adv()
		if p.cur().k == tEquals {
			p.adv()
			return &AssignNode{Target: tok.s, Value: p.parseExpr(0)}
		}
		return &StateRefNode{N: tok.s}
	case tType:
		if p.peekKind(1) == tLParen && (!isBuiltinName(tok.s) || tok.s == "Action") {
			return p.parseCall()
		}
		p.adv()
		return &RefNode{N: tok.s}
	case tBuiltin:
		if p.peekKind(1) == tLParen {
			c := p.parseCall()
			if !isBuiltinName(c.Name) {
				c.Fn = true
			}
			return c
		}
		p.adv()
		return &RefNode{N: tok.s}
	case tIdent:
		p.adv()
		return &RefNode{N: tok.s}
	case tNot:
		p.adv()
		return &UnaryNode{Op: "!", X: p.parseExpr(precUnary)}
	case tMinus:
		p.adv()
		return &UnaryNode{Op: "-", X: p.parseExpr(precUnary)}
	case tLParen:
		p.adv()
		inner := p.parseExpr(0)
		p.eat(tRParen)
		return inner
	}
	p.adv()
	return nullNode
}

func (p *exprParser) parseInfix(left Node) Node {
	tok := p.cur()
	bin := func(op string, prec int) Node {
		p.adv()
		return &BinOpNode{Op: op, L: left, R: p.parseExpr(prec)}
	}
	switch tok.k {
	case tPlus:
		return bin("+", precAdd)
	case tMinus:
		return bin("-", precAdd)
	case tStar:
		return bin("*", precMul)
	case tSlash:
		return bin("/", precMul)
	case tPercent:
		return bin("%", precMul)
	case tEqEq:
		return bin("==", precEq)
	case tNotEq:
		return bin("!=", precEq)
	case tGreater:
		return bin(">", precCmp)
	case tLess:
		return bin("<", precCmp)
	case tGreaterEq:
		return bin(">=", precCmp)
	case tLessEq:
		return bin("<=", precCmp)
	case tAnd:
		return bin("&&", precAnd)
	case tOr:
		return bin("||", precOr)
	case tQuestion:
		p.adv()
		then := p.parseExpr(0)
		p.eat(tColon)
		els := p.parseExpr(0)
		return &TernaryNode{Cond: left, Then: then, Else: els}
	case tDot:
		p.adv()
		return &MemberNode{X: left, Field: p.keyToken()}
	case tLBrack:
		p.adv()
		idx := p.parseExpr(0)
		p.eat(tRBrack)
		return &IndexNode{X: left, Index: idx}
	}
	return left
}

// keyToken reads an object key or member field token.
func (p *exprParser) keyToken() string {
	t := p.adv()
	switch t.k {
	case tIdent, tType, tStr:
		return t.s
	case tNum:
		return numKey(t.n)
	case tStateVar:
		return t.s[1:]
	}
	return "?"
}

func (p *exprParser) parseCall() *CallNode {
	name := p.cur().s
	key := ""
	if p.calls != nil {
		key = strconv.Itoa(*p.calls)
		*p.calls++
	}
	p.adv()
	p.eat(tLParen)
	var args []Node
	for !p.over && p.cur().k != tRParen && p.cur().k != tEOF {
		args = append(args, p.parseExpr(0))
		if p.cur().k == tComma {
			p.adv()
		}
	}
	p.eat(tRParen)
	return &CallNode{Name: name, Args: args, SourceKey: key}
}

func (p *exprParser) parseArr() Node {
	p.adv()
	var els []Node
	for !p.over && p.cur().k != tRBrack && p.cur().k != tEOF {
		els = append(els, p.parseExpr(0))
		if p.cur().k == tComma {
			p.adv()
		}
	}
	p.eat(tRBrack)
	return &ArrNode{Els: els}
}

func (p *exprParser) parseObj() Node {
	p.adv()
	var ents []ObjEntry
	for !p.over && p.cur().k != tRBrace && p.cur().k != tEOF {
		key := p.keyToken()
		p.eat(tColon)
		ents = append(ents, ObjEntry{Key: key, Value: p.parseExpr(0)})
		if p.cur().k == tComma {
			p.adv()
		}
	}
	p.eat(tRBrace)
	return &ObjNode{Entries: ents}
}

// parseExpression parses the tokens of one statement's right side. ok is
// false when a depth or node budget was exceeded. nodes carries the running
// node count across statements of one parse.
func parseExpression(toks []token, lim *Limits, calls *int, nodes *int) (Node, bool) {
	p := &exprParser{toks: toks, calls: calls}
	if lim != nil {
		p.maxDepth = lim.MaxDepth
		p.maxNodes = lim.MaxASTNodes
	}
	if nodes != nil {
		p.nodes = *nodes
	}
	n := p.parseExpr(0)
	if nodes != nil {
		*nodes = p.nodes
	}
	return n, !p.over
}
