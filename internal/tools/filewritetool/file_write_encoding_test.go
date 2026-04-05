package filewritetool

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/2456868764/rabbit-code/internal/tools/filereadtool"
)

func TestFileWrite_utf16lePreservesEncoding(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "u16.txt")
	plain := "alpha\nbeta"
	disk, err := encodeUTF16LEToBytes(plain)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, disk, 0o644); err != nil {
		t.Fatal(err)
	}
	abs, _ := filepath.Abs(p)
	fi, err := os.Stat(abs)
	if err != nil {
		t.Fatal(err)
	}
	st := filereadtool.NewReadFileStateMap()
	st.Set(abs, filereadtool.ReadFileStateEntry{
		Content:       plain,
		Timestamp:     fi.ModTime().UnixMilli(),
		IsPartialView: false,
	})
	ctx := filereadtool.WithRunContext(context.Background(), &filereadtool.RunContext{ReadFileState: st})
	fw := New()
	in, _ := json.Marshal(map[string]string{"file_path": abs, "content": "alpha\ngamma"})
	out, err := fw.Run(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil || m["type"] != "update" {
		t.Fatalf("%v %s", err, out)
	}
	round, err := os.ReadFile(abs)
	if err != nil {
		t.Fatal(err)
	}
	text, err := decodeFileBytesToUTF8(round, EncUTF16LE)
	if err != nil {
		t.Fatal(err)
	}
	if text != "alpha\ngamma" {
		t.Fatalf("got %q", text)
	}
}

func TestFileWrite_crlfUtf8LineEndings(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "crlf.txt")
	if err := os.WriteFile(p, []byte("x\r\ny\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	abs, _ := filepath.Abs(p)
	fi, err := os.Stat(abs)
	if err != nil {
		t.Fatal(err)
	}
	st := filereadtool.NewReadFileStateMap()
	st.Set(abs, filereadtool.ReadFileStateEntry{
		Content:       "x\ny",
		Timestamp:     fi.ModTime().UnixMilli(),
		IsPartialView: false,
	})
	ctx := filereadtool.WithRunContext(context.Background(), &filereadtool.RunContext{ReadFileState: st})
	fw := New()
	in, _ := json.Marshal(map[string]string{"file_path": abs, "content": "x\nz"})
	_, err = fw.Run(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(abs)
	if err != nil {
		t.Fatal(err)
	}
	// Mirrors TS writeTextContent: split('\n').join('\r\n') — no trailing CRLF after final line.
	if string(got) != "x\r\nz" {
		t.Fatalf("got %q want CRLF between lines", got)
	}
}

func TestFileWrite_fileEncodingMetadataOverride(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "meta.txt")
	if err := os.WriteFile(p, []byte{0xff, 0xfe}, 0o644); err != nil {
		t.Fatal(err)
	}
	abs, _ := filepath.Abs(p)
	fi, err := os.Stat(abs)
	if err != nil {
		t.Fatal(err)
	}
	st := filereadtool.NewReadFileStateMap()
	st.Set(abs, filereadtool.ReadFileStateEntry{
		Content:       "",
		Timestamp:     fi.ModTime().UnixMilli(),
		IsPartialView: false,
	})
	ctx := WithWriteContext(
		filereadtool.WithRunContext(context.Background(), &filereadtool.RunContext{ReadFileState: st}),
		&WriteContext{
			FileEncodingMetadata: func(string) (string, LineEndingType, bool) {
				return EncUTF16LE, LineEndingLF, true
			},
		},
	)
	fw := New()
	in, _ := json.Marshal(map[string]string{"file_path": abs, "content": "ok"})
	_, err = fw.Run(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		t.Fatal(err)
	}
	txt, err := decodeFileBytesToUTF8(b, EncUTF16LE)
	if err != nil || txt != "ok" {
		t.Fatalf("%v %q", err, txt)
	}
}
