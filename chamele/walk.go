package chamele

import (
	"crypto/md5"
	"os"
	"path/filepath"
	"strings"

	"github.com/iszlai/chamele-go/internal/stringx"
	"github.com/iszlai/chamele-go/languages"
	gitignore "github.com/sabhiram/go-gitignore"
)

// sourceFiles returns all source files under paths that pass the language,
// exclude, dedup, and gitignore filters.
//
// Nested .gitignore files are honoured at every directory level, which is a
// deliberate divergence from upstream lizard (see docs/divergences.md).
//
// Dedup is content-based (MD5 over the file contents), matching upstream
// lizard. The dedup read is performed by md5File; the analyzer reads the
// file again to tokenise. F-23 in CLEANUP.md flags this double-read as a
// perf concern; cache-the-bytes is the right fix but adds memory pressure
// for large repos so it's deferred.
func sourceFiles(paths []string, opts Options) []string {
	seen := make(map[[16]byte]bool)
	var result []string

	for _, root := range paths {
		fi, err := os.Stat(root)
		if err != nil {
			continue
		}
		if !fi.IsDir() {
			if shouldInclude(root, opts, seen) {
				result = append(result, root)
			}
			continue
		}

		type frame struct {
			absDir string
			parser *gitignore.GitIgnore
		}
		var ignoreStack []frame
		absRoot, _ := filepath.Abs(root)

		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			absPath, _ := filepath.Abs(path)

			if d.IsDir() {
				// Pop frames whose absDir is no longer an ancestor of absPath.
				// "Ancestor" means absPath == absDir OR absPath starts with
				// absDir + path-separator. We compare cleaned absolute paths
				// directly — no Rel() / fragile rel[:2] check.
				for len(ignoreStack) > 0 {
					top := ignoreStack[len(ignoreStack)-1].absDir
					if absPath != top && !strings.HasPrefix(absPath, top+string(filepath.Separator)) {
						ignoreStack = ignoreStack[:len(ignoreStack)-1]
					} else {
						break
					}
				}
				if p, err2 := gitignore.CompileIgnoreFile(filepath.Join(path, ".gitignore")); err2 == nil {
					ignoreStack = append(ignoreStack, frame{absPath, p})
				}
				return nil
			}

			rel, _ := filepath.Rel(absRoot, absPath)
			for _, f := range ignoreStack {
				if f.parser.MatchesPath(rel) {
					return nil
				}
			}

			if shouldInclude(path, opts, seen) {
				result = append(result, path)
			}
			return nil
		})
	}
	return result
}

func shouldInclude(path string, opts Options, seen map[[16]byte]bool) bool {
	r := languages.GetReaderForFilename(path)
	if r == nil {
		return false
	}
	if !matchesLanguageFilter(r, opts.Languages) {
		return false
	}
	for _, pat := range opts.Exclude {
		if matched, _ := filepath.Match(pat, filepath.Base(path)); matched {
			return false
		}
		if matched, _ := filepath.Match(pat, path); matched {
			return false
		}
	}
	h := md5File(path)
	if seen[h] {
		return false
	}
	seen[h] = true
	return true
}

func matchesLanguageFilter(r languages.Reader, filter []string) bool {
	if len(filter) == 0 {
		return true
	}
	for _, want := range filter {
		for _, have := range r.LanguageNames() {
			if have == want {
				return true
			}
		}
	}
	return false
}

func md5File(path string) [16]byte {
	data, err := stringx.ReadFile(path)
	if err != nil {
		return [16]byte{}
	}
	return md5.Sum(data)
}
