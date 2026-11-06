package focusquery

import (
	"fmt"
	"strings"
)

type Parser struct {
	tokens []Token
	pos    int
}

func NewParser(tokens []Token) *Parser {
	return &Parser{tokens: tokens, pos: 0}
}

func Parse(input string) (Node, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil, nil
	}

	tokens := Tokenize(trimmed)
	p := NewParser(tokens)
	node, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	if p.current().Type != TokenEOF {
		return nil, fmt.Errorf("unexpected token %q at character %d", p.current().Value, p.current().Position)
	}

	return node, nil
}

func (p *Parser) current() Token {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	}
	return Token{Type: TokenEOF}
}

func (p *Parser) advance() Token {
	tok := p.current()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return tok
}

// Expression = OrExpr
func (p *Parser) parseExpression() (Node, error) {
	return p.parseOr()
}

// OrExpr = AndExpr { ( "OR" | "||" | "," ) AndExpr }
func (p *Parser) parseOr() (Node, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for p.current().Type == TokenOr {
		p.advance() // consume OR / comma
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		if right == nil {
			return nil, fmt.Errorf("expected expression after OR operator")
		}
		left = &BinaryNode{
			Op:    "OR",
			Left:  left,
			Right: right,
		}
	}

	return left, nil
}

// AndExpr = UnaryExpr { [ "AND" | "&&" ] UnaryExpr }
// Note: Adjacent terms without an explicit operator are treated as implicit AND
func (p *Parser) parseAnd() (Node, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	if left == nil {
		return nil, nil
	}

	for {
		isExplicitAnd := p.current().Type == TokenAnd
		if isExplicitAnd {
			p.advance() // consume AND
		} else if !p.canStartExpression(p.current()) {
			break
		}

		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		if right == nil {
			if isExplicitAnd {
				return nil, fmt.Errorf("expected expression after AND operator")
			}
			break
		}

		left = &BinaryNode{
			Op:    "AND",
			Left:  left,
			Right: right,
		}
	}

	return left, nil
}

// UnaryExpr = [ "NOT" | "!" | "-" ] PrimaryExpr
func (p *Parser) parseUnary() (Node, error) {
	if p.current().Type == TokenNot {
		p.advance() // consume NOT
		child, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		if child == nil {
			return nil, fmt.Errorf("expected expression after NOT operator")
		}
		return &NotNode{Child: child}, nil
	}

	return p.parsePrimary()
}

// PrimaryExpr = ( "(" Expression ")" ) | [ Field ] ( Word | Phrase )
func (p *Parser) parsePrimary() (Node, error) {
	tok := p.current()

	// Parenthesized grouping
	if tok.Type == TokenLParen {
		startPos := tok.Position
		p.advance() // consume (
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if p.current().Type != TokenRParen {
			return nil, fmt.Errorf("unmatched opening parenthesis '(' at position %d", startPos)
		}
		p.advance() // consume )
		return expr, nil
	}

	// Optional field specifier e.g. title: or content:
	var field string
	if tok.Type == TokenField {
		field = tok.Value
		p.advance() // consume field:
		tok = p.current()
	}

	// Term (word or phrase)
	if tok.Type == TokenWord {
		p.advance()
		return &TermNode{
			Field:       field,
			Value:       tok.Value,
			ExactPhrase: false,
		}, nil
	}

	if tok.Type == TokenPhrase {
		p.advance()
		return &TermNode{
			Field:       field,
			Value:       tok.Value,
			ExactPhrase: true,
		}, nil
	}

	if field != "" {
		return nil, fmt.Errorf("expected keyword or phrase after %s: prefix at position %d", field, tok.Position)
	}

	return nil, nil
}

func (p *Parser) canStartExpression(tok Token) bool {
	switch tok.Type {
	case TokenWord, TokenPhrase, TokenLParen, TokenNot, TokenField:
		return true
	default:
		return false
	}
}
