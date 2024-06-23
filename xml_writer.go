package avsproperty

import (
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"reflect"
	"strconv"

	"golang.org/x/text/encoding"
)

func writeXML(prop *Property, wr io.Writer) error {
	encoding := prop.Encoding()
	state := &xmlWriteState{
		wr:       wr,
		encoding: encoding,
		encoder:  encoding.encoder(),
		pretty:   prop.Settings.Format == FormatPrettyXML,
	}

	return state.write(prop.Root)
}

type xmlWriteState struct {
	wr       io.Writer
	encoding *Encoding
	encoder  *encoding.Encoder
	pretty   bool

	depth int
}

func (state *xmlWriteState) write(node *Node) error {
	if err := state.writeDecl(); err != nil {
		return err
	}
	return node.Traverse(state.startNode, state.endNode)
}

func (state *xmlWriteState) startNode(node *Node) error {
	if state.pretty {
		if err := state.writeIndent(); err != nil {
			return err
		}
	}
	state.depth++

	if err := state.wr.(io.ByteWriter).WriteByte('<'); err != nil {
		return err
	}
	if _, err := io.WriteString(state.wr, node.name.String()); err != nil {
		return err
	}

	return state.writeInnerNode(node)
}

func (state *xmlWriteState) endNode(node *Node) (err error) {
	state.depth--
	if state.pretty && len(node.children) > 0 {
		if err = state.writeIndent(); err != nil {
			return
		}
	}

	if _, err = io.WriteString(state.wr, "</"); err != nil {
		return
	}
	if _, err = io.WriteString(state.wr, node.name.String()); err != nil {
		return
	}
	if err = state.wr.(io.ByteWriter).WriteByte('>'); err != nil {
		return
	}

	if state.pretty {
		state.wr.(io.ByteWriter).WriteByte('\n')
	}

	return
}

func (state *xmlWriteState) writeInnerNode(node *Node) error {
	if node.nodeType != VoidNode {
		if err := state.writeAttrib("__type", node.nodeType.names[0], false); err != nil {
			return err
		}

		if node.isArray || node.nodeType == BinNode {
			var (
				name string
				size int
			)
			if node.isArray {
				name = "__count"
				size = node.ArrayLength()
			} else {
				name = "__size"
				size = len(node.BinaryValue())
			}
			if err := state.writeAttrib(name, strconv.Itoa(size), false); err != nil {
				return err
			}
		}
	}

	for _, attrib := range node.attributes {
		if err := state.writeAttrib(attrib.key.String(), attrib.Value, true); err != nil {
			return err
		}
	}

	if err := state.wr.(io.ByteWriter).WriteByte('>'); err != nil {
		return err
	}

	if node.nodeType != VoidNode {
		return state.writeValue(node)
	}

	if state.pretty && len(node.children) > 0 {
		if err := state.wr.(io.ByteWriter).WriteByte('\n'); err != nil {
			return err
		}
	}

	return nil
}

func (state *xmlWriteState) writeValue(node *Node) error {
	if node.value == nil {
		return propertyError("node has a nil value")
	}

	rv := reflect.ValueOf(node.value)
	switch v := node.value.(type) {
	case BinValue:
		_, err := io.WriteString(state.wr, hex.EncodeToString(v))
		return err

	case string:
		return state.writeString(v)

	default:
		return state.writeValueRecursive(rv)
	}
}

func (state *xmlWriteState) writeValueRecursive(rv reflect.Value) error {
	if v, ok := rv.Interface().(net.IP); ok {
		_, err := io.WriteString(state.wr, v.String())
		return err
	}

	kind := rv.Kind()
	if kind == reflect.Interface {
		rv = rv.Elem()
		kind = rv.Kind()
	}

	if kind == reflect.Slice || kind == reflect.Array {
		for i := 0; i < rv.Len(); i++ {
			if i > 0 {
				if err := state.wr.(io.ByteWriter).WriteByte(' '); err != nil {
					return err
				}
			}

			if err := state.writeValueRecursive(rv.Index(i)); err != nil {
				return err
			}
		}
		return nil
	}

	_, err := fmt.Fprint(state.wr, rv)
	return err
}

func (state *xmlWriteState) writeAttrib(k, v string, encode bool) error {
	if err := state.wr.(io.ByteWriter).WriteByte(' '); err != nil {
		return err
	}
	if _, err := io.WriteString(state.wr, k); err != nil {
		return err
	}
	if _, err := io.WriteString(state.wr, "=\""); err != nil {
		return err
	}

	if encode {
		if err := state.writeString(v); err != nil {
			return err
		}
	} else {
		if err := xml.EscapeText(state.wr, []byte(v)); err != nil {
			return err
		}
	}

	return state.wr.(io.ByteWriter).WriteByte('"')
}

func (state *xmlWriteState) writeString(s string) error {
	var b []byte
	if state.encoder == nil {
		b = []byte(s)
	} else {
		encoded, err := state.encoder.Bytes([]byte(s))
		if err != nil {
			return err
		}
		b = encoded
	}
	return xml.EscapeText(state.wr, b)
}

func (state *xmlWriteState) writeDecl() (err error) {
	if _, err = io.WriteString(state.wr, "<?xml version=\"1.0\""); err != nil {
		return
	}
	if e := state.encoding; e != EncodingNone {
		if _, err = io.WriteString(state.wr, " encoding=\""); err != nil {
			return
		}
		if _, err = io.WriteString(state.wr, e.name); err != nil {
			return
		}
		if err = state.wr.(io.ByteWriter).WriteByte('"'); err != nil {
			return
		}
	}
	if _, err = io.WriteString(state.wr, "?>"); err != nil {
		return err
	}

	if state.pretty {
		err = state.wr.(io.ByteWriter).WriteByte('\n')
	}
	return
}

func (state *xmlWriteState) writeIndent() error {
	for i := 0; i < state.depth; i++ {
		if _, err := io.WriteString(state.wr, "    "); err != nil {
			return err
		}
	}
	return nil
}
