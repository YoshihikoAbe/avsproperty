package avsproperty

import (
	"bytes"
	"io"
)

var packedLut = []int{
	-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1,
	-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1,
	-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1,
	0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, -1, -1, -1, -1, -1,
	-1, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25,
	26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, -1, -1, -1, -1, 37,
	-1, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52,
	53, 54, 55, 56, 57, 58, 59, 60, 61, 62, 63, -1, -1, -1, -1, -1,
}

const nodeNameSize = 36

type NodeName struct {
	data   [nodeNameSize]byte
	length int
}

func NewNodeName(name string) (*NodeName, error) {
	n := &NodeName{}
	return n, n.Set(name)
}

func (n *NodeName) Set(s string) error {
	if !validateNodeNameString(s) {
		return propertyError("illegal node name")
	}
	n.length = len(s)

	var (
		b byte
		k int
	)
	for i, ch := range s {
		cur := packedLut[ch&127]
		if cur < 0 {
			return propertyError("invalid character in node name")
		}

		switch i % 4 {
		case 0:
			b = byte(cur << 2)

		case 1:
			n.data[k] = (b | byte(cur>>4))
			k++
			b = byte(cur << 4)

		case 2:
			n.data[k] = (b | byte(cur>>2))
			k++
			b = byte(cur << 6)

		case 3:
			n.data[k] = b | byte(cur)
			k++
		}
	}
	if n.length%4 != 0 {
		n.data[k] = b
	}

	return nil
}

func (n *NodeName) Length() int {
	return n.length
}

func (a *NodeName) Equals(b *NodeName) bool {
	return bytes.Equal(a.data[:], b.data[:])
}

func (n *NodeName) String() string {
	const charset = "0123456789:ABCDEFGHIJKLMNOPQRSTUVWXYZ_abcdefghijklmnopqrstuvwxyz"

	b := make([]byte, n.length)
	k := 0
	for i := 0; i < n.length; i++ {
		switch i % 4 {
		case 0:
			b[i] = charset[n.data[k]>>2]

		case 1:
			b[i] = charset[((n.data[k]&3)<<4)|(n.data[k+1]>>4)]
			k++

		case 2:
			b[i] = charset[((n.data[k]&15)<<2)|(n.data[k+1]>>6)]
			k++

		case 3:
			b[i] = charset[n.data[k]&63]
			k++
		}
	}

	return string(b)
}

func (n *NodeName) readBinary(rd io.Reader) (uint8, error) {
	size, err := rd.(io.ByteReader).ReadByte()
	if err != nil {
		return 0, err
	}
	if size > nodeNameSize || size == 0 {
		return 0, propertyError("invalid node name size")
	}

	physicalSize := (size*6 + 7) / 8
	if _, err := io.ReadFull(rd, n.data[:physicalSize]); err != nil {
		return 0, err
	}

	// check if the name starts with "__"
	if size >= 2 && (uint16(n.data[0])<<8|uint16(n.data[1]))>>4 == 0x965 {
		return 0, propertyError("node name uses reserved name")
	}

	n.length = int(size)
	return physicalSize + 1, nil
}

func (n *NodeName) writeBinary(wr io.Writer) error {
	if err := wr.(io.ByteWriter).WriteByte(byte(n.length)); err != nil {
		return err
	}

	_, err := wr.Write(n.data[:n.packedSize()])
	return err
}

func (n *NodeName) packedSize() int {
	return (n.length*6 + 7) / 8
}

func validateNodeNameString(name string) bool {
	if size := len(name); size > nodeNameSize {
		return false
	} else if size >= 2 {
		return (uint(name[0])<<8 | uint(name[1])) != 0x5F5F // __
	} else {
		return size > 0
	}
}
