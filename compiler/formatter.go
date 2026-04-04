package compiler

import (
	"sort"
	"strings"
)

// nodeCategory returns a category string for grouping nodes
func nodeCategory(n Node) string {
	switch n.(type) {
	case *ConstDecl:
		return "const"
	case *DeclEnum:
		return "enum"
	case *DeclModel:
		return "model"
	case *DeclService:
		return "service"
	case *DeclError:
		return "error"
	default:
		return "unknown"
	}
}

// categoryOrder returns the sort order for a node's category
// Order: const=0, enum=1, model=2, service=3, error=4
func categoryOrder(n Node) int {
	switch n.(type) {
	case *ConstDecl:
		return 0
	case *DeclEnum:
		return 1
	case *DeclModel:
		return 2
	case *DeclService:
		return 3
	case *DeclError:
		return 4
	default:
		return 5
	}
}

// AssociateComments attaches comments to their nearest nodes
func AssociateComments(prog *Program) []*CommentedNode {
	if len(prog.Nodes) == 0 {
		return nil
	}

	result := make([]*CommentedNode, len(prog.Nodes))
	for i, node := range prog.Nodes {
		result[i] = &CommentedNode{Node: node}
	}

	commentIdx := 0
	comments := prog.Comments

	for i, cn := range result {
		nodeTok := getTokenFromNode(cn.Node)
		if nodeTok == nil {
			continue
		}

		// Get the end offset of the previous node to exclude its internal comments
		prevEndOffset := 0
		if i > 0 {
			prevEndOffset = getEndOffset(result[i-1].Node)
		}

		// Collect leading comments (comments that appear before this node)
		for commentIdx < len(comments) {
			c := comments[commentIdx]
			if c.Pos.Offset >= nodeTok.Pos.Offset {
				break
			}

			// Skip comments that are inside the previous node (block internal comments)
			if c.Pos.Offset < prevEndOffset {
				commentIdx++
				continue
			}

			// Check if this comment belongs to the previous node as a trailing comment
			// (on the same line as the previous node's end)
			if i > 0 {
				prevEndLine := getEndLine(result[i-1].Node)
				if c.Pos.Line == prevEndLine && result[i-1].TrailingComment == nil {
					result[i-1].TrailingComment = c
					commentIdx++
					continue
				}
			}

			// This is a leading comment for the current node
			cn.LeadingComments = append(cn.LeadingComments, c)
			commentIdx++
		}

		// Now check for inline comments that are on the same line as this node's opening
		// but after the node's start position (for block nodes like service { # comment)
		nodeStartLine := nodeTok.Pos.Line
		for commentIdx < len(comments) {
			c := comments[commentIdx]
			// If the comment is on the same line as the node start, it's an inline trailing comment
			if c.Pos.Line == nodeStartLine && c.Pos.Offset > nodeTok.Pos.Offset {
				cn.TrailingComment = c
				commentIdx++
				break
			}
			break
		}
	}

	// Handle remaining comments as trailing comments for the last node
	if len(result) > 0 && commentIdx < len(comments) {
		lastNode := result[len(result)-1]
		lastEndLine := getEndLine(lastNode.Node)
		for commentIdx < len(comments) {
			c := comments[commentIdx]
			if c.Pos.Line == lastEndLine && lastNode.TrailingComment == nil {
				lastNode.TrailingComment = c
			}
			// Remaining comments after the last node are ignored for now
			// or could be added as trailing comments
			commentIdx++
		}
	}

	return result
}

func Format(prog *Program) string {
	var sb strings.Builder

	// Associate comments with nodes
	commentedNodes := AssociateComments(prog)
	if len(commentedNodes) == 0 {
		return ""
	}

	// Sort nodes by category order: const, enum, model, service, error
	sort.SliceStable(commentedNodes, func(i, j int) bool {
		return categoryOrder(commentedNodes[i].Node) < categoryOrder(commentedNodes[j].Node)
	})

	lastCategory := ""

	for i, cn := range commentedNodes {
		currentCategory := nodeCategory(cn.Node)

		// Add separator between declarations.
		// Keep const/error packed, but split enum/model/service declarations.
		if i > 0 {
			if lastCategory != currentCategory {
				sb.WriteString("\n\n")
			} else if shouldSplitSameCategory(currentCategory) {
				sb.WriteString("\n\n")
			} else {
				sb.WriteString("\n")
			}
		}

		// Print leading comments
		for j, c := range cn.LeadingComments {
			if j > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(c.Lit)
		}

		// Add newline after leading comments if any
		if len(cn.LeadingComments) > 0 {
			sb.WriteString("\n")
		}

		// Format the node itself, passing the trailing comment for block nodes
		formatNodeWithComments(&sb, cn.Node, prog.Comments, cn.TrailingComment)

		lastCategory = currentCategory
	}

	return sb.String()
}

func shouldSplitSameCategory(category string) bool {
	switch category {
	case "enum", "model", "service":
		return true
	default:
		return false
	}
}

func getEndLine(node Node) int {
	switch n := node.(type) {
	case *ConstDecl:
		return getEndLine(n.Assignment.Value)
	case *AssignmentStmt:
		return getEndLine(n.Value)
	case *ValueExprNumber:
		return n.Token.Pos.Line
	case *ValueExprString:
		return n.Token.Pos.Line
	case *ValueExprBool:
		return n.Token.Pos.Line
	case *ValueExprNull:
		return n.Token.Pos.Line
	case *IdenExpr:
		return n.Token.Pos.Line
	case *DeclModel:
		if n.CloseCurly != nil {
			return n.CloseCurly.Pos.Line
		}
	case *DeclEnum:
		if n.CloseCurly != nil {
			return n.CloseCurly.Pos.Line
		}
	case *DeclService:
		if n.CloseCurly != nil {
			return n.CloseCurly.Pos.Line
		}
	case *DeclError:
		if n.CloseCurly != nil {
			return n.CloseCurly.Pos.Line
		}
	case *DeclModelField:
		return getEndLine(n.Type)
	case *DeclCustomType:
		return n.Name.Token.Pos.Line
	case *DeclStringType:
		return n.Name.Token.Pos.Line
	case *DeclNumberType:
		return n.Name.Token.Pos.Line
	case *DeclBoolType:
		return n.Name.Token.Pos.Line
	case *DeclArrayType:
		return getEndLine(n.Type)
	case *DeclMapType:
		return n.Token.Pos.Line // Approximate, should be >
	}
	// Fallback
	tok := getTokenFromNode(node)
	if tok != nil {
		return tok.Pos.Line
	}
	return -1
}

func getEndOffset(node Node) int {
	switch n := node.(type) {
	case *DeclModel:
		if n.CloseCurly != nil {
			return n.CloseCurly.Pos.Offset
		}
	case *DeclEnum:
		if n.CloseCurly != nil {
			return n.CloseCurly.Pos.Offset
		}
	case *DeclService:
		if n.CloseCurly != nil {
			return n.CloseCurly.Pos.Offset
		}
	case *DeclError:
		if n.CloseCurly != nil {
			return n.CloseCurly.Pos.Offset
		}
	}
	// For non-block nodes, return 0 (meaning no internal space)
	return 0
}

func formatNode(sb *strings.Builder, node Node, comments []*Token, commentIndex *int, lastLine *int) {
	formatNodeWithTrailing(sb, node, comments, commentIndex, lastLine, nil)
}

func formatNodeWithTrailing(sb *strings.Builder, node Node, comments []*Token, commentIndex *int, lastLine *int, trailingComment *Token) {
	// Helper to print comments inside the node
	printCommentsUntil := func(limit int) bool {
		printed := false
		for *commentIndex < len(comments) {
			c := comments[*commentIndex]
			if c.Pos.Offset < limit {
				if *lastLine == c.Pos.Line {
					// Comment on the same line - append with space
					sb.WriteString(" ")
					sb.WriteString(c.Lit)
				} else {
					// Comment on a different line - newline + indent
					sb.WriteString("\n\t")
					sb.WriteString(c.Lit)
				}
				*lastLine = c.Pos.Line
				*commentIndex++
				printed = true
			} else {
				break
			}
		}
		return printed
	}

	switch n := node.(type) {
	case *DeclModel:
		sb.WriteString("model ")
		sb.WriteString(n.Name.String())
		sb.WriteString(" {")
		if trailingComment != nil {
			sb.WriteString(" ")
			sb.WriteString(trailingComment.Lit)
		}
		*lastLine = n.Token.Pos.Line // Approximate start line

		// Combine fields and extends to sort them by position
		type child struct {
			pos   int
			node  Node
			isExt bool
		}
		var children []child

		for _, ext := range n.Extends {
			children = append(children, child{pos: ext.Token.Pos.Offset, node: ext, isExt: true})
		}
		for _, field := range n.Fields {
			tok := getTokenFromNode(field)
			if tok != nil {
				children = append(children, child{pos: tok.Pos.Offset, node: field, isExt: false})
			}
		}

		sort.Slice(children, func(i, j int) bool {
			return children[i].pos < children[j].pos
		})

		for _, ch := range children {
			printCommentsUntil(ch.pos)
			sb.WriteString("\n\t")
			if ch.isExt {
				sb.WriteString("...")
				sb.WriteString(ch.node.String())
			} else {
				sb.WriteString(ch.node.String())
			}
			// Update lastLine based on the child node
			*lastLine = getEndLine(ch.node)
		}

		if n.CloseCurly != nil {
			printCommentsUntil(n.CloseCurly.Pos.Offset)
			sb.WriteString("\n")
			*lastLine = n.CloseCurly.Pos.Line
		} else {
			sb.WriteString("\n")
		}
		sb.WriteString("}")

	case *DeclEnum:
		sb.WriteString("enum ")
		sb.WriteString(n.Name.String())
		sb.WriteString(" {")
		if trailingComment != nil {
			sb.WriteString(" ")
			sb.WriteString(trailingComment.Lit)
		}
		*lastLine = n.Token.Pos.Line

		for _, val := range n.Values {
			tok := getTokenFromNode(val)
			if tok != nil {
				printCommentsUntil(tok.Pos.Offset)
			}
			sb.WriteString("\n\t")
			sb.WriteString(val.String())
			*lastLine = getEndLine(val)
		}
		if n.CloseCurly != nil {
			printCommentsUntil(n.CloseCurly.Pos.Offset)
			sb.WriteString("\n")
			*lastLine = n.CloseCurly.Pos.Line
		} else {
			sb.WriteString("\n")
		}
		sb.WriteString("}")

	case *DeclService:
		sb.WriteString("service ")
		sb.WriteString(n.Name.String())
		sb.WriteString(" {")
		if trailingComment != nil {
			sb.WriteString(" ")
			sb.WriteString(trailingComment.Lit)
		}
		*lastLine = n.Token.Pos.Line

		for _, method := range n.Methods {
			tok := getTokenFromNode(method)
			if tok != nil {
				printCommentsUntil(tok.Pos.Offset)
			}
			sb.WriteString("\n\t")
			sb.WriteString(method.String())
			*lastLine = getEndLine(method)
		}
		if n.CloseCurly != nil {
			printCommentsUntil(n.CloseCurly.Pos.Offset)
			sb.WriteString("\n")
			*lastLine = n.CloseCurly.Pos.Line
		} else {
			sb.WriteString("\n")
		}
		sb.WriteString("}")

	default:
		sb.WriteString(node.String())
		if trailingComment != nil {
			sb.WriteString(" ")
			sb.WriteString(trailingComment.Lit)
		}
		*lastLine = getEndLine(node)
	}
}

// formatNodeWithComments formats a node handling internal comments
func formatNodeWithComments(sb *strings.Builder, node Node, allComments []*Token, trailingComment *Token) {
	// Find comments that are inside this node (for block nodes like model, service, enum)
	nodeTok := getTokenFromNode(node)
	if nodeTok == nil {
		sb.WriteString(node.String())
		return
	}

	nodeStart := nodeTok.Pos.Offset
	nodeEndLine := getEndLine(node)

	// Filter comments that are internal to this node
	// Skip comments on the same line as the node start (they are trailing comments for the opening)
	var internalComments []*Token
	for _, c := range allComments {
		// Comment must be after node start and before the end line
		// Also skip comments on the same line as the node declaration (inline comments handled separately)
		if c.Pos.Offset > nodeStart && c.Pos.Line > nodeTok.Pos.Line && c.Pos.Line < nodeEndLine {
			internalComments = append(internalComments, c)
		}
	}

	commentIndex := 0
	lastLine := nodeTok.Pos.Line

	formatNodeWithTrailing(sb, node, internalComments, &commentIndex, &lastLine, trailingComment)
}
