package agent

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

// Folder tool names: they read the folder holding the open file,
// and never write to it.
const (
	ToolListDir  = "list_dir"
	ToolReadFile = "read_file"
	ToolGlob     = "glob"
	ToolGrep     = "grep"
)

// Limits that keep a folder tool's answer within what a model reads.
const (
	maxEntries   = 300
	maxMatches   = 200
	readLines    = 400
	maxLineRunes = 400
	maxFileSize  = 2 << 20
)

// IsFolderTool reports whether name is a tool that reads the folder.
func IsFolderTool(name string) bool {
	switch name {
	case ToolListDir, ToolReadFile, ToolGlob, ToolGrep:
		return true
	}
	return false
}

// FolderTools are the tools that read the folder holding the open file.
func FolderTools() []Tool {
	dir := str("A folder relative to the file's folder; that folder when omitted.")
	const skips = "Skips hidden, dependency and build folders unless path names one."
	return []Tool{
		{ToolListDir, "List a folder of the project: subfolders end with /, files show their size.",
			object(nil, map[string]any{"path": dir})},
		{ToolReadFile, fmt.Sprintf("Read a text file of the project with line numbers, %d lines at a time unless limit says otherwise.", readLines),
			object([]string{"path"}, map[string]any{"path": str("The file, relative to the open file's folder."),
				"offset": map[string]any{"type": "integer", "description": "The first line to read, from 1."},
				"limit":  map[string]any{"type": "integer", "description": "How many lines to read."}})},
		{ToolGlob, "Find files whose paths match a glob such as **/*.go or src/**/route*.ts; ** matches any folders. " + skips,
			object([]string{"pattern"}, map[string]any{"pattern": str("The glob, relative to path."), "path": dir})},
		{ToolGrep, "Search the project's text files for a regular expression (RE2 syntax) and list the matching lines as path:line: text. " + skips,
			object([]string{"pattern"}, map[string]any{"pattern": str("The regular expression."), "path": dir,
				"include": str("A glob such as *.go that a file's name must match.")})},
	}
}

// ReadFolder runs a folder tool on the project in fsys and returns what
// the model reads back, and whether the call failed.
func ReadFolder(fsys fs.FS, name, arguments string) (string, bool) {
	if strings.TrimSpace(arguments) == "" {
		arguments = "{}"
	}
	var a struct {
		Path, Pattern, Include string
		Offset, Limit          int
	}
	if err := json.Unmarshal([]byte(arguments), &a); err != nil {
		return "error: " + err.Error(), true
	}
	p, err := clean(a.Path)
	if err != nil {
		return "error: " + err.Error(), true
	}
	var out string
	switch name {
	case ToolListDir:
		out, err = listDir(fsys, p)
	case ToolReadFile:
		out, err = readFile(fsys, p, a.Offset, a.Limit)
	case ToolGlob:
		out, err = glob(fsys, p, a.Pattern)
	case ToolGrep:
		out, err = grep(fsys, p, a.Pattern, a.Include)
	default:
		err = fmt.Errorf("unknown tool %q", name)
	}
	if err != nil {
		return "error: " + err.Error(), true
	}
	return out, false
}

// clean turns a path the model gave into one fs.FS takes, refusing those
// that leave the project.
func clean(p string) (string, error) {
	p = strings.ReplaceAll(strings.TrimSpace(p), `\`, "/")
	q := path.Clean("/" + p)[1:]
	if q == "" {
		q = "."
	}
	if strings.HasPrefix(path.Clean(p), "..") || !fs.ValidPath(q) {
		return "", fmt.Errorf("%q is outside the project", p)
	}
	return q, nil
}

func join(dir, name string) string {
	if dir == "." {
		return name
	}
	return dir + "/" + name
}

// skipped are folders that hold what no chart needs and walks skip:
// version control, dependencies and build output.
func skipped(name string) bool {
	switch name {
	case "node_modules", "vendor", "dist", "build", "target", "bin", "obj", "__pycache__":
		return true
	}
	return strings.HasPrefix(name, ".") && name != "."
}

func listDir(fsys fs.FS, dir string) (string, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for i, e := range entries {
		if i == maxEntries {
			fmt.Fprintf(&b, "… %d more\n", len(entries)-i)
			break
		}
		if e.IsDir() {
			b.WriteString(e.Name() + "/\n")
			continue
		}
		size := ""
		if info, err := e.Info(); err == nil {
			size = fmt.Sprintf("  %d B", info.Size())
		}
		b.WriteString(e.Name() + size + "\n")
	}
	if b.Len() == 0 {
		return "(empty folder)", nil
	}
	return b.String(), nil
}

// text reads the file at name, refusing folders, big files and binary
// ones.
func text(fsys fs.FS, name string) (string, error) {
	info, err := fs.Stat(fsys, name)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("%s is a folder; use %s", name, ToolListDir)
	}
	if info.Size() > maxFileSize {
		return "", fmt.Errorf("%s is too big to read (%d B)", name, info.Size())
	}
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return "", err
	}
	if bytes.IndexByte(data[:min(len(data), 8<<10)], 0) >= 0 {
		return "", fmt.Errorf("%s is not a text file", name)
	}
	return string(data), nil
}

// cut shortens a long line.
func cut(line string) string {
	if utf8.RuneCountInString(line) <= maxLineRunes {
		return line
	}
	return string([]rune(line)[:maxLineRunes]) + "…"
}

func readFile(fsys fs.FS, name string, offset, limit int) (string, error) {
	data, err := text(fsys, name)
	if err != nil {
		return "", err
	}
	lines := strings.Split(strings.TrimSuffix(data, "\n"), "\n")
	offset = max(offset, 1)
	if limit <= 0 {
		limit = readLines
	}
	if offset > len(lines) {
		return "", fmt.Errorf("%s has %d lines", name, len(lines))
	}
	end := min(len(lines), offset-1+limit)
	var b strings.Builder
	for i := offset - 1; i < end; i++ {
		fmt.Fprintf(&b, "%6d\t%s\n", i+1, cut(strings.TrimSuffix(lines[i], "\r")))
	}
	if end < len(lines) {
		fmt.Fprintf(&b, "… %d more lines; read on with offset=%d\n", len(lines)-end, end+1)
	}
	return b.String(), nil
}

// walk calls fn with each file under dir, by its path from the root and
// from dir, until fn returns false.
func walk(fsys fs.FS, dir string, fn func(p, rel string) bool) error {
	stop := errors.New("stop")
	err := fs.WalkDir(fsys, dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if p == dir {
				return err
			}
			return nil
		}
		if d.IsDir() {
			if p != dir && skipped(d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		rel := p
		if dir != "." {
			rel = strings.TrimPrefix(p, dir+"/")
		}
		if !fn(p, rel) {
			return stop
		}
		return nil
	})
	if errors.Is(err, stop) {
		return nil
	}
	return err
}

// match reports whether name, a slash path, matches pattern, where **
// matches any run of folders.
func match(pattern, name string) bool {
	return matchParts(strings.Split(pattern, "/"), strings.Split(name, "/"))
}

func matchParts(pattern, name []string) bool {
	if len(pattern) == 0 {
		return len(name) == 0
	}
	if pattern[0] == "**" {
		for i := 0; i <= len(name); i++ {
			if matchParts(pattern[1:], name[i:]) {
				return true
			}
		}
		return false
	}
	if len(name) == 0 {
		return false
	}
	ok, _ := path.Match(pattern[0], name[0])
	return ok && matchParts(pattern[1:], name[1:])
}

func glob(fsys fs.FS, dir, pattern string) (string, error) {
	pattern = strings.TrimPrefix(strings.TrimSpace(pattern), "./")
	if pattern == "" {
		return "", errors.New("no pattern")
	}
	if _, err := path.Match(strings.ReplaceAll(pattern, "**", "*"), ""); err != nil {
		return "", err
	}
	var found []string
	more := false
	err := walk(fsys, dir, func(p, rel string) bool {
		if match(pattern, rel) {
			if len(found) == maxEntries {
				more = true
				return false
			}
			found = append(found, p)
		}
		return true
	})
	if err != nil {
		return "", err
	}
	if len(found) == 0 {
		return "no files match", nil
	}
	sort.Strings(found)
	out := strings.Join(found, "\n") + "\n"
	if more {
		out += fmt.Sprintf("… stopped at %d files; narrow the pattern\n", maxEntries)
	}
	return out, nil
}

func grep(fsys fs.FS, dir, pattern, include string) (string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", err
	}
	include = strings.TrimSpace(include)
	var b strings.Builder
	n := 0
	err = walk(fsys, dir, func(p, rel string) bool {
		if include != "" && !match(include, path.Base(p)) && !match(include, rel) {
			return true
		}
		data, err := text(fsys, p)
		if err != nil {
			return true
		}
		for i, line := range strings.Split(data, "\n") {
			if !re.MatchString(line) {
				continue
			}
			if n == maxMatches {
				fmt.Fprintf(&b, "… stopped at %d matches; narrow the pattern\n", maxMatches)
				n++
				return false
			}
			fmt.Fprintf(&b, "%s:%d: %s\n", p, i+1, cut(strings.TrimSpace(strings.TrimSuffix(line, "\r"))))
			n++
		}
		return true
	})
	if err != nil {
		return "", err
	}
	if n == 0 {
		return "no matches", nil
	}
	return b.String(), nil
}
