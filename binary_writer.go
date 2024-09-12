package avsproperty

import (
	"encoding/binary"
	"io"
	"reflect"
	"strconv"

	"golang.org/x/text/encoding"
)

func writeBinary(prop *Property, wr io.Writer) error {
	prop.Settings.Format = FormatBinary
	state := binaryWriteState{
		prop:    prop,
		wr:      wr,
		encoder: prop.Encoding().encoder(),
	}
	return state.write()
}

type binaryWriteState struct {
	prop *Property
	wr   io.Writer

	databody []byte
	i16, i8  int
	encoder  *encoding.Encoder
}

func (state *binaryWriteState) write() error {
	if err := state.writeHeader(); err != nil {
		return err
	}

	if err := state.writeMetadata(); err != nil {
		return err
	}

	if err := state.writeDatabody(); err != nil {
		return err
	}
	return nil
}

func (state *binaryWriteState) writeHeader() error {
	magic := binaryMagic
	if state.prop.Settings.UseLongNodeNames {
		magic = binaryMagicLong
	}
	if err := binary.Write(state.wr, binary.BigEndian, uint16(magic)); err != nil {
		return err
	}

	encoding := uint16(state.prop.Encoding().codepage << 5)
	err := binary.Write(state.wr, binary.BigEndian, encoding<<8|(^encoding&0xFF))
	return err
}

func (state *binaryWriteState) writeMetadata() error {
	size, padding, err := state.calculateMetadataSize(state.prop.Root)
	if err != nil {
		return err
	}

	if err := binary.Write(state.wr, binary.BigEndian, uint32(size)); err != nil {
		return err
	}

	if err := state.prop.Root.Traverse(state.writeMetadataStart, state.writeMetadataEnd); err != nil {
		return err
	}

	if err := state.wr.(io.ByteWriter).WriteByte(typeEnd); err != nil {
		return err
	}

	if padding > 0 {
		b := make([]byte, padding)
		if _, err := state.wr.Write(b); err != nil {
			return err
		}
	}

	return nil
}

func (state *binaryWriteState) writeMetadataStart(node *Node) error {
	id := byte(node.nodeType.id)
	if node.isArray {
		id |= arrayMask
	}

	wr := state.wr.(io.ByteWriter)

	if err := wr.WriteByte(id); err != nil {
		return err
	}

	long := state.prop.Settings.UseLongNodeNames
	if err := node.name.writeBinary(state.wr, long); err != nil {
		return err
	}

	for _, attrib := range node.attributes {
		if err := wr.WriteByte(typeAttribute); err != nil {
			return err
		}
		if err := attrib.key.writeBinary(state.wr, long); err != nil {
			return err
		}
	}

	return nil
}

func (state *binaryWriteState) writeMetadataEnd(node *Node) error {
	return state.wr.(io.ByteWriter).WriteByte(typeTraverseUp)
}

func (state *binaryWriteState) calculateMetadataSize(node *Node) (n int, padding int, err error) {
	node.Traverse(func(node *Node) error {
		// start, end, name size, and name
		long := state.prop.Settings.UseLongNodeNames
		n += 3 + node.name.binarySize(long)
		for _, attrib := range node.attributes {
			n += 2 + attrib.key.binarySize(long)
		}
		return nil
	}, nil)
	// EOF marker
	n++
	if r := n % 4; r != 0 {
		padding = 4 - r
		n += padding
	}
	return
}

func (state *binaryWriteState) writeDatabody() error {
	if err := state.prop.Root.Traverse(state.writeDatabodyNode, nil); err != nil {
		return err
	}

	if err := binary.Write(state.wr, binary.BigEndian, uint32(len(state.databody))); err != nil {
		return err
	}

	_, err := state.wr.Write(state.databody)
	return err
}

func (state *binaryWriteState) appendPadding() {
	if r := len(state.databody) % 4; r != 0 {
		state.databody = append(state.databody, make([]byte, 4-r)...)
	}
}

func (state *binaryWriteState) append32(b []byte) {
	state.databody = append(state.databody, b...)
	state.appendPadding()
}

func (state *binaryWriteState) allocate32(size int) []byte {
	old := len(state.databody)

	state.databody = append(state.databody, make([]byte, size)...)
	state.appendPadding()

	return state.databody[old:]
}

func (state *binaryWriteState) alignBoundary(i *int) {
	if *i%4 == 0 {
		*i = len(state.databody)
		state.allocate32(4)
	}
}

func (state *binaryWriteState) allocate(size int) (b []byte) {
	switch size {
	case 0:
		return nil

	case 1:
		state.alignBoundary(&state.i8)
		b = state.databody[state.i8:]
		state.i8 += 1

	case 2:
		state.alignBoundary(&state.i16)
		b = state.databody[state.i16:]
		state.i16 += 2

	default:
		b = state.allocate32(size)
	}
	return
}

func (state *binaryWriteState) appendU32(i uint32) {
	state.databody = binary.BigEndian.AppendUint32(state.databody, uint32(i))
}

func (state *binaryWriteState) writeString(s string) (err error) {
	var b []byte
	if state.encoder == nil {
		b = []byte(s)
	} else {
		b, err = state.encoder.Bytes([]byte(s))
		if err != nil {
			return
		}
	}
	// null-terminated
	b = append(b, 0)

	state.appendU32(uint32(len(b)))
	state.append32(b)

	return
}

func (state *binaryWriteState) writeArray(node *Node) {
	v := reflect.ValueOf(node.value)
	nt := node.nodeType
	size := v.Len() * nt.size

	state.appendU32(uint32(size))
	b := state.allocate32(size)
	for i := 0; i < v.Len(); i++ {
		node.nodeType.vtb(v.Index(i).Interface(), b[i*nt.size:])
	}
}

func (state *binaryWriteState) writeValue(node *Node) error {
	if size := node.ArrayLength() * node.nodeType.size; size > maxValueSize {
		return node.error("value too large: " + strconv.Itoa(size))
	}

	if node.isArray {
		state.writeArray(node)
	} else if node.nodeType == StrNode {
		if err := state.writeString(node.StringValue()); err != nil {
			return err
		}
	} else if node.nodeType == BinNode {
		b := node.BinaryValue()
		state.appendU32(uint32(len(b)))
		state.append32(b)
	} else {
		node.nodeType.vtb(node.value, state.allocate(node.nodeType.size))
	}
	return nil
}

func (state *binaryWriteState) writeDatabodyNode(node *Node) error {
	if node.nodeType != VoidNode {
		if node.value == nil {
			return node.error("node contains a nil value")
		}
		if err := state.writeValue(node); err != nil {
			return err
		}
	}

	for _, attib := range node.attributes {
		if err := state.writeString(attib.Value); err != nil {
			return err
		}
	}

	return nil
}
