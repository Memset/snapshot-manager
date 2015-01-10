// Memset snapshot manager

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/ncw/swift"
)

// Globals
var (
	// Flags
	chunkSize = flag.Int64("s", 64*1024*1024, "Size of the chunks to make")
	userName  = flag.String("user", "", "Memstore user name, eg myaccaa1.admin")
	apiKey    = flag.String("api-key", "", "Memstore api key")
	authUrl   = flag.String("auth-url", "https://auth.storage.memset.com/v1.0", "Swift Auth URL - default should be OK for Memstore")
	// Swift connection
	c = new(swift.Connection)
)

// List the snapshots available
func listSnapshots() {
	snapshots := getSnapshots()
	if len(snapshots) == 0 {
		fmt.Println("No snapshots found")
		return
	}
	for _, snapshot := range snapshots {
		snapshot.List()
	}
}

// Download a snapshot
func downloadSnaphot(name string) {
	objects := getSnapshotObjects(name)
	if len(objects) == 0 {
		log.Fatal("Snapshot or snapshot objects not found")
	}
	err := os.MkdirAll(name, 0755)
	if err != nil {
		log.Fatalf("Failed to make output directory %q", name)
	}
	err = os.Chdir(name)
	if err != nil {
		log.Fatalf("Failed chdir output directory %q", name)
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
			log.Fatalf("Failed to open output file %q: %v", leaf, err)
		}
		_, err = c.ObjectGet(snapshotContainer, objectPath, out, false, nil) // don't check MD5 because they are wrong for chunked files
		if err != nil {
			log.Fatalf("Failed to download %q: %v", name, err)
		}
		err = out.Close()
		if err != nil {
			log.Fatalf("Failed to close %q: %v", name, err)
		}
	}
}

// Upload a snapshot
func uploadSnaphot(name, file string) {
	leaf := strings.ToLower(path.Base(file))
	Path := name + "/" + leaf

	snapshot := &Snapshot{
		Name:       name,
		Path:       Path,
		Comment:    fmt.Sprintf("Uploaded from original file '%s'", file),
		Broken:     false,
		ImageLeaf:  leaf,
		Miniserver: "uploaded",
	}

	fmt.Printf("Uploading snapshot\n")
	snapshot.List()

	snapshot.Put(file)
}

// Delete a snapshot
func deleteSnaphot(name string) {
	objects, err := c.Objects(snapshotContainer, &swift.ObjectsOpts{
		Prefix: name + "/",
	})
	if err != nil {
		log.Fatalf("Failed to read snapshot %q: %v", name, err)
	}
	if len(objects) == 0 {
		log.Fatalf("Snapshot or snapshot objects not found")
	}

	errors := 0
	for _, object := range objects {
		if object.PseudoDirectory {
			continue
		}
		log.Printf("Deleting %q", object.Name)
		err = c.ObjectDelete(snapshotContainer, object.Name)
		if err != nil {
			errors += 1
			log.Printf("Failed to delete %q: %v", object.Name, err)
		}
	}
	if errors != 0 {
		log.Fatalf("Failed to delete %d objects", errors)
	}
}

// syntaxError prints the syntax
func syntaxError() {
	fmt.Fprintf(os.Stderr, `Manage snapshots in Memset Memstore

snapshot-manager <command> <arguments>

Commands

  list             - lists the snapshots
  download name    - downloads the snapshot
  upload name file - uploads a disk image as a snapshot
  delete name      - deletes the snapshot
  types            - available snapshot types

Full options:
`)
	flag.PrintDefaults()
}

// Exit with the message
func fatalf(message string, args ...interface{}) {
	syntaxError()
	fmt.Fprintf(os.Stderr, message, args...)
	fmt.Fprintln(os.Stderr)
	os.Exit(1)
}

func main() {
	flag.Usage = syntaxError
	flag.Parse()
	args := flag.Args()

	if *userName == "" || *apiKey == "" {
		fatalf("Flags -user and -api-key required")
	}

	if len(args) < 1 {
		fatalf("No command supplied")
	}
	command := strings.ToLower(args[0])
	args = args[1:]

	// checkArgs checks there are enough arguments and prints a message if not
	checkArgs := func(n int) {
		if len(args) != n {
			fatalf("%d arguments required for %q\n", n, command)
		}
	}

	var fn func()

	switch command {
	case "list":
		checkArgs(0)
		fn = listSnapshots
	case "download":
		checkArgs(1)
		fn = func() {
			downloadSnaphot(args[0])
		}
	case "upload":
		checkArgs(2)
		fn = func() {
			uploadSnaphot(args[0], args[1])
		}
	case "delete":
		checkArgs(1)
		fn = func() {
			deleteSnaphot(args[0])
		}
	case "types":
		checkArgs(0)
		fn = func() {
			listSnapshotTypes(os.Stdout)
		}
	default:
		fatalf("Command %q not understood", command)
	}

	// Create a v1 auth connection
	c.UserName = *userName
	c.ApiKey = *apiKey
	c.AuthUrl = *authUrl

	// Authenticate
	err := c.Authenticate()
	if err != nil {
		log.Fatalf("Failed to log in to Memstore: %v", err)
	}

	// Run the command
	fn()
}
