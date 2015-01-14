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

// GzipCounter is an adaptor to allow reading compressed data from an
// uncompressed input
type GzipCounter struct {
	w      io.Writer
	gzipRd io.ReadCloser
	pipeRd io.ReadCloser
	pipeWr io.WriteCloser
	err    error
	count  int64
	closed chan struct{}
}

// NewGzipCounter gunzips any data written to it and counts the bytes.
// The Size method can be used to return the numer of uncompressed
// bytes.
func NewGzipCounter() (*GzipCounter, error) {
	// Pump data through the pipe into a gzip.Reader
	z := &GzipCounter{
		closed: make(chan struct{}),
	}
	z.pipeRd, z.pipeWr = io.Pipe()
	// Read the data from the pipe and count it up
	go func() {
		var err error
		z.gzipRd, err = gzip.NewReader(z.pipeRd)
		z.setErr(err)
		if err != nil {
			return
		}
		buf := make([]byte, 16*1024)
		for {
			n, err := z.gzipRd.Read(buf)
			z.count += int64(n)
			if err != nil {
				if err != io.EOF {
					z.setErr(err)
				}
				break
			}
		}
		z.setErr(z.gzipRd.Close())
		z.setErr(z.pipeRd.Close())
		close(z.closed)
	}()
	return z, nil
}

// setErr sets z.err if it is nil and err != nil
func (z *GzipCounter) setErr(err error) {
	if err != nil && z.err == nil {
		z.err = err
	}
}

// Write compressed data
func (z *GzipCounter) Write(p []byte) (int, error) {
	if z.err != nil {
		return 0, z.err
	}
	return z.pipeWr.Write(p)
}

// Close the writer - you must call this and check the error
func (z *GzipCounter) Close() error {
	z.setErr(z.pipeWr.Close())
	<-z.closed
	return z.err
}

// Returns the count of bytes - run after Close
func (z *GzipCounter) Size() int64 {
	return z.count
}
