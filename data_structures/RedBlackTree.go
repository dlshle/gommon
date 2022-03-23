package data_structures

import (
	"fmt"
)

type IComparable interface {
	Compare(a IComparable) int
}

type color bool

const (
	black, red color = true, false
)

type Tree struct {
	root *Node
}

type Node struct {
	Key    IComparable
	Value  interface{}
	color  color
	Left   *Node
	Right  *Node
	Parent *Node
}

func NewRedBlackTree() *Tree {
	return &Tree{}
}

// stop on cb returns false
func (tree *Tree) ForEach(cb func(each interface{}) bool) {
	inOrderTraverse(tree.root, func(node *Node) bool {
		return cb(node.Value)
	})
}

func inOrderTraverse(node *Node, cb func(node *Node) bool) {
	if node.Left != nil {
		inOrderTraverse(node.Left, cb)
	}
	if !cb(node) {
		return
	}
	if node.Right != nil {
		inOrderTraverse(node.Right, cb)
	}
}

func (tree *Tree) Clear() {
	inOrderTraverse(tree.root, func(node *Node) bool {
		node.Left = nil
		node.Right = nil
		node.Value = nil
		return true
	})
	tree.root = nil
}

func (tree *Tree) PutKeyAsValue(n IComparable) {
	tree.Put(n, n)
}

func (tree *Tree) Put(key IComparable, value interface{}) {
	var insertedNode *Node
	if tree.root == nil {
		tree.root = &Node{Key: key, Value: value, color: red}
		insertedNode = tree.root
		tree.insertCase1(insertedNode)
		return
	}
	node := tree.root
	loop := true
	insertedNode = &Node{Key: key, Value: value, color: red}
	for loop {
		compare := key.Compare(node.Key)
		switch {
		case compare == 0:
			node.Key = key
			node.Value = value
			return
		case compare < 0:
			if node.Left == nil {
				node.Left = insertedNode
				loop = false
				break
			}
			node = node.Left
		case compare > 0:
			if node.Right == nil {
				node.Right = insertedNode
				loop = false
				break
			}
			node = node.Right
		}
	}
	insertedNode.Parent = node
	tree.insertCase1(insertedNode)
}

func (tree *Tree) Get(key IComparable) (interface{}, bool) {
	var node, found = tree.lookup(key)
	if node != nil {
		return node.Value, found
	}
	return nil, found
}

func (tree *Tree) Empty() bool {
	return tree.root == nil
}

func (tree *Tree) String() string {
	str := "RedBlackTree\n"
	if !tree.Empty() {
		output(tree.root, "", true, &str)
	}
	return str
}

func (node *Node) String() string {
	return fmt.Sprintf("%v", node.Key)
}

func output(node *Node, prefix string, isTail bool, str *string) {
	if node.Right != nil {
		newPrefix := prefix
		if isTail {
			newPrefix += "│   "
		} else {
			newPrefix += "    "
		}
		output(node.Right, newPrefix, false, str)
	}
	*str += prefix
	if isTail {
		*str += "└── "
	} else {
		*str += "┌── "
	}
	*str += node.String() + "\n"
	if node.Left != nil {
		newPrefix := prefix
		if isTail {
			newPrefix += "    "
		} else {
			newPrefix += "│   "
		}
		output(node.Left, newPrefix, true, str)
	}
}

func (tree *Tree) lookup(key IComparable) (*Node, bool) {
	node := tree.root
	for node != nil {
		compare := key.Compare(node.Key)
		switch {
		case compare == 0:
			return node, true
		case compare < 0:
			node = node.Left
		case compare > 0:
			node = node.Right
		}
	}
	return nil, false
}

func (node *Node) grandparent() *Node {
	if node != nil && node.Parent != nil {
		return node.Parent.Parent
	}
	return nil
}

func (node *Node) uncle() *Node {
	if node == nil || node.Parent == nil || node.Parent.Parent == nil {
		return nil
	}
	return node.Parent.sibling()
}

func (node *Node) sibling() *Node {
	if node == nil || node.Parent == nil {
		return nil
	}
	if node == node.Parent.Left {
		return node.Parent.Right
	}
	return node.Parent.Left
}

func (tree *Tree) rotateLeft(node *Node) {
	right := node.Right
	tree.replaceNode(node, right)
	node.Right = right.Left
	if right.Left != nil {
		right.Left.Parent = node
	}
	right.Left = node
	node.Parent = right
}

func (tree *Tree) rotateRight(node *Node) {
	left := node.Left
	tree.replaceNode(node, left)
	node.Left = left.Right
	if left.Right != nil {
		left.Right.Parent = node
	}
	left.Right = node
	node.Parent = left
}

func (tree *Tree) replaceNode(old *Node, new *Node) {
	if old.Parent == nil {
		tree.root = new
	} else {
		if old == old.Parent.Left {
			old.Parent.Left = new
		} else {
			old.Parent.Right = new
		}
	}
	if new != nil {
		new.Parent = old.Parent
	}
}

func (tree *Tree) insertCase1(node *Node) {
	if node.Parent == nil {
		node.color = black
		return
	}
	tree.insertCase2(node)
}

func (tree *Tree) insertCase2(node *Node) {
	if nodeColor(node.Parent) == black {
		return
	}
	tree.insertCase3(node)
}

func (tree *Tree) insertCase3(node *Node) {
	uncle := node.uncle()
	if nodeColor(uncle) == red {
		node.Parent.color = black
		uncle.color = black
		node.grandparent().color = red
		tree.insertCase1(node.grandparent())
		return
	}
	tree.insertCase4(node)
}

func (tree *Tree) insertCase4(node *Node) {
	grandparent := node.grandparent()
	if node == node.Parent.Right && node.Parent == grandparent.Left {
		tree.rotateLeft(node.Parent)
		node = node.Left
	} else if node == node.Parent.Left && node.Parent == grandparent.Right {
		tree.rotateRight(node.Parent)
		node = node.Right
	}
	tree.insertCase5(node)
}

func (tree *Tree) insertCase5(node *Node) {
	node.Parent.color = black
	grandparent := node.grandparent()
	grandparent.color = red
	if node == node.Parent.Left && node.Parent == grandparent.Left {
		tree.rotateRight(grandparent)
	} else if node == node.Parent.Right && node.Parent == grandparent.Right {
		tree.rotateLeft(grandparent)
	}
}

func (tree *Tree) Left() *Node {
	var parent *Node
	current := tree.root
	for current != nil {
		parent = current
		current = current.Left
	}
	return parent
}

func (tree *Tree) Right() *Node {
	var parent *Node
	current := tree.root
	for current != nil {
		parent = current
		current = current.Right
	}
	return parent
}

func nodeColor(node *Node) color {
	if node == nil {
		return black
	}
	return node.color
}
