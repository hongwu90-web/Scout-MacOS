package focusquery

import "strings"

// Node represents an element in the focus query Abstract Syntax Tree.
type Node interface {
	node()
}

// TermNode represents a keyword or quoted phrase, with an optional field specifier (e.g., "title", "content").
type TermNode struct {
	Field       string // "" = any (title, translated_title, content), "title" = title only, "content" = body
	Value       string // Search string
	ExactPhrase bool   // True if enclosed in double quotes
}

func (t *TermNode) node() {}

// BinaryNode represents a binary operation: AND, OR.
type BinaryNode struct {
	Op    string // "AND" | "OR"
	Left  Node
	Right Node
}

func (b *BinaryNode) node() {}

// NotNode represents negation: NOT.
type NotNode struct {
	Child Node
}

func (n *NotNode) node() {}

// ExtractPositiveTerms traverses the AST and collects all leaf term values that are not under a NOT node.
// This is used for keyword badging on cards and in the UI.
func ExtractPositiveTerms(n Node) []string {
	if n == nil {
		return nil
	}
	var terms []string
	var walk func(node Node, negated bool)
	walk = func(node Node, negated bool) {
		if node == nil {
			return
		}
		switch v := node.(type) {
		case *TermNode:
			if !negated {
				val := strings.TrimSpace(v.Value)
				if val != "" {
					terms = append(terms, val)
				}
			}
		case *NotNode:
			walk(v.Child, !negated)
		case *BinaryNode:
			walk(v.Left, negated)
			walk(v.Right, negated)
		}
	}
	walk(n, false)

	// Deduplicate preserving order
	seen := make(map[string]bool)
	var unique []string
	for _, term := range terms {
		lower := strings.ToLower(term)
		if !seen[lower] {
			seen[lower] = true
			unique = append(unique, term)
		}
	}
	return unique
}
