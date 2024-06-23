package avsproperty

import (
	"bytes"
	"encoding/hex"
	"encoding/xml"
	"io"
	"strconv"
	"strings"
)

func readXML(prop *Property, rd io.Reader) error {
	prop.Settings.Format = FormatXML
	prop.Settings.Encoding = EncodingUTF8
	decoder := xml.NewDecoder(rd)
	state := &xmlReadState{
		decoder: decoder,
		prop:    prop,
	}
	decoder.CharsetReader = state.readCharset
	return state.read()
}

type xmlReadState struct {
	decoder *xml.Decoder
	prop    *Property

	node  *Node
	count int
}

func (state *xmlReadState) read() error {
	for {
		token, err := state.decoder.Token()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		switch token := token.(type) {
		case xml.StartElement:
			err = state.readStartElement(token)

		case xml.CharData:
			if state.node == nil {
				continue
			}
			err = state.readCharData(token)

		case xml.EndElement:
			state.node = state.node.parent
		}
		if err != nil {
			return err
		}
	}
}

func (state *xmlReadState) readStartElement(elem xml.StartElement) error {
	err := state.newNode(elem)
	if err != nil {
		return err
	}

	for _, attr := range elem.Attr {
		if err := state.readAttrib(attr); err != nil {
			return err
		}
	}

	return nil
}

func (state *xmlReadState) readAttrib(attr xml.Attr) (err error) {
	node := state.node
	nt := node.nodeType
	switch attr.Name.Local {
	case "__type":
		nt = lookupTypeByName(attr.Value)
		if nt == nil {
			return node.error("invalid node type: " + attr.Value)
		}
		node.nodeType = nt

		// these types support empty values
		if nt == StrNode {
			node.value = ""
		} else if nt == BinNode {
			node.value = BinValue{}
		}

	case "__count":
		if nt == VoidNode || nt == StrNode || nt == BinNode {
			return node.error("__count attribute out of place")
		}
		state.count, err = strconv.Atoi(attr.Value)
		node.isArray = true

	case "__size":
		// this attribute seems to be superfluous, but we'll
		// still make sure that it's in the right place
		if nt != BinNode {
			return node.error("__size attribute out of place")
		}

	default:
		err = node.SetAttribute(attr.Name.Local, attr.Value)
	}
	return
}

func (state *xmlReadState) newNode(elem xml.StartElement) (err error) {
	if state.node == nil {
		state.node, err = NewNode(elem.Name.Local)
		state.prop.Root = state.node
	} else {
		state.node, err = state.node.NewNode(elem.Name.Local)
	}

	return
}

func (state *xmlReadState) readCharData(cd xml.CharData) error {
	nt := state.node.nodeType
	if nt != VoidNode && nt != StrNode {
		cd = bytes.TrimSpace(cd)
	}
	switch nt {
	case VoidNode:
		if len(bytes.TrimSpace(cd)) == 0 {
			break
		}
		state.node.nodeType = StrNode
		fallthrough
	case StrNode:
		state.node.value = string(cd)

	case BinNode:
		b, err := hex.DecodeString(string(cd))
		if err != nil {
			return err
		}
		state.node.value = BinValue(b)

	default:
		if state.node.isArray {
			split := strings.Split(string(cd), " ")
			if len(split) != nt.count*state.count {
				return state.node.error("invalid number of elements in value")
			}

			slice := make([]any, state.count)
			for i := range slice {
				var s string
				if nt.count > 1 {
					start := i * nt.count
					s = strings.Join(split[start:start+nt.count], " ")
				} else {
					s = split[i]
				}

				v, err := nt.stv(s)
				if err != nil {
					return err
				}
				slice[i] = v
			}
			state.node.value = slice
		} else {
			v, err := state.node.nodeType.stv(string(cd))
			if err != nil {
				return err
			}
			state.node.value = v
		}
	}

	return nil
}

func (state *xmlReadState) readCharset(charset string, rd io.Reader) (io.Reader, error) {
	encoding := EncodingByName(charset)
	if encoding == nil {
		return nil, propertyError("encoding not found")
	}
	state.prop.Settings.Encoding = encoding
	return encoding.charset.NewDecoder().Reader(rd), nil
}
