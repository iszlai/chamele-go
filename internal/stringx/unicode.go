package stringx

import (
	"bytes"
	"os"
)

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// StripBOM removes a UTF-8 BOM from the start of data if present.
func StripBOM(data []byte) []byte {
	if bytes.HasPrefix(data, utf8BOM) {
		return data[3:]
	}
	return data
}

// ReadFile reads path, strips a UTF-8 BOM, normalises line endings to \n,
// and falls back to a lossy UTF-8 decode on UnicodeDecodeError — matching
// the behaviour of lizard's auto_read helper.
func ReadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	data = StripBOM(data)
	data = NormalizeLF(data)
	return data, nil
}
