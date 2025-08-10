package speech

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
)

// gzipCompress compresses input data using gzip
func (c *Client) gzipCompress(input []byte) []byte {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	w.Write(input)
	w.Close()
	return b.Bytes()
}

// gzipDecompress decompresses gzip data
func (c *Client) gzipDecompress(input []byte) []byte {
	b := bytes.NewBuffer(input)
	r, _ := gzip.NewReader(b)
	out, _ := ioutil.ReadAll(r)
	r.Close()
	return out
}
