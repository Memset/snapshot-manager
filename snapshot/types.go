package snapshot

import (
	"fmt"
	"io"
	"strings"
)

type DiskSizeFrom int

const (
	DiskSizeFromUnknown = DiskSizeFrom(iota)
	DiskSizeFromFile
	DiskSizeFromUpload
	DiskSizeFromGzip
)

// Describe a Type
type Type struct {
	Suffix         string
	Upload         bool
	Virtualisation string
	Comment        string
	ImageType      string // for README.txt
	MimeType       string
	NeedsGzip      bool
	NeedsGunzip    bool
	DiskSizeFrom   DiskSizeFrom
}

// A list of types
type types []Type

// A list of snapshot types
var Types = types{
	{
		Suffix:         ".tar",
		Upload:         true,
		Virtualisation: "Paravirtualisation - Linux only",
		Comment:        "A tar of whole file system",
		ImageType:      "Tarball file",
		MimeType:       "application/x-tar",
		DiskSizeFrom:   DiskSizeFromFile,
	},
	{
		Suffix:         ".tar.gz",
		Upload:         true,
		Virtualisation: "Paravirtualisation - Linux only",
		Comment:        "A tar of whole file system",
		ImageType:      "Tarball file",
		MimeType:       "application/x-tar",
		NeedsGunzip:    true,
		DiskSizeFrom:   DiskSizeFromUpload,
	},
	{
		Suffix:         ".raw.gz",
		Upload:         true,
		Virtualisation: "Full virtualisation with PV Drivers",
		Comment:        "A raw disk image including partitions, gzipped",
		ImageType:      "gzipped Raw file",
		MimeType:       "x-application/x-gzip",
		DiskSizeFrom:   DiskSizeFromGzip,
	},
	{
		Suffix:         ".raw",
		Upload:         true,
		Virtualisation: "Full virtualisation with PV Drivers",
		Comment:        "A raw disk image including partitions",
		ImageType:      "gzipped Raw file",
		MimeType:       "x-application/x-gzip",
		NeedsGzip:      true,
		DiskSizeFrom:   DiskSizeFromFile,
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
	// {
	// 	Suffix:         ".qcow2",
	// 	Upload:         true,
	// 	Virtualisation: "Full virtualisation with PV Drivers",
	// 	Comment:        "Raw disk image with partitions, QCOW2 format",
	// 	ImageType:      "QCOW2",
	// 	MimeType:       "application/octet-stream",
	// },
}

// Finds the best match for Type for the file passed in
//
// Returns nil if not found
func (ts types) Find(file string) *Type {
	for i := range ts {
		Type := &ts[i]
		if strings.HasSuffix(file, Type.Suffix) {
			return Type
		}
	}
	return nil
}

// Lists all the snapshot types to an io.Writer
func (ts types) List(out io.Writer) {
	for i := range ts {
		Type := &ts[i]
		fmt.Fprintf(out, "%s - %s\n", Type.Suffix, Type.ImageType)
		fmt.Fprintf(out, "  Upload:         %v\n", Type.Upload)
		fmt.Fprintf(out, "  Comment:        %s\n", Type.Comment)
		fmt.Fprintf(out, "  Virtualisation: %s\n", Type.Virtualisation)
	}
}
