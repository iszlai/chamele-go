// Package languages provides the Reader interface and the language registry.
// Language readers register themselves via init() functions in their
// respective sub-packages.
package languages

import "iter"

// Reader is the interface implemented by each language reader.
// It is a stable public API from v0.1.
type Reader interface {
	// Extensions returns the file extensions this reader handles (without dot).
	Extensions() []string
	// LanguageNames returns the canonical names for this language (e.g. {"cpp","c"}).
	LanguageNames() []string
	// Tokenize returns a token sequence for the given source bytes.
	Tokenize(src []byte) iter.Seq[string]
}

var registry []Reader

// Register adds a reader to the global registry. Call from init().
func Register(r Reader) { registry = append(registry, r) }

// GetReaderForFilename returns the reader whose extension list matches the
// file's extension, or nil if none matches.
func GetReaderForFilename(path string) Reader {
	ext := extension(path)
	for _, r := range registry {
		for _, e := range r.Extensions() {
			if e == ext {
				return r
			}
		}
	}
	return nil
}

// Get returns the first reader whose LanguageNames contains the given name.
func Get(name string) Reader {
	for _, r := range registry {
		for _, n := range r.LanguageNames() {
			if n == name {
				return r
			}
		}
	}
	return nil
}

func extension(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '.' {
			return path[i+1:]
		}
		if path[i] == '/' || path[i] == '\\' {
			break
		}
	}
	return ""
}
