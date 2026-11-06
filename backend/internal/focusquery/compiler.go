package focusquery

import (
	"database/sql"
	"fmt"
	"strings"
)

type Compiler struct {
	paramCounter int
	args         []any
}

func NewCompiler() *Compiler {
	return &Compiler{
		paramCounter: 0,
		args:         []any{},
	}
}

// Compile compiles the given AST root node into a SQL clause starting with " AND (...)" and a list of sql.Named args.
func Compile(root Node) (string, []any, error) {
	if root == nil {
		return "", nil, nil
	}

	c := NewCompiler()
	clause, err := c.compileNode(root)
	if err != nil {
		return "", nil, err
	}

	if clause == "" {
		return "", nil, nil
	}

	return " AND (" + clause + ")", c.args, nil
}

func (c *Compiler) compileNode(node Node) (string, error) {
	if node == nil {
		return "", nil
	}

	switch n := node.(type) {
	case *TermNode:
		return c.compileTerm(n)
	case *NotNode:
		childClause, err := c.compileNode(n.Child)
		if err != nil {
			return "", err
		}
		if childClause == "" {
			return "", nil
		}
		return "(NOT " + childClause + ")", nil
	case *BinaryNode:
		leftClause, err := c.compileNode(n.Left)
		if err != nil {
			return "", err
		}
		rightClause, err := c.compileNode(n.Right)
		if err != nil {
			return "", err
		}

		if leftClause == "" && rightClause == "" {
			return "", nil
		}
		if leftClause == "" {
			return rightClause, nil
		}
		if rightClause == "" {
			return leftClause, nil
		}

		return "(" + leftClause + " " + n.Op + " " + rightClause + ")", nil
	default:
		return "", fmt.Errorf("unknown AST node type: %T", node)
	}
}

func (c *Compiler) compileTerm(term *TermNode) (string, error) {
	val := strings.TrimSpace(term.Value)
	if val == "" {
		return "", nil
	}

	paramName := fmt.Sprintf("ff_q_%d", c.paramCounter)
	c.paramCounter++
	c.args = append(c.args, sql.Named(paramName, "%"+val+"%"))

	switch term.Field {
	case "title":
		return fmt.Sprintf("(items.title LIKE :%s OR items.translated_title LIKE :%s)", paramName, paramName), nil
	case "content":
		return fmt.Sprintf("items.content LIKE :%s", paramName), nil
	default:
		return fmt.Sprintf("(items.title LIKE :%s OR items.translated_title LIKE :%s OR items.content LIKE :%s)", paramName, paramName, paramName), nil
	}
}
