package stringx

import "bytes"

// NormalizeLF replaces all \r\n and bare \r with \n.
// Called once at file-read time so state machines see only \n.
func NormalizeLF(data []byte) []byte {
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	return bytes.ReplaceAll(data, []byte("\r"), []byte("\n"))
}
