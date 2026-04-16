package bashtool

import (
	"reflect"
	"sort"
	"testing"
)

// PATH_EXTRACTORS tests

func TestPathExtractors_cd(t *testing.T) {
	ex := PATH_EXTRACTORS["cd"]
	if got := ex([]string{"mydir"}); !reflect.DeepEqual(got, []string{"mydir"}) {
		t.Fatalf("cd single arg: %v", got)
	}
	if got := ex([]string{"my", "dir"}); !reflect.DeepEqual(got, []string{"my dir"}) {
		t.Fatalf("cd multi arg: %v", got)
	}
}

func TestPathExtractors_ls(t *testing.T) {
	ex := PATH_EXTRACTORS["ls"]
	if got := ex([]string{"-la", "foo", "bar"}); !reflect.DeepEqual(got, []string{"foo", "bar"}) {
		t.Fatalf("ls with flags: %v", got)
	}
	if got := ex(nil); !reflect.DeepEqual(got, []string{"."}) {
		t.Fatalf("ls empty: %v", got)
	}
}

func TestPathExtractors_find(t *testing.T) {
	ex := PATH_EXTRACTORS["find"]
	if got := ex([]string{".", "-name", "*.go"}); !reflect.DeepEqual(got, []string{"."}) {
		t.Fatalf("find basic: %v", got)
	}
	// After -- everything is a path
	got := ex([]string{"--", "-weird-path"})
	if len(got) != 1 || got[0] != "-weird-path" {
		t.Fatalf("find after --: %v", got)
	}
	// -newer takes a path arg
	got = ex([]string{".", "-newer", "ref.txt"})
	if len(got) < 2 {
		t.Fatalf("find -newer: %v", got)
	}
}

func TestPathExtractors_grep(t *testing.T) {
	ex := PATH_EXTRACTORS["grep"]
	// pattern then files
	if got := ex([]string{"foo", "file1.txt", "file2.txt"}); !reflect.DeepEqual(got, []string{"file1.txt", "file2.txt"}) {
		t.Fatalf("grep pattern+files: %v", got)
	}
	// recursive → default to "."
	if got := ex([]string{"-r", "foo"}); !reflect.DeepEqual(got, []string{"."}) {
		t.Fatalf("grep -r no path: %v", got)
	}
}

func TestPathExtractors_rg(t *testing.T) {
	ex := PATH_EXTRACTORS["rg"]
	// no paths → default "."
	if got := ex([]string{"pattern"}); !reflect.DeepEqual(got, []string{"."}) {
		t.Fatalf("rg no path: %v", got)
	}
	// with file
	if got := ex([]string{"pattern", "src/"}); !reflect.DeepEqual(got, []string{"src/"}) {
		t.Fatalf("rg with path: %v", got)
	}
}

func TestPathExtractors_sed(t *testing.T) {
	ex := PATH_EXTRACTORS["sed"]
	// no -e: first non-flag is script, rest are files
	got := ex([]string{"-n", "1p", "file.txt"})
	if !reflect.DeepEqual(got, []string{"file.txt"}) {
		t.Fatalf("sed basic: %v", got)
	}
	// with -e: all non-flag non-expression args are files
	got = ex([]string{"-e", "s/a/b/", "file.txt"})
	if !reflect.DeepEqual(got, []string{"file.txt"}) {
		t.Fatalf("sed with -e: %v", got)
	}
	// -f script file
	got = ex([]string{"-f", "script.sed", "data.txt"})
	if !reflect.DeepEqual(got, []string{"script.sed", "data.txt"}) {
		t.Fatalf("sed with -f: %v", got)
	}
}

func TestPathExtractors_jq(t *testing.T) {
	ex := PATH_EXTRACTORS["jq"]
	// filter then files
	got := ex([]string{".", "file1.json", "file2.json"})
	if !reflect.DeepEqual(got, []string{"file1.json", "file2.json"}) {
		t.Fatalf("jq filter+files: %v", got)
	}
	// no files → stdin → empty
	got = ex([]string{"."})
	if len(got) != 0 {
		t.Fatalf("jq no file: %v", got)
	}
}

func TestPathExtractors_git(t *testing.T) {
	ex := PATH_EXTRACTORS["git"]
	// git diff --no-index extracts 2 paths
	got := ex([]string{"diff", "--no-index", "a.txt", "b.txt"})
	if !reflect.DeepEqual(got, []string{"a.txt", "b.txt"}) {
		t.Fatalf("git diff --no-index: %v", got)
	}
	// other git subcommands → no paths
	if got := ex([]string{"status"}); len(got) != 0 {
		t.Fatalf("git status: %v", got)
	}
}

func TestPathExtractors_tr(t *testing.T) {
	ex := PATH_EXTRACTORS["tr"]
	// tr SET1 SET2 FILE
	got := ex([]string{"a-z", "A-Z", "file.txt"})
	if !reflect.DeepEqual(got, []string{"file.txt"}) {
		t.Fatalf("tr SET1 SET2 FILE: %v", got)
	}
	// tr -d SET FILE
	got = ex([]string{"-d", "a-z", "file.txt"})
	if !reflect.DeepEqual(got, []string{"file.txt"}) {
		t.Fatalf("tr -d SET FILE: %v", got)
	}
}

func TestPathExtractors_simpleCommands(t *testing.T) {
	for _, cmd := range []string{"mkdir", "touch", "rm", "cat", "head", "stat", "diff"} {
		ex := PATH_EXTRACTORS[cmd]
		if ex == nil {
			t.Fatalf("%s: extractor missing", cmd)
		}
		got := ex([]string{"-r", "foo", "bar"})
		if !reflect.DeepEqual(got, []string{"foo", "bar"}) {
			t.Fatalf("%s filterOutFlags: %v", cmd, got)
		}
	}
}

func TestCommandOperationType(t *testing.T) {
	if COMMAND_OPERATION_TYPE["rm"] != FileOperationWrite {
		t.Fatal("rm should be write")
	}
	if COMMAND_OPERATION_TYPE["cat"] != FileOperationRead {
		t.Fatal("cat should be read")
	}
	if COMMAND_OPERATION_TYPE["mkdir"] != FileOperationCreate {
		t.Fatal("mkdir should be create")
	}
	if COMMAND_OPERATION_TYPE["sed"] != FileOperationWrite {
		t.Fatal("sed should be write")
	}
}

// StripWrappersFromArgv tests

func TestStripWrappersFromArgv_noWrapper(t *testing.T) {
	in := []string{"ls", "-la"}
	got := StripWrappersFromArgv(in)
	if !reflect.DeepEqual(got, in) {
		t.Fatalf("no wrapper: %v", got)
	}
}

func TestStripWrappersFromArgv_timeout(t *testing.T) {
	got := StripWrappersFromArgv([]string{"timeout", "10", "ls", "-la"})
	if !reflect.DeepEqual(got, []string{"ls", "-la"}) {
		t.Fatalf("timeout: %v", got)
	}
}

func TestStripWrappersFromArgv_timeoutWithFlags(t *testing.T) {
	got := StripWrappersFromArgv([]string{"timeout", "--kill-after", "5s", "10", "rm", "-rf", "foo"})
	if !reflect.DeepEqual(got, []string{"rm", "-rf", "foo"}) {
		t.Fatalf("timeout with flags: %v", got)
	}
}

func TestStripWrappersFromArgv_nice(t *testing.T) {
	got := StripWrappersFromArgv([]string{"nice", "-n", "10", "ls"})
	if !reflect.DeepEqual(got, []string{"ls"}) {
		t.Fatalf("nice -n N: %v", got)
	}
	got = StripWrappersFromArgv([]string{"nice", "ls"})
	if !reflect.DeepEqual(got, []string{"ls"}) {
		t.Fatalf("nice bare: %v", got)
	}
}

func TestStripWrappersFromArgv_nohup(t *testing.T) {
	got := StripWrappersFromArgv([]string{"nohup", "ls", "-la"})
	if !reflect.DeepEqual(got, []string{"ls", "-la"}) {
		t.Fatalf("nohup: %v", got)
	}
}

func TestStripWrappersFromArgv_stdbuf(t *testing.T) {
	got := StripWrappersFromArgv([]string{"stdbuf", "-o0", "ls"})
	if !reflect.DeepEqual(got, []string{"ls"}) {
		t.Fatalf("stdbuf: %v", got)
	}
}

func TestStripWrappersFromArgv_env(t *testing.T) {
	got := StripWrappersFromArgv([]string{"env", "FOO=bar", "ls", "-la"})
	if !reflect.DeepEqual(got, []string{"ls", "-la"}) {
		t.Fatalf("env: %v", got)
	}
}

func TestStripWrappersFromArgv_nested(t *testing.T) {
	got := StripWrappersFromArgv([]string{"timeout", "10", "nice", "ls"})
	if !reflect.DeepEqual(got, []string{"ls"}) {
		t.Fatalf("nested timeout+nice: %v", got)
	}
}

// CheckPathConstraints extended coverage via PATH_EXTRACTORS

func TestCheckPathConstraints_workdirRoot_catOutside(t *testing.T) {
	err := CheckPathConstraints("cat /etc/passwd", PathValidationOptions{
		Cwd:         "/home/user/project",
		WorkdirRoot: "/home/user/project",
	})
	if err == nil {
		t.Fatal("expected error for cat /etc/passwd outside workdir")
	}
}

func TestCheckPathConstraints_workdirRoot_catInside(t *testing.T) {
	err := CheckPathConstraints("cat README.md", PathValidationOptions{
		Cwd:         "/home/user/project",
		WorkdirRoot: "/home/user/project",
	})
	if err != nil {
		t.Fatalf("unexpected error for cat inside workdir: %v", err)
	}
}

func TestCheckPathConstraints_workdirRoot_grepOutside(t *testing.T) {
	err := CheckPathConstraints("grep foo /etc/passwd", PathValidationOptions{
		Cwd:         "/home/user/project",
		WorkdirRoot: "/home/user/project",
	})
	if err == nil {
		t.Fatal("expected error for grep /etc/passwd outside workdir")
	}
}

// helper for sorted slices comparison
func sortedStringSlice(s []string) []string {
	cp := append([]string(nil), s...)
	sort.Strings(cp)
	return cp
}
