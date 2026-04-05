package filewritetool

import (
	"bytes"
	"testing"
)

func TestDetectEncodingFromPrefix(t *testing.T) {
	if g := DetectEncodingFromPrefix(nil); g != encUTF8 {
		t.Fatal(g)
	}
	if g := DetectEncodingFromPrefix([]byte{0xff, 0xfe, 'x'}); g != encUTF16LE {
		t.Fatal(g)
	}
	if g := DetectEncodingFromPrefix([]byte{0xef, 0xbb, 0xbf, 'x'}); g != encUTF8 {
		t.Fatal(g)
	}
}

func TestUTF16LERoundTrip(t *testing.T) {
	const s = "hello\nworld"
	b, err := encodeUTF16LEToBytes(s)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(b, []byte{0xff, 0xfe}) {
		t.Fatal("missing BOM")
	}
	got, err := decodeFileBytesToUTF8(b, encUTF16LE)
	if err != nil || got != s {
		t.Fatalf("%v %q", err, got)
	}
}

func TestDetectLineEndingsForString(t *testing.T) {
	if DetectLineEndingsForString("a\nb\nc", 100) != LineEndingLF {
		t.Fatal()
	}
	if DetectLineEndingsForString("a\r\nb\r\nc", 100) != LineEndingCRLF {
		t.Fatal()
	}
}
