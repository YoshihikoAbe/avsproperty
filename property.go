package avsproperty

import (
	"bufio"
	"io"
	"net"
	"os"
	"reflect"
)

type propertyError string

func (err propertyError) Error() string {
	return "avsproperty: " + string(err)
}

type PropertyFormat int

const (
	FormatBinary PropertyFormat = iota
	FormatXML
	FormatPrettyXML
)

type PropertySettings struct {
	Format           PropertyFormat
	Encoding         *Encoding
	UseLongNodeNames bool
}

// Property represents a property tree.
type Property struct {
	// Settings defines how a property should be serialized.
	// After a read operation (successful or not) this field
	// is automatically updatedwith the settings of the
	// property that was read.
	Settings PropertySettings

	Root *Node
}

// NewProperty creates a new Property with the default settings
// and creates a new root Node with the specified name
func NewProperty(root string) (*Property, error) {
	node, err := NewNode(root)
	if err != nil {
		return nil, err
	}
	return &Property{
		Root: node,
	}, nil
}

// Read reads a document from the Reader into the Property.
// The format of the document is automatically inferred from
// the first byte in the stream
func (p *Property) Read(rd io.Reader) error {
	p.Root = nil

	if _, ok := rd.(io.ByteScanner); !ok {
		rd = bufio.NewReader(rd)
	}

	scan := rd.(io.ByteScanner)
	magic, err := scan.ReadByte()
	if err != nil {
		return err
	}
	scan.UnreadByte()

	var reader func(*Property, io.Reader) error
	switch magic {
	case binaryMagic >> 8:
		reader = readBinary
	case '<':
		reader = readXML
	default:
		return propertyError("could not detect format")
	}
	return reader(p, rd)
}

// Write serializes and writes the property to the Writer.
// The way in which the Property is serialized is defined
// by its Settings field.
func (p *Property) Write(wr io.Writer) error {
	if p.Root == nil {
		return propertyError("property is empty")
	}

	if _, ok := wr.(io.ByteWriter); !ok {
		bio := bufio.NewWriter(wr)
		defer bio.Flush()
		wr = bio
	}

	var writer func(*Property, io.Writer) error
	switch p.Settings.Format {
	case FormatBinary:
		writer = writeBinary
	case FormatPrettyXML:
		fallthrough
	case FormatXML:
		writer = writeXML
	default:
		panic("invalid format")
	}
	return writer(p, wr)
}

// Write serializes and writes the property to a file
// at the specified path. The way in which the Property
// should be serialized is defined by its Settings field.
func (p *Property) WriteFile(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return p.Write(f)
}

// Read reads a document from a file at the specified path into the
// Property. The format of the document is automatically inferred
// from the first byte in the file
func (p *Property) ReadFile(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return p.Read(f)
}

// Encoding returns the Property's encoding. If Settings.Encoding is
// nil, EncodingNone is returned instead
func (p *Property) Encoding() *Encoding {
	if p.Settings.Encoding == nil {
		return EncodingNone
	}
	return p.Settings.Encoding
}

// Attribute represents an attribute in a property tree
type Attribute struct {
	key   *NodeName
	Value string
}

func (a Attribute) Key() *NodeName {
	return a.key
}

// Attribute represents a node in a property tree
type Node struct {
	parent *Node

	name     *NodeName
	nodeType *NodeType

	isArray bool
	value   any

	children   []*Node
	attributes []*Attribute
}

// NewNode creates a new Node using the supplied name
func NewNode(name string) (*Node, error) {
	n, err := NewNodeName(name)
	if err != nil {
		return nil, err
	}
	return &Node{
		name:     n,
		nodeType: VoidNode,
	}, nil
}

// NewNode creates a new Node using the supplied name and value
func NewNodeWithValue(name string, value any) (*Node, error) {
	n, err := NewNode(name)
	if err != nil {
		return nil, err
	}
	if err := n.SetValue(value); err != nil {
		return nil, err
	}
	return n, nil
}

func (n *Node) Parent() *Node {
	return n.parent
}

func (n *Node) Name() *NodeName {
	return n.name
}

func (n *Node) Type() *NodeType {
	return n.nodeType
}

func (n *Node) Value() any {
	return n.value
}

func (n *Node) IsArray() bool {
	return n.isArray
}

// Children returns a list of the Node's children. The returned slice is
// owned by the Node and should not be modified in any way.
// This function may return nil if the Node does not have any children
func (n *Node) Children() []*Node {
	return n.children
}

// SearchChildren returns a list of the Node's children
// with the specified name
func (n *Node) SearchChildren(name string) []*Node {
	if name, err := NewNodeName(name); err != nil {
		return nil
	} else {
		return n.SearchChildrenNodeName(name)
	}
}

// SearchChildrenNodeName returns a list of the Node's children
// with the specified name
func (n *Node) SearchChildrenNodeName(name *NodeName) []*Node {
	children := make([]*Node, 0)

	for _, c := range n.children {
		if c.name.Equals(name) {
			children = append(children, c)
		}
	}

	return children
}

// SearchChild returns the first child of the Node with the
// specified name, or nil if no child is found
func (n *Node) SearchChild(name string) *Node {
	if name, err := NewNodeName(name); err != nil {
		return nil
	} else {
		return n.SearchChildNodeName(name)
	}
}

// SearchChildNodeName returns the first child of the Node with the
// specified name, or nil if no child is found
func (n *Node) SearchChildNodeName(name *NodeName) *Node {
	for _, c := range n.children {
		if c.name.Equals(name) {
			return c
		}
	}
	return nil
}

// ChildValue returns the value of the first child of the
// Node with the specified name, or nil if no child is found
func (n *Node) ChildValue(name string) any {
	if name, err := NewNodeName(name); err != nil {
		return nil
	} else {
		return n.ChildValueNodeName(name)
	}
}

// ChildValueNodeName returns the value of the first child of the
// Node with the specified name, or nil if no child is found
func (n *Node) ChildValueNodeName(name *NodeName) any {
	child := n.SearchChildNodeName(name)
	if child != nil {
		return child.value
	}
	return nil
}

// Attributes returns a list of the Node's attributes. The returned slice is owned
// by the Node and should not be modified in any way.
// This function may return nil the Node does not have any attributes
func (n *Node) Attributes() []*Attribute {
	return n.attributes
}

// SearchAttributeNodeName returns an attribute with the
// specified key, or nil if no attribute is found
func (n *Node) SearchAttribute(k string) *Attribute {
	if k, err := NewNodeName(k); err != nil {
		return nil
	} else {
		return n.SearchAttributeNodeName(k)
	}
}

// SearchAttributeNodeName returns an attribute with the
// specified key, or nil if no attribute is found
func (n *Node) SearchAttributeNodeName(k *NodeName) *Attribute {
	for _, a := range n.attributes {
		if a.key.Equals(k) {
			return a
		}
	}
	return nil
}

// AttributeValue returns the value of an attribute with the
// specified key. If the attribute is not present, an empty
// string is returned instead
func (n *Node) AttributeValue(k string) string {
	if k, err := NewNodeName(k); err != nil {
		return ""
	} else {
		return n.AttributeValueNodeName(k)
	}
}

// AttributeValueNodeName returns the value of an attribute with the
// specified key. If the attribute is not present, an empty
// string is returned instead
func (n *Node) AttributeValueNodeName(k *NodeName) string {
	for _, a := range n.attributes {
		if a.key.Equals(k) {
			return a.Value
		}
	}
	return ""
}

// ArrayLength returns the length of an array or string value.
// If the Node does not contain an array or string value, 1
// is returned instead.
func (n *Node) ArrayLength() int {
	if (n.nodeType != StrNode && n.nodeType != BinNode) &&
		!n.isArray || n.value == nil {
		return 1
	}
	return reflect.ValueOf(n.value).Len()
}

// IntValue returns the Node's value as a signed integer, or 0 if the
// Node does not contain a signed integer value.
func (n *Node) IntValue() int64 {
	switch v := n.value.(type) {
	case int8:
		return int64(v)
	case int16:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	default:
		return 0
	}
}

// IntValue returns the Node's value as an unsigned integer, or 0 if the
// Node does not contain an unsigned integer value.
func (n *Node) UintValue() uint64 {
	switch v := n.value.(type) {
	case uint8:
		return uint64(v)
	case uint16:
		return uint64(v)
	case uint32:
		return uint64(v)
	case uint64:
		return v
	default:
		return 0
	}
}

// StringValue returns the Node's value as a string, or an empty string
// if the Node does not contain a string value.
func (n *Node) StringValue() string {
	s, _ := n.value.(string)
	return s
}

// BinaryValue returns the Node's value as a BinValue, or nil
// if the Node does not contain a BinValue.
func (n *Node) BinaryValue() BinValue {
	b, _ := n.value.(BinValue)
	return b
}

// AppendChild adds c as the last child of the Node.
func (n *Node) AppendChild(c *Node) error {
	if c.parent != nil {
		return n.error("child already has a parent")
	}

	if n.nodeType != VoidNode {
		n.nodeType = VoidNode
		n.value = nil
	}

	c.parent = n
	n.children = append(n.children, c)

	return nil
}

// NewNode creates a new Node, and adds it as the last child of the Node.
func (n *Node) NewNode(name string) (*Node, error) {
	c, err := NewNode(name)
	if err != nil {
		return nil, err
	}

	c.parent = n
	n.children = append(n.children, c)

	n.nodeType = VoidNode
	n.value = nil

	return c, nil
}

// NewNode creates a new Node with a value, and adds it as the last child of the Node.
func (n *Node) NewNodeWithValue(name string, value any) (*Node, error) {
	c, err := NewNodeWithValue(name, value)
	if err != nil {
		return nil, err
	}

	c.parent = n
	n.children = append(n.children, c)

	return c, nil
}

// SetAttribute creates an attribute using k and v as the key and value
// respectively. If an attribute with the same key is already present,
// its value will be updated with v.
func (n *Node) SetAttribute(k, v string) error {
	name, err := NewNodeName(k)
	if err != nil {
		return err
	}

	if a := n.SearchAttributeNodeName(name); a != nil {
		a.Value = v
		return nil
	}

	n.attributes = append(n.attributes, &Attribute{name, v})

	return nil
}

// SetValue sets the Node's value to v. Refer to type.go to see how
// Go types are mapped to Property types.
func (n *Node) SetValue(v any) error {
	if len(n.children) > 0 {
		return n.error("cannot assign value to node that has children")
	}

	if v, ok := v.(net.IP); ok && v.To4() == nil {
		return n.error("invalid ip size")
	}

	rt := reflect.TypeOf(v)
	isArray := false
	if (rt != BinNode.rt && rt != IPv4Node.rt) && (rt != nil && rt.Kind() == reflect.Slice) {
		isArray = true
		rt = rt.Elem()
	}

	pt, ok := typeLut[rt]
	if !ok {
		return n.error("invalid Go type")
	}
	if (pt == StrNode || pt == BinNode) && isArray {
		return n.error("invalid array type")
	}

	n.nodeType = pt
	n.value = v
	n.isArray = isArray

	return nil
}

func (n *Node) Traverse(start, end func(*Node) error) error {
	if start != nil {
		if err := start(n); err != nil {
			return err
		}
	}

	for _, child := range n.children {
		if err := child.Traverse(start, end); err != nil {
			return err
		}
	}

	if end != nil {
		return end(n)
	}
	return nil
}

func (n *Node) error(s string) error {
	return propertyError(n.name.String() + ": " + s)
}
