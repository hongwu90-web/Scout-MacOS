package focusquery

import (
	"strings"
	"unicode"
)

type TokenType int

const (
	TokenEOF TokenType = iota
	TokenWord
	TokenPhrase
	TokenAnd
	TokenOr
	TokenNot
	TokenLParen
	TokenRParen
	TokenField
)

type Token struct {
	Type     TokenType
	Value    string
	Position int
}

type Lexer struct {
	input []rune
	pos   int
}

func NewLexer(input string) *Lexer {
	return &Lexer{
		input: []rune(input),
		pos:   0,
	}
}

func (l *Lexer) NextToken() Token {
	l.skipWhitespace()

	if l.pos >= len(l.input) {
		return Token{Type: TokenEOF, Position: l.pos}
	}

	start := l.pos
	ch := l.input[l.pos]

	switch ch {
	case '(':
		l.pos++
		return Token{Type: TokenLParen, Value: "(", Position: start}
	case ')':
		l.pos++
		return Token{Type: TokenRParen, Value: ")", Position: start}
	case ',':
		l.pos++
		return Token{Type: TokenOr, Value: "OR", Position: start}
	case '!':
		l.pos++
		return Token{Type: TokenNot, Value: "NOT", Position: start}
	case '-':
		// A minus sign represents NOT
		l.pos++
		return Token{Type: TokenNot, Value: "NOT", Position: start}
	case '&':
		if l.peek() == '&' {
			l.pos += 2
			return Token{Type: TokenAnd, Value: "AND", Position: start}
		}
	case '|':
		if l.peek() == '|' {
			l.pos += 2
			return Token{Type: TokenOr, Value: "OR", Position: start}
		}
	case '"', '\'', '“', '”': // Support standard and smart quotes
		quoteChar := ch
		if ch == '“' {
			quoteChar = '”'
		}
		l.pos++ // skip opening quote
		phraseStart := l.pos
		for l.pos < len(l.input) {
			if l.input[l.pos] == quoteChar || (quoteChar == '\'' && l.input[l.pos] == '\'') || (quoteChar == '"' && l.input[l.pos] == '"') {
				break
			}
			l.pos++
		}
		val := string(l.input[phraseStart:l.pos])
		if l.pos < len(l.input) {
			l.pos++ // skip closing quote
		}
		return Token{Type: TokenPhrase, Value: strings.TrimSpace(val), Position: start}
	}

	// Read word or field identifier
	wordStart := l.pos
	for l.pos < len(l.input) {
		c := l.input[l.pos]
		if unicode.IsSpace(c) || c == '(' || c == ')' || c == ',' || c == '"' || c == '\'' || c == '“' || c == '”' {
			break
		}
		if c == '&' && l.peek() == '&' {
			break
		}
		if c == '|' && l.peek() == '|' {
			break
		}
		l.pos++
	}

	word := string(l.input[wordStart:l.pos])

	// Check if this is a field specifier like title: or content: or body:
	if strings.HasSuffix(word, ":") {
		fieldName := strings.ToLower(strings.TrimSuffix(word, ":"))
		if fieldName == "title" || fieldName == "body" || fieldName == "content" {
			if fieldName == "body" {
				fieldName = "content"
			}
			return Token{Type: TokenField, Value: fieldName, Position: wordStart}
		}
	}

	// Check for case-insensitive keywords: AND, OR, NOT
	upper := strings.ToUpper(word)
	switch upper {
	case "AND":
		return Token{Type: TokenAnd, Value: "AND", Position: wordStart}
	case "OR":
		return Token{Type: TokenOr, Value: "OR", Position: wordStart}
	case "NOT":
		return Token{Type: TokenNot, Value: "NOT", Position: wordStart}
	default:
		return Token{Type: TokenWord, Value: word, Position: wordStart}
	}
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) && unicode.IsSpace(l.input[l.pos]) {
		l.pos++
	}
}

func (l *Lexer) peek() rune {
	if l.pos+1 < len(l.input) {
		return l.input[l.pos+1]
	}
	return 0
}

func Tokenize(input string) []Token {
	lexer := NewLexer(input)
	var tokens []Token
	for {
		tok := lexer.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == TokenEOF {
			break
		}
	}
	return tokens
}
