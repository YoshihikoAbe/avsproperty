package avsproperty

import (
	"encoding/binary"
	"math"
	"net"
	"reflect"
	"strconv"
	"strings"
	"unsafe"
)

// control types
const (
	typeVoid       = 1
	typeAttribute  = 46
	typeTraverseUp = 254
	typeEnd        = 255
)

type (
	bytesToValue  func([]byte) (any, error)
	valueToBytes  func(any, []byte)
	stringToValue func(string) (any, error)
)

type NodeType struct {
	id    int
	names []string
	size  int
	count int
	rt    reflect.Type

	btv bytesToValue
	vtb valueToBytes
	stv stringToValue
}

func (t NodeType) Name() string {
	return t.names[0]
}

type (
	// BinValue represents the value of a binary node.
	BinValue []byte
	// BoolValue represents the value of a bool node
	BoolValue bool
	// TimeValue represents the value of a time node
	TimeValue uint32
)

func (bv BoolValue) String() string {
	if bv {
		return "1"
	}
	return "0"
}

// node types
var (
	VoidNode = &NodeType{
		typeVoid, []string{"void"}, -1, -1, nil, nil, nil, nil,
	}
	S8Node = &NodeType{
		2, []string{"s8"}, 1, 1, reflect.TypeOf(int8(0)), int8BytesToValue, int8ValueToBytes, intStringToValue[int8],
	}
	U8Node = &NodeType{
		3, []string{"u8"}, 1, 1, reflect.TypeOf(uint8(0)), uint8BytesToValue, uint8ValueToBytes, uintStringToValue[uint8],
	}
	S16Node = &NodeType{
		4, []string{"s16"}, 2, 1, reflect.TypeOf(int16(0)), int16BytesToValue, int16ValueToBytes, intStringToValue[int16],
	}
	U16Node = &NodeType{
		5, []string{"u16"}, 2, 1, reflect.TypeOf(uint16(0)), uint16BytesToValue, uint16ValueToBytes, uintStringToValue[uint16],
	}
	S32Node = &NodeType{
		6, []string{"s32"}, 4, 1, reflect.TypeOf(int32(0)), int32BytesToValue, int32ValueToBytes, intStringToValue[int32],
	}
	U32Node = &NodeType{
		7, []string{"u32"}, 4, 1, reflect.TypeOf(uint32(0)), uint32BytesToValue, uint32ValueToBytes, uintStringToValue[uint32],
	}
	S64Node = &NodeType{
		8, []string{"s64"}, 8, 1, reflect.TypeOf(int64(0)), int64BytesToValue, int64ValueToBytes, intStringToValue[int64],
	}
	U64Node = &NodeType{
		9, []string{"u64"}, 8, 1, reflect.TypeOf(uint64(0)), uint64BytesToValue, uint64ValueToBytes, uintStringToValue[uint64],
	}
	BinNode = &NodeType{
		10, []string{"bin", "binary"}, 1, 1, reflect.TypeOf(BinValue{}), nil, nil, nil,
	}
	StrNode = &NodeType{
		11, []string{"str", "string"}, 1, 1, reflect.TypeOf(""), nil, nil, nil,
	}
	IPv4Node = &NodeType{
		12, []string{"ip4"}, 4, 1, reflect.TypeOf(net.IP{}), ip4BytesToValue, ip4ValueToBytes, ip4StringToValue,
	}
	TimeNode = &NodeType{
		13, []string{"time"}, 4, 1, reflect.TypeOf(TimeValue(0)), timeBytesToValue, timeValueToBytes, uintStringToValue[TimeValue],
	}
	FloatNode = &NodeType{
		14, []string{"float", "f"}, 4, 1, reflect.TypeOf(float32(0)), floatBytesToValue, floatValueToBytes, floatStringToValue[float32],
	}
	DoubleNode = &NodeType{
		15, []string{"double", "d"}, 8, 1, reflect.TypeOf(float64(0)), doubleBytesToValue, doubleValueToBytes, floatStringToValue[float64],
	}

	Vec2S8Node = &NodeType{
		16, []string{"2s8"}, 2, 2, reflect.TypeOf([2]int8{}), vectorBytesToValue[[2]any](1, int8BytesToValue), vectorValueToBytes(1, int8ValueToBytes), vectorStringToValue[[2]any](intStringToValue[int8]),
	}
	Vec2U8Node = &NodeType{
		17, []string{"2u8"}, 2, 2, reflect.TypeOf([2]uint8{}), vectorBytesToValue[[2]any](1, uint8BytesToValue), vectorValueToBytes(1, uint8ValueToBytes), vectorStringToValue[[2]any](uintStringToValue[uint8]),
	}
	Vec2S16Node = &NodeType{
		18, []string{"2s16"}, 4, 2, reflect.TypeOf([2]int16{}), vectorBytesToValue[[2]any](2, int16BytesToValue), vectorValueToBytes(2, int16ValueToBytes), vectorStringToValue[[2]any](intStringToValue[int16]),
	}
	Vec2U16Node = &NodeType{
		19, []string{"2u16"}, 4, 2, reflect.TypeOf([2]uint16{}), vectorBytesToValue[[2]any](2, uint16BytesToValue), vectorValueToBytes(2, uint16ValueToBytes), vectorStringToValue[[2]any](uintStringToValue[uint16]),
	}
	Vec2S32Node = &NodeType{
		20, []string{"2s32"}, 8, 2, reflect.TypeOf([2]int32{}), vectorBytesToValue[[2]any](4, int32BytesToValue), vectorValueToBytes(4, int32ValueToBytes), vectorStringToValue[[2]any](intStringToValue[int32]),
	}
	Vec2U32Node = &NodeType{
		21, []string{"2u32"}, 8, 2, reflect.TypeOf([2]uint32{}), vectorBytesToValue[[2]any](4, uint32BytesToValue), vectorValueToBytes(4, uint32ValueToBytes), vectorStringToValue[[2]any](uintStringToValue[uint32]),
	}
	Vec2S64Node = &NodeType{
		22, []string{"vs64", "2s64"}, 16, 2, reflect.TypeOf([2]int64{}), vectorBytesToValue[[2]any](8, int64BytesToValue), vectorValueToBytes(8, int64ValueToBytes), vectorStringToValue[[2]any](intStringToValue[int64]),
	}
	Vec2U64Node = &NodeType{
		23, []string{"vu64", "2u64"}, 16, 2, reflect.TypeOf([2]uint64{}), vectorBytesToValue[[2]any](8, uint64BytesToValue), vectorValueToBytes(8, uint64ValueToBytes), vectorStringToValue[[2]any](uintStringToValue[uint64]),
	}
	Vec2FloatNode = &NodeType{
		24, []string{"2f"}, 8, 2, reflect.TypeOf([2]float32{}), vectorBytesToValue[[2]any](4, floatBytesToValue), vectorValueToBytes(4, floatValueToBytes), vectorStringToValue[[2]any](floatStringToValue[float32]),
	}
	Vec2DoubleNode = &NodeType{
		25, []string{"vd", "2d"}, 16, 2, reflect.TypeOf([2]float64{}), vectorBytesToValue[[2]any](8, doubleBytesToValue), vectorValueToBytes(8, doubleValueToBytes), vectorStringToValue[[2]any](floatStringToValue[float64]),
	}

	Vec3S8Node = &NodeType{
		26, []string{"3s8"}, 3, 3, reflect.TypeOf([3]int8{}), vectorBytesToValue[[3]any](1, int8BytesToValue), vectorValueToBytes(1, int8ValueToBytes), vectorStringToValue[[3]any](intStringToValue[int8]),
	}
	Vec3U8Node = &NodeType{
		27, []string{"3u8"}, 3, 3, reflect.TypeOf([3]uint8{}), vectorBytesToValue[[3]any](1, uint8BytesToValue), vectorValueToBytes(1, uint8ValueToBytes), vectorStringToValue[[3]any](uintStringToValue[uint8]),
	}
	Vec3S16Node = &NodeType{
		28, []string{"3s16"}, 6, 3, reflect.TypeOf([3]int16{}), vectorBytesToValue[[3]any](2, int16BytesToValue), vectorValueToBytes(2, int16ValueToBytes), vectorStringToValue[[3]any](intStringToValue[int16]),
	}
	Vec3U16Node = &NodeType{
		29, []string{"3u16"}, 6, 3, reflect.TypeOf([3]uint16{}), vectorBytesToValue[[3]any](2, uint16BytesToValue), vectorValueToBytes(2, uint16ValueToBytes), vectorStringToValue[[3]any](uintStringToValue[uint16]),
	}
	Vec3S32Node = &NodeType{
		30, []string{"3s32"}, 12, 3, reflect.TypeOf([3]int32{}), vectorBytesToValue[[3]any](4, int32BytesToValue), vectorValueToBytes(4, int32ValueToBytes), vectorStringToValue[[3]any](intStringToValue[int32]),
	}
	Vec3U32Node = &NodeType{
		31, []string{"3u32"}, 12, 3, reflect.TypeOf([3]uint32{}), vectorBytesToValue[[3]any](4, uint32BytesToValue), vectorValueToBytes(4, uint32ValueToBytes), vectorStringToValue[[3]any](uintStringToValue[uint32]),
	}
	Vec3S64Node = &NodeType{
		32, []string{"3s64"}, 24, 3, reflect.TypeOf([3]int64{}), vectorBytesToValue[[3]any](8, int64BytesToValue), vectorValueToBytes(8, int64ValueToBytes), vectorStringToValue[[3]any](intStringToValue[int64]),
	}
	Vec3U64Node = &NodeType{
		33, []string{"3u64"}, 24, 3, reflect.TypeOf([3]uint64{}), vectorBytesToValue[[3]any](8, uint64BytesToValue), vectorValueToBytes(8, uint64ValueToBytes), vectorStringToValue[[3]any](uintStringToValue[uint64]),
	}
	Vec3FloatNode = &NodeType{
		34, []string{"3f"}, 12, 3, reflect.TypeOf([3]float32{}), vectorBytesToValue[[3]any](4, floatBytesToValue), vectorValueToBytes(4, floatValueToBytes), vectorStringToValue[[3]any](floatStringToValue[float32]),
	}
	Vec3DoubleNode = &NodeType{
		35, []string{"3d"}, 24, 3, reflect.TypeOf([3]float64{}), vectorBytesToValue[[3]any](8, doubleBytesToValue), vectorValueToBytes(8, doubleValueToBytes), vectorStringToValue[[3]any](floatStringToValue[float64]),
	}

	Vec4S8Node = &NodeType{
		36, []string{"4s8"}, 4, 4, reflect.TypeOf([4]int8{}), vectorBytesToValue[[4]any](1, int8BytesToValue), vectorValueToBytes(1, int8ValueToBytes), vectorStringToValue[[4]any](intStringToValue[int8]),
	}
	Vec4U8Node = &NodeType{
		37, []string{"4u8"}, 4, 4, reflect.TypeOf([4]uint8{}), vectorBytesToValue[[4]any](1, uint8BytesToValue), vectorValueToBytes(1, uint8ValueToBytes), vectorStringToValue[[4]any](uintStringToValue[uint8]),
	}
	Vec4S16Node = &NodeType{
		38, []string{"4s16"}, 8, 4, reflect.TypeOf([4]int16{}), vectorBytesToValue[[4]any](2, int16BytesToValue), vectorValueToBytes(2, int16ValueToBytes), vectorStringToValue[[4]any](intStringToValue[int16]),
	}
	Vec4U16Node = &NodeType{
		39, []string{"4u16"}, 8, 4, reflect.TypeOf([4]uint16{}), vectorBytesToValue[[4]any](2, uint16BytesToValue), vectorValueToBytes(2, uint16ValueToBytes), vectorStringToValue[[4]any](uintStringToValue[uint16]),
	}
	Vec4S32Node = &NodeType{
		40, []string{"vs32", "4s32"}, 16, 4, reflect.TypeOf([4]int32{}), vectorBytesToValue[[4]any](4, int32BytesToValue), vectorValueToBytes(4, int32ValueToBytes), vectorStringToValue[[4]any](intStringToValue[int32]),
	}
	Vec4U32Node = &NodeType{
		41, []string{"vu32", "4u32"}, 16, 4, reflect.TypeOf([4]uint32{}), vectorBytesToValue[[4]any](4, uint32BytesToValue), vectorValueToBytes(4, uint32ValueToBytes), vectorStringToValue[[4]any](uintStringToValue[uint32]),
	}
	Vec4S64Node = &NodeType{
		42, []string{"4s64"}, 32, 4, reflect.TypeOf([4]int64{}), vectorBytesToValue[[4]any](8, int64BytesToValue), vectorValueToBytes(8, int64ValueToBytes), vectorStringToValue[[4]any](intStringToValue[int64]),
	}
	Vec4U64Node = &NodeType{
		43, []string{"4u64"}, 32, 4, reflect.TypeOf([4]uint64{}), vectorBytesToValue[[4]any](8, uint64BytesToValue), vectorValueToBytes(8, uint64ValueToBytes), vectorStringToValue[[4]any](uintStringToValue[uint64]),
	}
	Vec4FloatNode = &NodeType{
		44, []string{"vf", "4f"}, 16, 4, reflect.TypeOf([4]float32{}), vectorBytesToValue[[4]any](4, floatBytesToValue), vectorValueToBytes(4, floatValueToBytes), vectorStringToValue[[4]any](floatStringToValue[float32]),
	}
	Vec4DoubleNode = &NodeType{
		45, []string{"4d"}, 32, 4, reflect.TypeOf([4]float64{}), vectorBytesToValue[[4]any](8, doubleBytesToValue), vectorValueToBytes(8, doubleValueToBytes), vectorStringToValue[[4]any](floatStringToValue[float64]),
	}

	Vec16S8Node = &NodeType{
		48, []string{"vs8", "16s8"}, 16, 16, reflect.TypeOf([16]int8{}), vectorBytesToValue[[16]any](1, int8BytesToValue), vectorValueToBytes(1, int8ValueToBytes), vectorStringToValue[[16]any](intStringToValue[int8]),
	}
	Vec16U8Node = &NodeType{
		49, []string{"vu8", "16s8"}, 16, 16, reflect.TypeOf([16]uint8{}), vectorBytesToValue[[16]any](1, uint8BytesToValue), vectorValueToBytes(1, uint8ValueToBytes), vectorStringToValue[[16]any](uintStringToValue[uint8]),
	}
	Vec8S16Node = &NodeType{
		50, []string{"vs16", "8s16"}, 16, 8, reflect.TypeOf([8]int16{}), vectorBytesToValue[[8]any](2, int16BytesToValue), vectorValueToBytes(2, int16ValueToBytes), vectorStringToValue[[8]any](intStringToValue[int16]),
	}
	Vec8U16Node = &NodeType{
		51, []string{"vu16", "8u16"}, 16, 8, reflect.TypeOf([8]uint16{}), vectorBytesToValue[[8]any](2, uint16BytesToValue), vectorValueToBytes(2, uint16ValueToBytes), vectorStringToValue[[8]any](uintStringToValue[uint16]),
	}

	BoolNode = &NodeType{
		52, []string{"bool", "b"}, 1, 1, reflect.TypeOf(BoolValue(false)), boolBytesToValue, boolValueToBytes, boolStringToValue,
	}
	Vec2BoolNode = &NodeType{
		53, []string{"2b"}, 2, 2, reflect.TypeOf([2]BoolValue{}), vectorBytesToValue[[2]any](1, boolBytesToValue), vectorValueToBytes(1, boolValueToBytes), vectorStringToValue[[2]any](boolStringToValue),
	}
	Vec3BoolNode = &NodeType{
		54, []string{"3b"}, 3, 3, reflect.TypeOf([3]BoolValue{}), vectorBytesToValue[[3]any](1, boolBytesToValue), vectorValueToBytes(1, boolValueToBytes), vectorStringToValue[[3]any](boolStringToValue),
	}
	Vec4BoolNode = &NodeType{
		55, []string{"4b"}, 4, 4, reflect.TypeOf([4]BoolValue{}), vectorBytesToValue[[4]any](1, boolBytesToValue), vectorValueToBytes(1, boolValueToBytes), vectorStringToValue[[4]any](boolStringToValue),
	}
	Vec16BoolNode = &NodeType{
		56, []string{"vb", "16b"}, 16, 16, reflect.TypeOf([16]BoolValue{}), vectorBytesToValue[[16]any](1, boolBytesToValue), vectorValueToBytes(1, boolValueToBytes), vectorStringToValue[[16]any](boolStringToValue),
	}

	idLut = []*NodeType{
		1: VoidNode,
		S8Node,
		U8Node,
		S16Node,
		U16Node,
		S32Node,
		U32Node,
		S64Node,
		U64Node,
		BinNode,
		StrNode,
		IPv4Node,
		TimeNode,
		FloatNode,
		DoubleNode,

		Vec2S8Node,
		Vec2U8Node,
		Vec2S16Node,
		Vec2U16Node,
		Vec2S32Node,
		Vec2U32Node,
		Vec2S64Node,
		Vec2U64Node,
		Vec2FloatNode,
		Vec2DoubleNode,

		Vec3S8Node,
		Vec3U8Node,
		Vec3S16Node,
		Vec3U16Node,
		Vec3S32Node,
		Vec3U32Node,
		Vec3S64Node,
		Vec3U64Node,
		Vec3FloatNode,
		Vec3DoubleNode,

		Vec4S8Node,
		Vec4U8Node,
		Vec4S16Node,
		Vec4U16Node,
		Vec4S32Node,
		Vec4U32Node,
		Vec4S64Node,
		Vec4U64Node,
		Vec4FloatNode,
		Vec4DoubleNode,

		nil,
		nil,

		Vec16S8Node,
		Vec16U8Node,
		Vec8S16Node,
		Vec8U16Node,
		BoolNode,
		Vec2BoolNode,
		Vec3BoolNode,
		Vec4BoolNode,
		Vec16BoolNode,
	}
	typeLut = map[reflect.Type]*NodeType{}
	nameLut = map[string]*NodeType{}
)

func init() {
	for _, t := range idLut {
		if t != nil {
			typeLut[t.rt] = t

			for _, name := range t.names {
				nameLut[name] = t
			}
		}
	}
}

func lookupTypeByName(name string) *NodeType {
	return nameLut[name]
}

func lookupTypeById(id byte) *NodeType {
	if int(id) >= len(idLut) {
		return nil
	}
	return idLut[id]
}

func int8BytesToValue(b []byte) (any, error) {
	return int8(b[0]), nil
}

func uint8BytesToValue(b []byte) (any, error) {
	return b[0], nil
}

func int16BytesToValue(b []byte) (any, error) {
	return int16(binary.BigEndian.Uint16(b)), nil
}

func uint16BytesToValue(b []byte) (any, error) {
	return binary.BigEndian.Uint16(b), nil
}

func int32BytesToValue(b []byte) (any, error) {
	return int32(binary.BigEndian.Uint32(b)), nil
}

func uint32BytesToValue(b []byte) (any, error) {
	return binary.BigEndian.Uint32(b), nil
}

func timeBytesToValue(b []byte) (any, error) {
	return TimeValue(binary.BigEndian.Uint32(b)), nil
}

func int64BytesToValue(b []byte) (any, error) {
	return int64(binary.BigEndian.Uint64(b)), nil
}

func uint64BytesToValue(b []byte) (any, error) {
	return binary.BigEndian.Uint64(b), nil
}

func ip4BytesToValue(b []byte) (any, error) {
	return net.IPv4(b[0], b[1], b[2], b[3]), nil
}

func floatBytesToValue(b []byte) (any, error) {
	return math.Float32frombits(binary.BigEndian.Uint32(b)), nil
}

func doubleBytesToValue(b []byte) (any, error) {
	return math.Float64frombits(binary.BigEndian.Uint64(b)), nil
}

func boolBytesToValue(b []byte) (any, error) {
	switch b[0] {
	case 0:
		return BoolValue(false), nil
	case 1:
		return BoolValue(true), nil
	default:
		return nil, propertyError("invalid bool byte")
	}
}

func vectorBytesToValue[T [2]any | [3]any | [4]any | [8]any | [16]any](size int, f bytesToValue) bytesToValue {
	return func(b []byte) (any, error) {
		var o T
		for i := 0; i < len(o); i++ {
			v, err := f(b[i*size:])
			if err != nil {
				return nil, err
			}
			o[i] = v
		}
		return o, nil
	}
}

func int8ValueToBytes(v any, b []byte) {
	b[0] = uint8(v.(int8))
}

func uint8ValueToBytes(v any, b []byte) {
	b[0] = v.(uint8)
}

func int16ValueToBytes(v any, b []byte) {
	binary.BigEndian.PutUint16(b, uint16(v.(int16)))
}

func uint16ValueToBytes(v any, b []byte) {
	binary.BigEndian.PutUint16(b, v.(uint16))
}

func int32ValueToBytes(v any, b []byte) {
	binary.BigEndian.PutUint32(b, uint32(v.(int32)))
}

func uint32ValueToBytes(v any, b []byte) {
	binary.BigEndian.PutUint32(b, v.(uint32))
}

func timeValueToBytes(v any, b []byte) {
	binary.BigEndian.PutUint32(b, uint32(v.(TimeValue)))
}

func int64ValueToBytes(v any, b []byte) {
	binary.BigEndian.PutUint64(b, uint64(v.(int64)))
}

func uint64ValueToBytes(v any, b []byte) {
	binary.BigEndian.PutUint64(b, v.(uint64))
}

func ip4ValueToBytes(v any, b []byte) {
	copy(b, v.(net.IP).To4())
}

func floatValueToBytes(v any, b []byte) {
	uint32ValueToBytes(math.Float32bits(v.(float32)), b)
}

func doubleValueToBytes(v any, b []byte) {
	uint64ValueToBytes(math.Float64bits(v.(float64)), b)
}

func boolValueToBytes(v any, b []byte) {
	if v.(BoolValue) {
		b[0] = 1
	} else {
		b[0] = 0
	}
}

func vectorValueToBytes(size int, f valueToBytes) valueToBytes {
	return func(v any, b []byte) {
		vo := reflect.ValueOf(v)
		for i := 0; i < vo.Len(); i++ {
			f(vo.Index(i).Interface(), b[i*size:])
		}
	}
}

func intStringToValue[T int8 | int16 | int32 | int64](s string) (any, error) {
	i, err := strconv.ParseInt(s, 10, int(unsafe.Sizeof(T(0))*8))
	return T(i), err
}

func uintStringToValue[T uint8 | uint16 | uint32 | uint64 | TimeValue](s string) (any, error) {
	i, err := strconv.ParseUint(s, 10, int(unsafe.Sizeof(T(0))*8))
	return T(i), err
}

func ip4StringToValue(s string) (any, error) {
	if ip := net.ParseIP(s).To4(); ip != nil {
		return ip, nil
	}
	return nil, propertyError("invalid ip address")
}

func floatStringToValue[T float32 | float64](s string) (any, error) {
	f, err := strconv.ParseFloat(s, int(unsafe.Sizeof(T(0))))
	return T(f), err
}

func boolStringToValue(s string) (any, error) {
	switch s {
	case "1":
		return BoolValue(true), nil
	case "0":
		return BoolValue(false), nil
	default:
		return nil, propertyError("invalid bool string")
	}
}

func vectorStringToValue[T [2]any | [3]any | [4]any | [8]any | [16]any](f stringToValue) stringToValue {
	return func(s string) (any, error) {
		var o T

		spl := strings.Split(s, " ")
		if len(spl) != len(o) {
			return nil, propertyError("vector string contains an invalid number of elements")
		}
		for i, s := range spl {
			v, err := f(s)
			if err != nil {
				return nil, err
			}
			o[i] = v
		}
		return o, nil
	}
}
