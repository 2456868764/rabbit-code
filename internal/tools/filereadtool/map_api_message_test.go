package filereadtool_test

import (
	"bytes"
	"encoding/json"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/tools/filereadtool"
)

var mapOpt = filereadtool.MapReadResultOptions{}

func TestMapReadResultForMessagesAPI_pdf(t *testing.T) {
	raw := []byte(`{"type":"pdf","file":{"filePath":"/tmp/x.pdf","base64":"SlRBQQ==","originalSize":99}}`)
	c, sup := filereadtool.MapReadResultForMessagesAPI(raw, mapOpt)
	s, ok := c.(string)
	if !ok || !strings.Contains(s, "x.pdf") || !strings.Contains(s, "99") {
		t.Fatalf("summary: %v", c)
	}
	if len(sup) != 1 || len(sup[0]) != 1 {
		t.Fatalf("supplemental: %#v", sup)
	}
	doc, ok := sup[0][0].(map[string]any)
	if !ok || doc["type"] != "document" {
		t.Fatalf("doc block: %#v", sup[0][0])
	}
	src := doc["source"].(map[string]any)
	if src["media_type"] != "application/pdf" || src["data"] != "SlRBQQ==" {
		t.Fatalf("source: %#v", src)
	}
}

func TestMapReadResultForMessagesAPI_image(t *testing.T) {
	raw := []byte(`{"type":"image","file":{"base64":"eA==","type":"image/png","originalSize":1}}`)
	c, sup := filereadtool.MapReadResultForMessagesAPI(raw, mapOpt)
	arr, ok := c.([]any)
	if !ok || len(arr) != 1 || len(sup) != 0 {
		t.Fatalf("c=%T sup=%v", c, sup)
	}
	bl := arr[0].(map[string]any)
	if bl["type"] != "image" {
		t.Fatal(bl)
	}
}

func TestMapReadResultForMessagesAPI_parts(t *testing.T) {
	dir := t.TempDir()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "page-01.jpg"), buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(map[string]any{
		"type": "parts",
		"file": map[string]any{
			"filePath":     "/a.pdf",
			"originalSize": 10,
			"count":        1,
			"outputDir":    dir,
		},
	})
	c, sup := filereadtool.MapReadResultForMessagesAPI(payload, mapOpt)
	if _, ok := c.(string); !ok {
		t.Fatalf("want string summary got %T", c)
	}
	if len(sup) != 1 || len(sup[0]) != 1 {
		t.Fatalf("sup=%#v", sup)
	}
	im := sup[0][0].(map[string]any)
	if im["type"] != "image" {
		t.Fatal(im)
	}
}

func TestMapReadResultForMessagesAPI_textLineNumbersAndCyber(t *testing.T) {
	raw := []byte(`{"type":"text","file":{"content":"a\nb","startLine":1,"totalLines":2,"numLines":2,"filePath":"p"}}`)
	c, sup := filereadtool.MapReadResultForMessagesAPI(raw, filereadtool.MapReadResultOptions{
		MainLoopModel: "claude-opus-4-6",
	})
	s, ok := c.(string)
	if !ok || len(sup) != 0 {
		t.Fatalf("%v %v", c, sup)
	}
	if !strings.Contains(s, "1→a") || strings.Contains(s, "Whenever you read a file") {
		t.Fatalf("exempt model should skip cyber: %q", s)
	}

	c2, _ := filereadtool.MapReadResultForMessagesAPI(raw, filereadtool.MapReadResultOptions{})
	s2, ok := c2.(string)
	if !ok || !strings.Contains(s2, "Whenever you read a file") {
		t.Fatalf("default should append cyber: %q", s2)
	}
}

func TestMapReadResultForMessagesAPI_textOffsetBeyond(t *testing.T) {
	raw := []byte(`{"type":"text","file":{"content":"","startLine":5,"totalLines":1,"numLines":0,"filePath":"p"}}`)
	c, _ := filereadtool.MapReadResultForMessagesAPI(raw, mapOpt)
	s, ok := c.(string)
	if !ok || !strings.Contains(s, "shorter than the provided offset") || !strings.Contains(s, "5") {
		t.Fatalf("%q", s)
	}
}

func TestMapReadResultForMessagesAPI_textEmptyFile(t *testing.T) {
	raw := []byte(`{"type":"text","file":{"content":"","startLine":1,"totalLines":0,"numLines":0,"filePath":"p"}}`)
	c, _ := filereadtool.MapReadResultForMessagesAPI(raw, mapOpt)
	s, ok := c.(string)
	if !ok || !strings.Contains(s, "contents are empty") {
		t.Fatalf("%q", s)
	}
}

func TestMapReadResultForMessagesAPI_fileUnchanged(t *testing.T) {
	raw := []byte(`{"type":"file_unchanged","file":{"filePath":"/x"}}`)
	c, sup := filereadtool.MapReadResultForMessagesAPI(raw, mapOpt)
	s, ok := c.(string)
	if !ok || s != filereadtool.FileUnchangedStub || len(sup) != 0 {
		t.Fatalf("%q sup=%v", c, sup)
	}
}

func TestMapReadResultForMessagesAPI_notebook(t *testing.T) {
	raw := []byte(`{"type":"notebook","file":{"filePath":"n.ipynb","cells":[{"cellType":"markdown","source":"# t","cell_id":"c0"}]}}`)
	c, sup := filereadtool.MapReadResultForMessagesAPI(raw, mapOpt)
	arr, ok := c.([]any)
	if !ok || len(arr) != 1 || len(sup) != 0 {
		t.Fatalf("%T %v", c, sup)
	}
	tb := arr[0].(map[string]any)
	tx := tb["text"].(string)
	if tb["type"] != "text" || !strings.Contains(tx, `<cell id="c0">`) || !strings.Contains(tx, "markdown") {
		t.Fatal(tb)
	}
}

func TestMapReadResultForMessagesAPI_notebookAdjacentTextMerge(t *testing.T) {
	// Same as mapNotebookCellsToToolResult: adjacent text blocks merge (including across cells).
	raw := []byte(`{"type":"notebook","file":{"filePath":"n.ipynb","cells":[{"cellType":"markdown","source":"a","cell_id":"c0"},{"cellType":"markdown","source":"b","cell_id":"c1"}]}}`)
	c, _ := filereadtool.MapReadResultForMessagesAPI(raw, mapOpt)
	arr := c.([]any)
	if len(arr) != 1 {
		t.Fatalf("want 1 merged text block, got %d", len(arr))
	}
	tx := arr[0].(map[string]any)["text"].(string)
	if !strings.Contains(tx, "c0") || !strings.Contains(tx, "c1") {
		t.Fatalf("merged text should include both cells: %q", tx)
	}
}

func TestMapReadResultForMessagesAPI_nonReadShapeFallback(t *testing.T) {
	raw := []byte(`{"type":"unknown","x":1}`)
	c, sup := filereadtool.MapReadResultForMessagesAPI(raw, mapOpt)
	if s, ok := c.(string); !ok || s != string(raw) || sup != nil {
		t.Fatalf("%v %v", c, sup)
	}
}
