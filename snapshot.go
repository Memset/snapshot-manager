package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ncw/swift"
)

const (
	// Name of the container with the shapshots
	snapshotContainer = "miniserver-snapshots"
	// Date format for the snapshots directory names
	snapshotDirectoryDate = "2006-01-02-15-04-05"
	// Python date format as used in the README.txt
	pyDateFormat = "2006-01-02T15:04:05.999999999"
)

// Check the snapshotContainer exists
func findContainer() bool {
	_, _, err := c.Container(snapshotContainer)
	if err == swift.ContainerNotFound {
		return false
	}
	if err != nil {
		log.Fatalf("Error for container %q: %v", snapshotContainer, err)
	}
	return true
}

// Describe a snapshotType
type snapshotType struct {
	Suffix         string
	Upload         bool
	Virtualisation string
	Comment        string
	ImageType      string // for README.txt
	MimeType       string
}

// A list of snapshot types
//
// FIXME allow .tar.gz & .tgz ?
// FIXME does .raw.gz work?
// FIXME allow .raw?
var snapshotTypes = []snapshotType{
	{
		Suffix:         ".tar",
		Upload:         true,
		Virtualisation: "Paravirtualisation - Linux only",
		Comment:        "A tar of whole file system",
		ImageType:      "Tarball file",
		MimeType:       "application/x-tar",
	},
	{
		Suffix:         ".raw.gz",
		Upload:         true,
		Virtualisation: "Full virtualisation with PV Drivers",
		Comment:        "A raw disk image including partitions, gzipped",
		ImageType:      "gzipped Raw file",
		MimeType:       "x-application/x-gzip",
	},
	{
		Suffix:         ".xmbr",
		Upload:         false, // don't allow ntfsclone uploads - too complicated
		Virtualisation: "Full virtualisation with PV Drivers",
		Comment:        "ntfsclone + boot sector + partitions",
		ImageType:      "gzipped NTFS Image file",
		MimeType:       "x-application/x-gzip",
	},
	{
		Suffix:         ".vmdk",
		Upload:         false, // FIXME
		Virtualisation: "Full virtualisation with PV Drivers",
		Comment:        "Raw disk image with partitions, VMDK format",
		ImageType:      "VMDK",
		MimeType:       "application/vmdk",
	},
	{
		Suffix:         ".vhd",
		Upload:         false, // FIXME
		Virtualisation: "Full virtualisation with PV Drivers",
		Comment:        "Raw disk image with partitions, VMDK format",
		ImageType:      "VHD",
		MimeType:       "application/vhd",
	},
	{
		Suffix:         ".qcow2",
		Upload:         true,
		Virtualisation: "Full virtualisation with PV Drivers",
		Comment:        "Raw disk image with partitions, QCOW2 format",
		ImageType:      "QCOW2",
		MimeType:       "application/octet-stream",
	},
}

// Finds the best match for snapshotType for the file passed in
//
// Returns nil if not found
func findSnapshotType(file string) *snapshotType {
	for i := range snapshotTypes {
		snapshotType := &snapshotTypes[i]
		if strings.HasSuffix(file, snapshotType.Suffix) {
			return snapshotType
		}
	}
	return nil
}

// Lists all the snapshot types to a string
func listSnapshotTypes(out io.Writer) {
	for i := range snapshotTypes {
		snapshotType := &snapshotTypes[i]
		fmt.Fprintf(out, "%s - %s\n", snapshotType.Suffix, snapshotType.ImageType)
		fmt.Fprintf(out, "  Upload:         %v\n", snapshotType.Upload)
		fmt.Fprintf(out, "  Comment:        %s\n", snapshotType.Comment)
		fmt.Fprintf(out, "  Virtualisation: %s\n", snapshotType.Virtualisation)
	}
}

// Describes a snapshot
type Snapshot struct {
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
func (snapshot *Snapshot) Exists() bool {
	objects, err := c.Objects(snapshotContainer, &swift.ObjectsOpts{
		Prefix:    snapshot.Name + "/",
		Delimiter: '/',
	})
	if err == swift.ContainerNotFound || err == swift.ObjectNotFound {
		return false
	}
	if err != nil {
		log.Fatalf("Failed to list snapshots: %v", err)
	}
	return len(objects) != 0
}

// Lists the snapshot to stdout
func (snapshot *Snapshot) List() {
	fmt.Printf("%s\n", snapshot.Name)
	if snapshot.Comment != "" {
		fmt.Printf("  Comment    - %s\n", snapshot.Comment)
	}
	if snapshot.Path != "" {
		fmt.Printf("  Path       - %s\n", snapshot.Path)
	}
	if !snapshot.Date.IsZero() {
		fmt.Printf("  Date       - %s\n", snapshot.Date)
	}
	fmt.Printf("  Broken     - %v\n", snapshot.Broken)
	if snapshot.Miniserver != "" {
		fmt.Printf("  Miniserver - %s\n", snapshot.Miniserver)
	}
	if snapshot.ImageType != "" {
		fmt.Printf("  ImageType  - %s\n", snapshot.ImageType)
	}
	if snapshot.ImageLeaf != "" {
		fmt.Printf("  ImageLeaf  - %s\n", snapshot.ImageLeaf)
	}
	if snapshot.Md5 != "" {
		fmt.Printf("  Md5        - %s\n", snapshot.Md5)
	}
	if snapshot.DiskSize != 0 {
		fmt.Printf("  DiskSize   - %d\n", snapshot.DiskSize)
	}
}

// Parses the README.txt
func (snapshot *Snapshot) ParseReadme(readme string) {
	var err error
	snapshot.ReadMe = readme
	for _, line := range strings.Split(readme, "\n") {
		if !strings.Contains(line, "=") {
			continue
		}
		tokens := strings.SplitN(line, "=", 2)
		token := strings.ToLower(strings.TrimSpace(tokens[0]))
		value := strings.TrimSpace(tokens[1])
		switch token {
		case "user_comment":
			snapshot.Comment = value
		case "date": // 2015-01-08T15:44:16.695676
			snapshot.Date, err = time.Parse(pyDateFormat, value)
			if err != nil {
				log.Printf("Failed to parse date from %q: %v", value, err)
			}
		case "miniserver": // myaccaa1
			snapshot.Miniserver = value
		case "image_type": // Tarball file
			snapshot.ImageType = value
		case "snapshot_image": // myacaa1.tar
			snapshot.ImageLeaf = value
		case "md5(snapshot_image)": // 09e29a798ec4f3e4273981cc176adc32
			snapshot.Md5 = value
		case "disk_size": // 42949672960
			snapshot.DiskSize, err = strconv.ParseInt(value, 10, 64)
			if err != nil {
				log.Printf("Failed to parse disk size from %q: %v", value, err)
			}
		}
	}
}

// Creates the README from the Snapshot
func (snapshot *Snapshot) CreateReadme() {
	out := new(bytes.Buffer)
	fmt.Fprintf(out, `; This directory contains a virtual machine disk image snapshot.
; The files in this directory are described below.
; For more information see: http://www.memset.com/docs/
;
; Uploaded by snapshot-manager on %v to %q
;
`, time.Now(), snapshot.Name)
	if !snapshot.Date.IsZero() {
		fmt.Fprintf(out, "date = %s\n", snapshot.Date.Format(pyDateFormat))
	}
	if snapshot.Miniserver != "" {
		fmt.Fprintf(out, "miniserver = %s\n", snapshot.Miniserver)
	}
	if snapshot.Comment != "" {
		fmt.Fprintf(out, "user_comment = %s\n", snapshot.Comment)
	}
	if snapshot.ImageType != "" {
		fmt.Fprintf(out, "image_type = %s\n", snapshot.ImageType)
	}
	if snapshot.ImageLeaf != "" {
		fmt.Fprintf(out, "snapshot_image = %s\n", snapshot.ImageLeaf)
	}
	if snapshot.Md5 != "" {
		fmt.Fprintf(out, "md5(snapshot_image) = %s\n", snapshot.Md5)
	}
	if snapshot.DiskSize != 0 {
		fmt.Fprintf(out, "disk_size = %d\n", snapshot.DiskSize)
	}
	snapshot.ReadMe = out.String()
}

// putChunkedFile puts file of size to continer/obectPath storing the chunks in chunksContainer/chunksPath
func putChunkedFile(file string, size int64, container, objectPath string, chunksContainer, chunksPath string, mimeType string) {
	// Open input file
	in, err := os.Open(file)
	if err != nil {
		log.Fatalf("Failed to open %q: %v", file, err)
	}
	defer in.Close()

	// Read chunks from the file
	chunk := 1
	buf := make([]byte, *chunkSize)
	for size > 0 {
		if size < int64(len(buf)) {
			buf = buf[:size]
		}
		n, err := io.ReadFull(in, buf)
		if err != nil {
			log.Fatalf("Error reading %q: %v", file, err)
		}
		size -= int64(n)
		chunkPath := fmt.Sprintf("%s/%04d", chunksPath, chunk)
		// FIXME retry
		log.Printf("Uploading chunk %q", chunkPath)
		err = c.ObjectPutBytes(container, chunkPath, buf, mimeType)
		if err != nil {
			log.Fatalf("Failed to upload chunk %q: %v", chunkPath, err)
		}
		chunk += 1
	}

	// Put the manifest if all was successful
	log.Printf("Uploading manifest %q", objectPath)
	contents := strings.NewReader("")
	headers := swift.Headers{
		"X-Object-Manifest": chunksContainer + "/" + chunksPath,
	}
	_, err = c.ObjectPut(container, objectPath, contents, true, "", "application/octet-stream", headers)
}

// Puts a snapshot
func (snapshot *Snapshot) Put(file string) {
	// Work out where to put things
	leaf := snapshot.ImageLeaf
	snapshotType := findSnapshotType(file)
	if snapshotType == nil {
		log.Fatalf("Unknown snapshot type %q - use types command to see available", leaf)
	}
	if !snapshotType.Upload {
		log.Fatalf("Can't upload snapshot type %q - use types command to see available", leaf)
	}
	snapshot.ImageType = snapshotType.ImageType
	chunksPath := snapshot.Name + "/" + leaf[:len(leaf)-len(snapshotType.Suffix)]

	// Get file stat
	fi, err := os.Stat(file)
	if err != nil {
		log.Fatalf("Failed to stat %q: %v", file, err)
	}
	if fi.IsDir() {
		log.Fatalf("%q is a directory", file)
	}
	snapshot.Date = fi.ModTime()
	snapshot.DiskSize = fi.Size() // FIXME not right for non raw images

	// Check file doesn't exist and container does
	if snapshot.Exists() {
		log.Fatalf("Snapshot %q already exists - delete it first", snapshot.Name)
	}
	if !findContainer() {
		err := c.ContainerCreate(snapshotContainer, nil)
		if err != nil {
			log.Fatalf("Failed to create container %q: %v", snapshotContainer, err)
		}
	}

	// Upload the file with chunks
	putChunkedFile(file, fi.Size(), snapshotContainer, snapshot.Path, snapshotContainer, chunksPath, snapshotType.MimeType)

	// Write the README.txt
	snapshot.CreateReadme()
	err = c.ObjectPutString(snapshotContainer, snapshot.Name+"/README.txt", snapshot.ReadMe, "text/plain")
	if err != nil {
		log.Fatalf("Failed to create README.txt: %v", err)
	}
}

// Read the objects in the snapshot
func getSnapshotObjects(name string) []swift.Object {
	objects, err := c.Objects(snapshotContainer, &swift.ObjectsOpts{
		Prefix:    name + "/",
		Delimiter: '/',
	})
	if err != nil {
		log.Fatalf("Failed to read snapshot %q: %v", name, err)
	}
	return objects
}

// Gets information about the named snapshot
func getSnapshot(name string) *Snapshot {
	var snapshot Snapshot
	snapshot.Name = name

	objects := getSnapshotObjects(name)

	// check for README.txt for the user comment
	for _, object := range objects {
		if strings.HasSuffix(object.Name, "README.txt") {
			readme, err := c.ObjectGetString(snapshotContainer, object.Name)
			if err != nil {
				log.Printf("Couldn't read %q - ignoring: %v", object.Name, err)
				continue
			}
			snapshot.ParseReadme(readme)
		}
	}

	// we could get these from the README.txt, but currently this is
	// easier/more reliable than parsing the .txt file
	for _, object := range objects {
		snapshotType := findSnapshotType(object.Name)
		if snapshotType != nil {
			snapshot.Path = object.Name
			if snapshot.Date.IsZero() {
				snapshot.Date = object.LastModified
			}
			break
		}
	}

	// it might be a broken or active snapshot
	if snapshot.Path == "" {
		snapshot.Broken = true
		snapshot.Comment = "The snapshot probably failed and some files were left behind."
	}

	return &snapshot
}

// Information about snapshots found
func getSnapshots() []*Snapshot {
	if !findContainer() {
		return nil
	}
	objects, err := c.Objects(snapshotContainer, &swift.ObjectsOpts{
		Prefix:    "",
		Delimiter: '/',
	})
	if err != nil {
		log.Fatalf("Failed to list snapshots: %v", err)
	}
	if len(objects) == 0 {
		return nil
	}
	var snapshots []*Snapshot
	for _, obj := range objects {
		if obj.PseudoDirectory {
			name := strings.TrimRight(obj.Name, "/")
			snapshots = append(snapshots, getSnapshot(name))
		}
	}
	return snapshots
}
