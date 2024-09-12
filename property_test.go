package avsproperty

import (
	"bytes"
	"io"
	"net"
	"os"
	"reflect"
	"testing"
)

var (
	testcaseBinary     []byte
	testcaseBinaryLong []byte
	testcaseXML        []byte
	testcaseNode       *Node
)

func init() {
	var err error
	if testcaseBinary, err = os.ReadFile("testcases/test.bin"); err != nil {
		panic(err)
	}
	if testcaseBinaryLong, err = os.ReadFile("testcases/test_long.bin"); err != nil {
		panic(err)
	}
	if testcaseXML, err = os.ReadFile("testcases/test.xml"); err != nil {
		panic(err)
	}

	prop := Property{}
	if err := prop.Read(bytes.NewReader(testcaseBinary)); err != nil {
		panic(err)
	}
	testcaseNode = prop.Root
}

func TestRoundtrip(t *testing.T) {
	testcases := [][]byte{
		testcaseBinary,
		testcaseBinaryLong,
	}

	prop := &Property{}
	format := FormatBinary

	for i, testcase := range testcases {
		data := testcase
		for i := 0; i < 2; i++ {
			prop.Root = nil
			if err := prop.Read(bytes.NewReader(data)); err != nil {
				t.Fatalf("%d: read: %v", i, err)
			}
			if prop.Settings.Format != format {
				t.Fatal("incorrect format")
			}

			format = (format + 1) % 2
			prop.Settings.Format = format

			wr := &bytes.Buffer{}
			if err := prop.Write(wr); err != nil {
				t.Fatalf("%d: write: %v", i, err)
			}
			data = wr.Bytes()
		}

		if !bytes.Equal(data, testcase) {
			os.Stdout.Write(data)
			t.Fatalf("%d: roundtrip failed", i)
		}
	}
}

func TestTypes(t *testing.T) {
	for _, nt := range idLut {
		if nt == nil || nt == VoidNode || nt == IPv4Node {
			continue
		}

		// create with regular value
		v := reflect.New(nt.rt).Elem()
		node, err := NewNodeWithValue("foo", v.Interface())
		if err != nil {
			t.Fatal(err)
		}
		if reflect.TypeOf(node.value) != nt.rt {
			t.Fatal("type does not match")
		}

		// create with array value
		v = reflect.MakeSlice(reflect.SliceOf(nt.rt), 2, 2)
		node, err = NewNodeWithValue("bar", v.Interface())
		if nt != StrNode && nt != BinNode {
			if err != nil {
				t.Fatal(err)
			}
			if reflect.TypeOf(node.value).Elem() != nt.rt {
				t.Fatal("array type does not match")
			}
		} else if err == nil {
			// creating an array of strings or binary values should fail
			t.Fatal("invalid array")
		}
	}

	// make sure that only IPv4 addresses are accepted
	if _, err := NewNodeWithValue("test", net.IPv4(133, 221, 3, 27)); err != nil {
		t.Fatal(err)
	}
	if _, err := NewNodeWithValue("test", net.IP{0, 10, 2, 80, 116}); err == nil {
		t.Fatal("invalid ip")
	}
}

func BenchmarkReadBinary(b *testing.B) {
	prop := Property{}
	rd := bytes.NewReader(testcaseXML)
	for i := 0; i < b.N; i++ {
		if err := prop.Read(rd); err != nil {
			b.Fatal(err)
		}
		prop.Root = nil
		rd.Reset(testcaseBinary)
	}
}

func BenchmarkReadXML(b *testing.B) {
	prop := Property{}
	rd := bytes.NewReader(testcaseXML)
	for i := 0; i < b.N; i++ {
		if err := prop.Read(rd); err != nil {
			b.Fatal(err)
		}
		prop.Root = nil
		rd.Reset(testcaseXML)
	}
}

func BenchmarkWriteBinary(b *testing.B) {
	prop := Property{
		Settings: PropertySettings{Format: FormatBinary},
		Root:     testcaseNode,
	}
	for i := 0; i < b.N; i++ {
		if err := prop.Write(io.Discard); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWriteXML(b *testing.B) {
	prop := Property{
		Settings: PropertySettings{Format: FormatXML},
		Root:     testcaseNode,
	}
	for i := 0; i < b.N; i++ {
		if err := prop.Write(io.Discard); err != nil {
			b.Fatal(err)
		}
	}
}
