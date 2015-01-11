// Memset snapshot manager

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/memset/snapshot-manager/snapshot"
	"github.com/ncw/swift"
)

// Globals
var (
	// Flags
	chunkSize = flag.Int("s", 64*1024*1024, "Size of the chunks to make")
	userName  = flag.String("user", "", "Memstore user name, eg myaccaa1.admin")
	apiKey    = flag.String("api-key", "", "Memstore api key")
	authUrl   = flag.String("auth-url", "https://auth.storage.memset.com/v1.0", "Swift Auth URL - default should be OK for Memstore")
	// Snapshot manager
	sm *snapshot.Manager
)

// List the snapshots available
func listSnapshots() {
	snapshots, err := sm.List()
	if err != nil {
		log.Fatalf("List failed: %v", err)
	}
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
	s, err := sm.ReadSnapshot(name)
	if err != nil {
		log.Fatalf("Failed to read snapshot: %v", err)
	}
	err = s.Get(name)
	if err != nil {
		log.Fatalf("Failed to get snapshot: %v", err)
	}
}

// Upload a snapshot
func uploadSnaphot(name, file string) {
	s := sm.NewSnapshotForUpload(name, file)
	log.Printf("Uploading snapshot")
	err := s.Put(file)
	if err != nil {
		log.Fatalf("Failed to upload snapshot: %v", err)
	}
}

// Delete a snapshot
func deleteSnaphot(name string) {
	s, err := sm.ReadSnapshot(name)
	if err != nil {
		log.Fatalf("Failed to read snapshot: %v", err)
	}
	err = s.Delete()
	if err != nil {
		log.Fatalf("Failed to delete snapshot: %v", err)
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
			snapshot.Types.List(os.Stdout)
		}
	default:
		fatalf("Command %q not understood", command)
	}

	// Create a v1 auth connection
	c := swift.Connection{
		UserName: *userName,
		ApiKey:   *apiKey,
		AuthUrl:  *authUrl,
	}

	// Authenticate
	err := c.Authenticate()
	if err != nil {
		log.Fatalf("Failed to log in to Memstore: %v", err)
	}

	// Create the manager
	sm = &snapshot.Manager{
		Swift:     &c,
		ChunkSize: *chunkSize,
	}
	sm.Init()

	// Run the command
	fn()
}
