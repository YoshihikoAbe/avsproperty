package avsproperty

import (
	"encoding/binary"
	"io"

	"golang.org/x/text/encoding"
)

const (
	binaryMagic          = 0xA042
	binaryMagicLong      = 0xA045
	maxValueSize         = 0x1000000
	arrayMask       byte = (1 << 6)

	maxMetaDepth = 100
)

var (
	errMetadata = propertyError("malformed metadata")
	errDatabody = propertyError("malformed databody")
)

func readBinary(prop *Property, rd io.Reader) error {
	prop.Settings.Format = FormatBinary
	state := binaryReadState{
		prop: prop,
		rd:   rd,
	}
	return state.read()
}

type binaryReadState struct {
	rd      io.Reader
	prop    *Property
	decoder *encoding.Decoder

	b8, b16 []byte
}

func (state *binaryReadState) read() error {
	if err := state.readHeader(); err != nil {
		return err
	}
	if err := state.readMetadata(); err != nil {
		return err
	}
	return state.readDatabody()
}

func (state *binaryReadState) readHeader() error {
	header := make([]byte, 4)
	if _, err := io.ReadFull(state.rd, header); err != nil {
		return err
	}

	if magic := binary.BigEndian.Uint16(header); magic == binaryMagic {
		state.prop.Settings.UseLongNodeNames = false
	} else if magic == binaryMagicLong {
		state.prop.Settings.UseLongNodeNames = true
	} else {
		return propertyError("invalid magic number")
	}

	if header[2] != ^header[3] {
		return propertyError("invalid encoding checksum")
	}
	if state.prop.Settings.Encoding = encodingById(header[2] >> 5); state.prop.Settings.Encoding == nil {
		return propertyError("invalid encoding")
	}
	state.decoder = state.prop.Encoding().decoder()

	return nil
}

func (state *binaryReadState) readMetadata() error {
	size, err := state.readSectionSize()
	if err != nil {
		return err
	}

	var (
		node  *Node
		depth int
	)
	for {
		id, err := state.rd.(io.ByteReader).ReadByte()
		if err != nil {
			return err
		}
		size--

		if id == typeEnd {
			if node != nil {
				return errMetadata
			}
			break
		} else if id == typeTraverseUp {
			depth--
			if depth < 0 {
				return errMetadata
			}
			node = node.parent
			continue
		}

		name := &NodeName{}
		read, err := name.readBinary(state.rd, state.prop.Settings.UseLongNodeNames)
		if err != nil {
			return err
		}
		size -= int64(read)

		if id == typeAttribute {
			if node == nil || node.SearchAttributeNodeName(name) != nil {
				return errMetadata
			}
			node.attributes = append(node.attributes, &Attribute{key: name})
			continue
		}

		depth++
		if depth > maxMetaDepth {
			return propertyError("max depth exceeded")
		}

		typ := lookupTypeById(id & ^arrayMask)
		if typ == nil {
			return errMetadata
		}

		newNode := &Node{
			name:     name,
			nodeType: typ,
			isArray:  id&arrayMask != 0,
		}
		if node == nil {
			if state.prop.Root != nil {
				return errMetadata
			}
			state.prop.Root = newNode
		} else {
			if err := node.AppendChild(newNode); err != nil {
				return err
			}
		}
		node = newNode
	}

	if size != 0 {
		if size > 4 || size < 0 {
			return errMetadata
		}
		// skip padding
		pad := make([]byte, 4)
		state.rd.Read(pad[:size])
	}

	return nil
}

func (state *binaryReadState) readDatabody() error {
	// skip
	b := make([]byte, 4)
	state.rd.Read(b)
	return state.prop.Root.Traverse(state.readDatabodyNode, nil)
}

func (state *binaryReadState) readDatabodyNode(node *Node) error {
	if node.nodeType != VoidNode {
		if err := state.readValue(node); err != nil {
			return err
		}
	}

	for _, attr := range node.attributes {
		s, err := state.readString()
		if err != nil {
			return err
		}
		attr.Value = s
	}

	return nil
}

func (state *binaryReadState) readValue(node *Node) (err error) {
	if node.nodeType == StrNode {
		s, err := state.readString()
		if err != nil {
			return err
		}
		node.value = s
	} else if node.nodeType == BinNode {
		b, err := state.readArray()
		if err != nil {
			return err
		}
		node.value = BinValue(b)
	} else if node.isArray {
		data, err := state.readArray()
		if err != nil {
			return err
		}
		if len(data)%node.nodeType.size != 0 {
			return errDatabody
		}

		slice := make([]any, len(data)/node.nodeType.size)
		for i := range slice {
			var k any
			k, err = node.nodeType.btv(data[i*node.nodeType.size:])
			if err != nil {
				break
			}
			slice[i] = k
		}
		node.value = slice
	} else {
		err = state.readAligned(node)
	}
	return
}

func (state *binaryReadState) read32(size int) ([]byte, error) {
	if size < 0 {
		return nil, errDatabody
	}

	aligned := size
	if r := aligned % 4; r != 0 {
		aligned += 4 - r
	}

	b := make([]byte, aligned)
	if _, err := io.ReadFull(state.rd, b); err != nil {
		return nil, err
	}

	return b[:size], nil
}

func (state *binaryReadState) readArray() (b []byte, err error) {
	if b, err = state.read32(4); err != nil {
		return
	}
	size := binary.BigEndian.Uint32(b)
	if size > maxValueSize {
		return nil, errDatabody
	}
	return state.read32(int(size))
}

func (state *binaryReadState) readString() (string, error) {
	b, err := state.readArray()
	if err != nil {
		return "", err
	}
	if len(b) == 0 {
		return "", errDatabody
	}
	b = b[:len(b)-1]

	if state.decoder == nil {
		return string(b), err
	}
	s, err := state.decoder.Bytes(b)
	return string(s), err
}

func (state *binaryReadState) refillBoundary(b []byte) ([]byte, error) {
	if len(b) != 0 {
		return b, nil
	}
	out := make([]byte, 4)
	_, err := io.ReadFull(state.rd, out)

	return out, err
}

func (state *binaryReadState) readAligned(node *Node) (err error) {
	var data []byte
	switch size := node.nodeType.size; size {
	case 0:
		data = make([]byte, 0)

	case 1:
		if state.b8, err = state.refillBoundary(state.b8); err != nil {
			return
		}
		data = state.b8[:1]
		state.b8 = state.b8[1:]

	case 2:
		if state.b16, err = state.refillBoundary(state.b16); err != nil {
			return
		}
		data = state.b16[:2]
		state.b16 = state.b16[2:]

	default:
		if data, err = state.read32(size); err != nil {
			return
		}
	}
	node.value, err = node.nodeType.btv(data)
	return
}

func (state *binaryReadState) readSectionSize() (int64, error) {
	data := make([]byte, 4)
	if _, err := io.ReadFull(state.rd, data); err != nil {
		return 0, err
	}

	size := int64(binary.BigEndian.Uint32(data))
	if size%4 != 0 {
		return 0, propertyError("invalid section alignment")
	}
	return size, nil
}
