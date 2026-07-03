package uri_trie

import (
	"fmt"
	"strings"
	"sync"

	"github.com/dlshle/gommon/utils"
)

type MatchContext struct {
	UriPattern  string
	QueryParams map[string]string
	PathParams  map[string]string
	Value       interface{}
}

type UriContext struct {
	params map[string]bool
}

func parseQueryParams(queryParamString string) (map[string]string, error) {
	pMap := make(map[string]string)
	if queryParamString == "" {
		return pMap, nil
	}
	exps := strings.Split(queryParamString, "&")
	for _, exp := range exps {
		if exp == "" {
			continue
		}
		key, val, err := getSplittedQueryParamStrings(exp)
		if err != nil {
			return nil, err
		}
		pMap[key] = val
	}
	return pMap, nil
}

func getSplittedQueryParamStrings(queryParam string) (string, string, error) {
	if queryParam == "" {
		return "", "", fmt.Errorf("empty query param")
	}
	eqIdx := strings.Index(queryParam, "=")
	if eqIdx == -1 {
		return "", "", fmt.Errorf("invalid query param %s: missing '='", queryParam)
	}
	if eqIdx == 0 {
		return "", "", fmt.Errorf("invalid query param %s: empty key", queryParam)
	}
	return queryParam[:eqIdx], queryParam[eqIdx+1:], nil
}

func splitRemaining(remaining string) (string, string) {
	if len(remaining) == 0 {
		return "", ""
	}
	i := 0
	for i < len(remaining) && remaining[i] != '/' {
		i++
	}
	if i == len(remaining) {
		return remaining, ""
	}
	return remaining[:i], remaining[i:]
}

func splitQueryParams(path string) (queries string, remaining string) {
	remaining = path
	iSplitter := strings.LastIndex(path, "?")
	if iSplitter == -1 {
		return
	}
	return path[iSplitter+1:], path[0:iSplitter]
}

const (
	tnTypeP = 0 // param node (e.g. :param)
	tnTypeW = 1 // wildcard node
	tnTypeC = 2 // constant or literal node
)

type trieNode struct {
	parent        *trieNode
	wildcardChild *trieNode
	paramChild    *trieNode
	constChildren map[string]*trieNode
	param         string
	value         interface{}
	path          string
	t             uint8
}

func (n *trieNode) addParam(param string) (*trieNode, error) {
	if n.wildcardChild != nil || (n.paramChild != nil && n.paramChild.param != param) {
		return nil, fmt.Errorf("can not add a new param node \"%s\" over a wildcard/const node or a param node w/ different param \"%s\"", param, n.param)
	}
	if n.paramChild == nil {
		n.paramChild = &trieNode{parent: n, param: param, t: tnTypeP}
	}
	return n.paramChild, nil
}

func (n *trieNode) addWildcard(param string) (*trieNode, error) {
	if (n.wildcardChild != nil && n.wildcardChild.param != param) || n.paramChild != nil {
		return nil, fmt.Errorf("can not add a new wildcard node \"%s\" over a param/const node or a wildcard node w/ different param \"%s\"", param, n.param)
	}
	if n.wildcardChild == nil {
		n.wildcardChild = &trieNode{parent: n, param: param, t: tnTypeW}
	}
	return n.wildcardChild, nil
}

func (n *trieNode) addConst(subPath string) (*trieNode, error) {
	if n.constChildren == nil {
		n.constChildren = make(map[string]*trieNode)
	}
	node := n.constChildren[subPath]
	if node == nil {
		node = &trieNode{parent: n, t: tnTypeC}
		n.constChildren[subPath] = node
	}
	return node, nil
}

func (n *trieNode) addPath(ctx UriContext, path string, value interface{}, override bool) (node *trieNode, isNew bool, err error) {
	if len(path) == 0 {
		if n.value != nil && !override {
			return n, false, fmt.Errorf("path / has already been taken, please use override=true to replace current value")
		}
		isNew = n.value == nil
		n.value = value
		n.path = "/"
		return n, isNew, nil
	}
	node = n
	remaining := path
	for len(remaining) > 0 {
		token := remaining[0]
		remaining = remaining[1:]
		switch token {
		case ':':
			var param string
			param, remaining = splitRemaining(remaining)
			err = utils.ProcessWithErrors(
				func() error {
					if ctx.params[param] {
						return fmt.Errorf("param %s has already been taken in url %s", param, path)
					}
					ctx.params[param] = true
					return nil
				},
				func() error {
					node, err = node.addParam(param)
					return err
				},
			)
		case '*':
			param := remaining
			remaining = ""
			node, err = node.addWildcard(param)
		case '/':
			node, err = node.addConst("/")
		default:
			var subPath string
			subPath, remaining = splitRemaining(remaining)
			subPath = fmt.Sprintf("%c%s", token, subPath)
			node, err = node.addConst(subPath)
		}
		if err != nil {
			return nil, false, err
		}
	}
	if node.value != nil && !override {
		return node, false, fmt.Errorf("path %s has already been taken, please use override=true to replace current value", path)
	}
	isNew = node.value == nil
	node.value = value
	node.path = path
	return node, isNew, nil
}

func (n *trieNode) removeFromParent() {
	if n.parent == nil || n.value != nil {
		return
	}
	if n.paramChild != nil || n.wildcardChild != nil || len(n.constChildren) > 0 {
		return
	}
	parent := n.parent
	if parent.paramChild == n {
		parent.paramChild = nil
	} else if parent.wildcardChild == n {
		parent.wildcardChild = nil
	} else if parent.constChildren != nil {
		for k, v := range parent.constChildren {
			if v == n {
				delete(parent.constChildren, k)
				break
			}
		}
	}
	n.parent = nil
	parent.removeFromParent()
}

func (n *trieNode) clean() {
	if n.paramChild != nil {
		n.paramChild.clean()
		n.paramChild = nil
	}
	if n.wildcardChild != nil {
		n.wildcardChild.clean()
		n.wildcardChild = nil
	}
	for k, c := range n.constChildren {
		c.clean()
		delete(n.constChildren, k)
	}
	n.value = nil
	n.path = ""
}

func (n *trieNode) findByPath(path string) *trieNode {
	if len(path) == 0 {
		return n
	}
	curr := n
	remaining := path
	for len(remaining) > 0 {
		if curr == nil {
			break
		}
		token := remaining[0]
		remaining = remaining[1:]
		switch token {
		case '/':
			curr = curr.constChildren["/"]
		default:
			var subPath string
			subPath, remaining = splitRemaining(remaining)
			if tCurr := curr.constChildren[fmt.Sprintf("%c%s", token, subPath)]; tCurr != nil {
				curr = tCurr
				continue
			}
			if curr.wildcardChild != nil {
				curr = curr.wildcardChild
				break
			} else if curr.paramChild != nil {
				curr = curr.paramChild
			}
		}
	}
	return curr
}

// findPatternNode walks the trie using a route pattern (e.g. "/x/:y/*z")
// rather than a concrete URI, so it can locate the exact node that Add
// created for removal.
func (n *trieNode) findPatternNode(pattern string) *trieNode {
	if len(pattern) == 0 {
		return n
	}
	curr := n
	remaining := pattern
	for len(remaining) > 0 {
		if curr == nil {
			return nil
		}
		token := remaining[0]
		remaining = remaining[1:]
		switch token {
		case '/':
			curr = curr.constChildren["/"]
		case ':':
			param, rest := splitRemaining(remaining)
			if curr.paramChild == nil || curr.paramChild.param != param {
				return nil
			}
			curr = curr.paramChild
			remaining = rest
		case '*':
			param := remaining
			if curr.wildcardChild == nil || curr.wildcardChild.param != param {
				return nil
			}
			return curr.wildcardChild
		default:
			var subPath string
			subPath, remaining = splitRemaining(remaining)
			subPath = fmt.Sprintf("%c%s", token, subPath)
			curr = curr.constChildren[subPath]
		}
	}
	return curr
}

func (n *trieNode) match(path string, ctx *MatchContext) (node *trieNode, err error) {
	if len(path) == 0 {
		return n, nil
	}
	curr := n
	remaining := path
	for len(remaining) > 0 {
		if curr == nil {
			err = fmt.Errorf("mismatched remaining path %s from %s- no routing found", remaining, path)
			curr = nil
			break
		}
		token := remaining[0]
		remaining = remaining[1:]
		switch token {
		case '/':
			curr = curr.constChildren["/"]
		default:
			var subPath string
			subPath, remaining = splitRemaining(remaining)
			subPath = fmt.Sprintf("%c%s", token, subPath)
			if tCurr := curr.constChildren[subPath]; tCurr != nil {
				curr = tCurr
				continue
			}
			if curr.wildcardChild != nil {
				ctx.PathParams[curr.wildcardChild.param] = subPath + remaining
				curr = curr.wildcardChild
				break
			} else if curr.paramChild != nil {
				ctx.PathParams[curr.paramChild.param] = subPath
				curr = curr.paramChild
			}
		}
	}
	if curr != nil {
		node = curr
		ctx.Value = node.value
		ctx.UriPattern = node.path
	} else if err == nil {
		err = fmt.Errorf("no routing found for path %s", path)
	}
	return
}

func (n *trieNode) matchByPath(pathWithoutQueryParams string, ctx *MatchContext) (*MatchContext, error) {
	if len(pathWithoutQueryParams) == 0 {
		if n.value == nil {
			return nil, fmt.Errorf("no value associated with path /")
		}
		ctx.Value = n.value
		ctx.UriPattern = n.path
		return ctx, nil
	}
	node, err := n.match(pathWithoutQueryParams, ctx)
	if err != nil || node == nil {
		return nil, err
	}
	if node.value == nil {
		return nil, fmt.Errorf("no value associated with path %s", pathWithoutQueryParams)
	}
	ctx.Value = node.value
	ctx.UriPattern = node.path
	return ctx, nil
}

type TrieTree struct {
	root *trieNode
	size int
	lock sync.RWMutex
}

func NewTrieTree() *TrieTree {
	return &TrieTree{
		root: &trieNode{},
	}
}

func (t *TrieTree) Size() int {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.size
}

func (t *TrieTree) Match(path string) (*MatchContext, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("empty path")
	}
	if path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}
	t.lock.RLock()
	defer t.lock.RUnlock()
	paramStr, remaining := splitQueryParams(path)
	queryParams, err := parseQueryParams(paramStr)
	if err != nil {
		return nil, err
	}
	c, e := t.root.matchByPath(remaining, &MatchContext{
		PathParams:  make(map[string]string),
		QueryParams: queryParams,
	})
	if c == nil || e != nil {
		return nil, e
	}
	return c, nil
}

func (t *TrieTree) Add(path string, value interface{}, override bool) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("empty path")
	}
	if path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}
	t.lock.Lock()
	defer t.lock.Unlock()
	_, isNew, err := t.root.addPath(UriContext{make(map[string]bool)}, path, value, override)
	if err != nil {
		return err
	}
	if isNew {
		t.size++
	}
	return nil
}

func (t *TrieTree) Remove(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	if path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}
	t.lock.Lock()
	defer t.lock.Unlock()
	node := t.root.findPatternNode(path)
	if node == nil || node.value == nil {
		return false
	}
	node.value = nil
	node.path = ""
	node.removeFromParent()
	t.size--
	return true
}

func (t *TrieTree) SupportsUri(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	if path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}
	t.lock.RLock()
	defer t.lock.RUnlock()
	paramStr, remaining := splitQueryParams(path)
	_, err := parseQueryParams(paramStr)
	if err != nil {
		return false
	}
	n := t.root.findByPath(remaining)
	if n == nil {
		return false
	}
	return n.value != nil
}

func (t *TrieTree) RemoveAll() {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.root.clean()
	t.size = 0
}
