package thinking

import (
	"strings"
	"text/template"
	"text/template/parse"
)

type visitContext int

const (
	contextAction visitContext = iota
	contextPipeline
)

func templateVisit(n parse.Node, ctx visitContext, enterFn func(parse.Node, visitContext) bool, exitFn func(parse.Node, visitContext)) {
	if n == nil {
		return
	}
	shouldContinue := enterFn(n, ctx)
	if !shouldContinue {
		return
	}
	switch x := n.(type) {
	case *parse.ListNode:
		for _, c := range x.Nodes {
			templateVisit(c, contextAction, enterFn, exitFn)
		}
	case *parse.BranchNode:
		if x.Pipe != nil {
			templateVisit(x.Pipe, contextPipeline, enterFn, exitFn)
		}
		if x.List != nil {
			templateVisit(x.List, contextAction, enterFn, exitFn)
		}
		if x.ElseList != nil {
			templateVisit(x.ElseList, contextAction, enterFn, exitFn)
		}
	case *parse.ActionNode:
		templateVisit(x.Pipe, contextAction, enterFn, exitFn)
	case *parse.WithNode:
		templateVisit(&x.BranchNode, ctx, enterFn, exitFn)
	case *parse.RangeNode:
		templateVisit(&x.BranchNode, ctx, enterFn, exitFn)
	case *parse.IfNode:
		templateVisit(&x.BranchNode, ctx, enterFn, exitFn)
	case *parse.TemplateNode:
		templateVisit(x.Pipe, contextAction, enterFn, exitFn)
	case *parse.PipeNode:
		for _, c := range x.Cmds {
			templateVisit(c, ctx, enterFn, exitFn)
		}
	case *parse.CommandNode:
		for _, a := range x.Args {
			templateVisit(a, ctx, enterFn, exitFn)
		}
		// text, field, number, etc. are leaves – nothing to recurse into
	}
	if exitFn != nil {
		exitFn(n, ctx)
	}
}

// InferTags uses a heuristic to infer the tags that surround thinking traces:
// We look for a range node that iterates over "Messages" and then look for a
// reference to "Thinking" like `{{.Thinking}}`. We then go up to the nearest
// ListNode and take the first and last TextNodes as the opening and closing
// tags.
func InferTags(t *template.Template) (string, string) {
	ancestors := []parse.Node{}

	openingTag := ""
	closingTag := ""

	enterFn := func(n parse.Node, ctx visitContext) bool {
		ancestors = append(ancestors, n)

		switch x := n.(type) {
		case *parse.FieldNode:
			if ctx == contextPipeline {
				return true
			}
			if len(x.Ident) > 0 && x.Ident[0] == "Thinking" {
				var mostRecentRange *parse.RangeNode
				for i := len(ancestors) - 1; i >= 0; i-- {
					if r, ok := ancestors[i].(*parse.RangeNode); ok {
						mostRecentRange = r
						break
					}
				}
				if mostRecentRange == nil || !rangeUsesField(mostRecentRange, "Messages") {
					return true
				}

				// TODO(drifkin): to be more robust, check that it's in the action
				// part, not the `if`'s pipeline part. We do match on the nearest list
				// that starts and ends with text nodes, which makes this not strictly
				// necessary for our heuristic

				// go up to the nearest ancestor that is a *parse.ListNode
				for i := len(ancestors) - 1; i >= 0; i-- {
					if l, ok := ancestors[i].(*parse.ListNode); ok {
						firstNode := l.Nodes[0]
						if t, ok := firstNode.(*parse.TextNode); ok {
							openingTag = strings.TrimSpace(t.String())
						}
						lastNode := l.Nodes[len(l.Nodes)-1]
						if t, ok := lastNode.(*parse.TextNode); ok {
							closingTag = strings.TrimSpace(t.String())
						}

						break
					}
				}
			}
		}

		return true
	}

	exitFn := func(n parse.Node, ctx visitContext) {
		ancestors = ancestors[:len(ancestors)-1]
	}

	templateVisit(t.Root, contextAction, enterFn, exitFn)

	return openingTag, closingTag
}

// checks to see if the given field name is present in the pipeline of the given range node
func rangeUsesField(rangeNode *parse.RangeNode, field string) bool {
	found := false
	enterFn := func(n parse.Node, ctx visitContext) bool {
		switch x := n.(type) {
		case *parse.FieldNode:
			if x.Ident[0] == field {
				found = true
			}
		}
		return true
	}
	templateVisit(rangeNode.BranchNode.Pipe, contextPipeline, enterFn, nil)
	return found
}
