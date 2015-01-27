// A library to Manage Memset snapshots
package snapshot

// FIXME return the total bytes from putChunked and adjust the manifest appropriately with source size (if .raw) or returned size (if .raw.gz)

import (
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ncw/swift"
)

const (
	// Name of the container with the shapshots
	DefaultContainer = "miniserver-snapshots"
	// Date format for the snapshots directory names
	DirectoryDate = "2006-01-02-15-04-05"
	// Python date format as used in the README.txt
	ReadmeDateFormat = "2006-01-02T15:04:05.999999999"
)

// Describes a snapshot
type Snapshot struct {
	Manager    *Manager
	Name       string
	Path       string
	Comment    string
	Date       time.Time
	ReadMe     string
	Broken     bool
	Miniserver string
	ImageType  string
	ImageLeaf  string
	Md5        string
	DiskSize   int64
}

// Return whether the snapshot exists
func (s *Snapshot) Exists() (bool, error) {
	objects, err := s.Manager.Swift.Objects(s.Manager.Container, &swift.ObjectsOpts{
		Prefix:    s.Name + "/",
		Delimiter: '/',
	})
	if err == swift.ContainerNotFound || err == swift.ObjectNotFound {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to list snapshots: %v", err)
	}
	return len(objects) != 0, nil
}

// Lists the snapshot to stdout
func (s *Snapshot) List() {
	fmt.Printf("%s\n", s.Name)
	if s.Comment != "" {
		fmt.Printf("  Comment    - %s\n", s.Comment)
	}
	if s.Path != "" {
		fmt.Printf("  Path       - %s\n", s.Path)
	}
	if !s.Date.IsZero() {
		fmt.Printf("  Date       - %s\n", s.Date)
	}
	fmt.Printf("  Broken     - %v\n", s.Broken)
	if s.Miniserver != "" {
		fmt.Printf("  Miniserver - %s\n", s.Miniserver)
	}
	if s.ImageType != "" {
		fmt.Printf("  ImageType  - %s\n", s.ImageType)
	}
	if s.ImageLeaf != "" {
		fmt.Printf("  ImageLeaf  - %s\n", s.ImageLeaf)
	}
	if s.Md5 != "" {
		fmt.Printf("  Md5        - %s\n", s.Md5)
	}
	if s.DiskSize != 0 {
		fmt.Printf("  DiskSize   - %d\n", s.DiskSize)
	}
}

// Parses the README.txt
func (s *Snapshot) ParseReadme(readme string) {
	var err error
	s.ReadMe = readme
	for _, line := range strings.Split(readme, "\n") {
		if !strings.Contains(line, "=") {
			continue
		}
		tokens := strings.SplitN(line, "=", 2)
		token := strings.ToLower(strings.TrimSpace(tokens[0]))
		value := strings.TrimSpace(tokens[1])
		switch token {
		case "user_comment":
			s.Comment = value
		case "date": // 2015-01-08T15:44:16.695676
			s.Date, err = time.Parse(ReadmeDateFormat, value)
			if err != nil {
				log.Printf("Failed to parse date from %q: %v", value, err)
			}
		case "miniserver": // myaccaa1
			s.Miniserver = value
		case "image_type": // Tarball file
			s.ImageType = value
		case "snapshot_image": // myacaa1.tar
			s.ImageLeaf = value
		case "md5(snapshot_image)": // 09e29a798ec4f3e4273981cc176adc32
			s.Md5 = value
		case "disk_size": // 42949672960
			s.DiskSize, err = strconv.ParseInt(value, 10, 64)
			if err != nil {
				log.Printf("Failed to parse disk size from %q: %v", value, err)
			}
		}
	}
}

// Creates the README from the Snapshot
func (s *Snapshot) CreateReadme() {
	out := new(bytes.Buffer)
	fmt.Fprintf(out, `; This directory contains a virtual machine disk image snapshot.
; The files in this directory are described below.
; For more information see: http://www.memset.com/docs/
;
; Uploaded by snapshot-manager on %v to %q
;
`, time.Now(), s.Name)
	if !s.Date.IsZero() {
		fmt.Fprintf(out, "date = %s\n", s.Date.Format(ReadmeDateFormat))
	}
	if s.Miniserver != "" {
		fmt.Fprintf(out, "miniserver = %s\n", s.Miniserver)
	}
	if s.Comment != "" {
		fmt.Fprintf(out, "user_comment = %s\n", s.Comment)
	}
	if s.ImageType != "" {
		fmt.Fprintf(out, "image_type = %s\n", s.ImageType)
	}
	if s.ImageLeaf != "" {
		fmt.Fprintf(out, "snapshot_image = %s\n", s.ImageLeaf)
	}
	if s.Md5 != "" {
		fmt.Fprintf(out, "md5(snapshot_image) = %s\n", s.Md5)
	}
	if s.DiskSize != 0 {
		fmt.Fprintf(out, "disk_size = %d\n", s.DiskSize)
	}
	s.ReadMe = out.String()
}

// putChunkedFile puts in to continer/obectPath storing the chunks in
// chunksContainer/chunksPath.  It returns the number of bytes
// uploaded and an error
func (s *Snapshot) putChunkedFile(in io.Reader, container, objectPath string, chunksContainer, chunksPath string, mimeType string) (int64, error) {
	// Pool of buffers for upload
	bufPool := sync.Pool{
		New: func() interface{} {
			return make([]byte, s.Manager.ChunkSize)
		},
	}

	const inFlight = 2
	type upload struct {
		chunkPath string
		buf       []byte
		n         int
	}
	uploads := make(chan upload, inFlight)
	errs := make(chan error, inFlight)

	// Read chunks from the file
	size := int64(0)
	finished := false
	go func() {
		for chunk := 1; !finished; chunk++ {
			buf := bufPool.Get().([]byte)
			n, err := io.ReadFull(in, buf)
			size += int64(n)
			if err == io.EOF {
				break
			} else if err == io.ErrUnexpectedEOF {
				finished = true
			} else if err != io.ErrUnexpectedEOF && err != nil {
				errs <- fmt.Errorf("error reading %v", err)
				break
			}
			uploads <- upload{
				chunkPath: fmt.Sprintf("%s/%04d", chunksPath, chunk),
				buf:       buf,
				n:         n,
			}
		}
		close(uploads)
	}()

	// Upload chunks as they come in
	go func() {
		for upload := range uploads {
			// FIXME retry
			log.Printf("Uploading chunk %q", upload.chunkPath)
			err := s.Manager.Swift.ObjectPutBytes(container, upload.chunkPath, upload.buf[:upload.n], mimeType)
			if err != nil {
				errs <- fmt.Errorf("failed to upload chunk %q: %v", upload.chunkPath, err)
			}
			bufPool.Put(upload.buf)
		}
		close(errs)
	}()

	// Collect errors
	for err := range errs {
		finished = true
		return size, err
	}

	// Put the manifest if all was successful
	log.Printf("Uploading manifest %q", objectPath)
	contents := strings.NewReader("")
	headers := swift.Headers{
		"X-Object-Manifest": chunksContainer + "/" + chunksPath,
	}
	_, err := s.Manager.Swift.ObjectPut(container, objectPath, contents, true, "", "application/octet-stream", headers)
	return size, err
}

// Download a snapshot into outputDirectory
func (s *Snapshot) Get(outputDirectory string) error {
	objects, err := s.Manager.Objects(s.Name)
	if len(objects) == 0 {
		log.Fatal("Snapshot or snapshot objects not found")
	}
	err = os.MkdirAll(outputDirectory, 0755)
	if err != nil {
		return fmt.Errorf("failed to make output directory %q", outputDirectory)
	}
	err = os.Chdir(outputDirectory)
	if err != nil {
		return fmt.Errorf("failed chdir output directory %q", outputDirectory)
	}
	for _, object := range objects {
		if object.PseudoDirectory {
			continue
		}
		objectPath := object.Name
		leaf := path.Base(objectPath)
		fmt.Printf("Downloading %s\n", objectPath)
		out, err := os.Create(leaf)
		if err != nil {
			return fmt.Errorf("failed to open output file %q: %v", leaf, err)
		}
		_, err = s.Manager.Swift.ObjectGet(s.Manager.Container, objectPath, out, false, nil) // don't check MD5 because they are wrong for chunked files
		if err != nil {
			return fmt.Errorf("failed to download %q: %v", s.Name, err)
		}
		err = out.Close()
		if err != nil {
			return fmt.Errorf("failed to close %q: %v", s.Name, err)
		}
	}
	return nil
}

// checkClose is used to check the return from Close in a defer
// statement.
func checkClose(c io.Closer, err *error) {
	cerr := c.Close()
	if *err == nil {
		*err = cerr
	}
}

// countWriter acts as an io.Writer counting the output
type countWriter int64

// Write counts up the data and ignores it
func (c *countWriter) Write(p []byte) (int, error) {
	*c += countWriter(len(p))
	return len(p), nil
}

// Puts a snapshot
func (s *Snapshot) Put(file string) (err error) {
	// Work out where to put things
	leaf := s.ImageLeaf
	Type := Types.Find(file)
	if Type == nil {
		return fmt.Errorf("unknown snapshot type %q - use types command to see available", leaf)
	}
	if !Type.Upload {
		return fmt.Errorf("can't upload snapshot type %q - use types command to see available", leaf)
	}
	s.ImageType = Type.ImageType
	chunksPath := s.Name + "/" + leaf[:len(leaf)-len(Type.Suffix)] + ".part"
	objectPath := s.Path

	// Get file stat
	fi, err := os.Stat(file)
	if err != nil {
		return fmt.Errorf("failed to stat %q: %v", file, err)
	}
	if fi.IsDir() {
		return fmt.Errorf("%q is a directory", file)
	}
	s.Date = fi.ModTime()

	// Check file doesn't exist and container does
	ok, err := s.Exists()
	if err != nil {
		return err
	}
	if ok {
		return fmt.Errorf("snapshot %q already exists - delete it first", s.Name)
	}
	err = s.Manager.CreateContainer()
	if err != nil {
		return err
	}

	// Upload the file with chunks
	var in io.Reader
	fileIn, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("failed to open %q: %v", file, err)
	}
	in = fileIn
	defer checkClose(fileIn, &err)

	// If we need to read the size from the ungzipped data then do
	// it as we go along
	var gzipCounter *GzipCounter
	if Type.DiskSizeFrom == DiskSizeFromGzip {
		log.Printf("Gunzipping on the fly to count size")
		gzipCounter, err = NewGzipCounter()
		if err != nil {
			return fmt.Errorf("failed to make gzip counter: %v", err)
		}
		in = io.TeeReader(in, gzipCounter)
	}

	// Check if needs gunzip
	if Type.NeedsGunzip {
		log.Printf("Gunzipping on the fly")
		objectPath = objectPath[:len(objectPath)-3]
		s.ImageLeaf = s.ImageLeaf[:len(s.ImageLeaf)-3]
		var gzipRd io.ReadCloser
		gzipRd, err = gzip.NewReader(in)
		if err != nil {
			return fmt.Errorf("failed to make gzip decompressor: %v", err)
		}
		defer checkClose(gzipRd, &err)
		in = gzipRd
	}

	// Check if needs gzip
	if Type.NeedsGzip {
		log.Printf("Gzipping on the fly")
		objectPath += ".gz"
		s.ImageLeaf += ".gz"
		var gzipRd io.ReadCloser
		gzipRd, err = NewGzipReader(in)
		if err != nil {
			return fmt.Errorf("failed to make gzip compressor: %v", err)
		}
		defer checkClose(gzipRd, &err)
		in = gzipRd
	}

	// Calculate the MD5 of the uploaded object on the fly
	hash := md5.New()
	in = io.TeeReader(in, hash)

	// Put the file in chunks
	size, err := s.putChunkedFile(in, s.Manager.Container, objectPath, s.Manager.Container, chunksPath, Type.MimeType)
	if err != nil {
		return err
	}

	// Set the Md5
	s.Md5 = fmt.Sprintf("%x", hash.Sum(nil))

	// Set the DiskSize to the raw size of the upload
	switch Type.DiskSizeFrom {
	case DiskSizeFromUpload:
		// .tar.gz -> .tar
		s.DiskSize = size
	case DiskSizeFromFile:
		// .raw -> raw.gz
		// .tar
		s.DiskSize = fi.Size()
	case DiskSizeFromGzip:
		// .raw.gz
		err = gzipCounter.Close()
		if err != nil {
			return fmt.Errorf("error closing gzip counter: %v", err)
		}
		s.DiskSize = gzipCounter.Size()
	default:
		log.Printf("Can't figure out the disk size for %q - using the file size", Type.Suffix)
		s.DiskSize = fi.Size()
	}

	// Write the README.txt
	s.CreateReadme()
	log.Printf("Uploading README.txt\n%s\n", s.ReadMe)
	err = s.Manager.Swift.ObjectPutString(s.Manager.Container, s.Name+"/README.txt", s.ReadMe, "text/plain")
	if err != nil {
		return fmt.Errorf("failed to create README.txt: %v", err)
	}
	return nil
}

// Delete all the objects in the snapshot
func (s *Snapshot) Delete() error {
	objects, err := s.Manager.Swift.Objects(s.Manager.Container, &swift.ObjectsOpts{
		Prefix: s.Name + "/",
	})
	if err != nil {
		return fmt.Errorf("failed to read snapshot %q: %v", s.Name, err)
	}
	if len(objects) == 0 {
		return fmt.Errorf("snapshot or snapshot objects not found")
	}

	errors := 0
	for _, object := range objects {
		if object.PseudoDirectory {
			continue
		}
		log.Printf("Deleting %q", object.Name)
		err = s.Manager.Swift.ObjectDelete(s.Manager.Container, object.Name)
		if err != nil {
			errors += 1
			log.Printf("Failed to delete %q: %v", object.Name, err)
		}
	}
	if errors != 0 {
		return fmt.Errorf("failed to delete %d objects", errors)
	}
	return nil
}
