package snapshot

import (
	"compress/gzip"
	"io"
)

// GzipReader is an adaptor to allow reading compressed data from an
// uncompressed input
type GzipReader struct {
	fileRd         io.Reader
	pipeRd         io.ReadCloser
	pipeWr, gzipWr io.WriteCloser
	err            error
}

// NewGzipReader takes an io.Reader and returns an io.ReadCloser which
// reads compressed data.
func NewGzipReader(fileRd io.Reader) (*GzipReader, error) {
	// Pump data into gzip.Writer through the pipe and
	// give a reader to putChunkedFile
	var err error
	z := &GzipReader{
		fileRd: fileRd,
	}
	z.pipeRd, z.pipeWr = io.Pipe()
	z.gzipWr, err = gzip.NewWriterLevel(z.pipeWr, 6)
	if err != nil {
		return nil, err
	}
	// Pump the data through the pipe and return errors
	go func() {
		_, err := io.Copy(z.gzipWr, z.fileRd)
		z.setErr(err)
		z.setErr(z.gzipWr.Close())
		z.setErr(z.pipeWr.Close())
	}()
	return z, nil
}

// setErr sets z.err if it is nil and err != nil
func (z *GzipReader) setErr(err error) {
	if err != nil && z.err == nil {
		z.err = err
	}
}

// Read compressed data
func (z *GzipReader) Read(p []byte) (int, error) {
	if z.err != nil {
		return 0, z.err
	}
	return z.pipeRd.Read(p)
}

// Close the reader - you must call this and check the error
func (z *GzipReader) Close() error {
	z.setErr(z.pipeRd.Close())
	return z.err
}
