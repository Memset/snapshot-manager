// Memset snapshot manager

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/user"
	"path"
	"runtime"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/memset/snapshot-manager/snapshot"
	"github.com/ncw/swift"
)

const (
	configFileName   = ".snapshot-manager.conf"
	chunkSizeDefault = 64 * 1024 * 1024
)

// Globals
var (
	// Home directory
	homeDir = configHome()
	// Default config file path
	defaultConfigPath = path.Join(homeDir, configFileName)
	// Config file
	configFile string
	// Snapshot manager
	sm *snapshot.Manager
)

var Config, flagsConfig struct {
	User      string
	Password  string
	AuthUrl   string
	ChunkSize int
}

// Flags
func init() {
	Config.ChunkSize = chunkSizeDefault
	flag.StringVar(&configFile, "config", defaultConfigPath, "Path to config file")
	flag.IntVar(&flagsConfig.ChunkSize, "chunk-size", chunkSizeDefault, "Size of the chunks to make")
	flag.StringVar(&flagsConfig.User, "user", "", "Memstore user name, eg myaccaa1.admin")
	flag.StringVar(&flagsConfig.Password, "password", "", "Memstore password")
	flag.StringVar(&flagsConfig.AuthUrl, "auth-url", "https://auth.storage.memset.com/v1.0", "Swift Auth URL - default is for Memstore")
}

// Override the config file with the flags
func overrideConfigFileWithFlags() {
	if flagsConfig.User != "" {
		Config.User = flagsConfig.User
	}
	if flagsConfig.Password != "" {
		Config.Password = flagsConfig.Password
	}
	if flagsConfig.AuthUrl != "" {
		Config.AuthUrl = flagsConfig.AuthUrl
	}
	if flagsConfig.ChunkSize != chunkSizeDefault {
		Config.ChunkSize = flagsConfig.ChunkSize
	}
}

// Find the config directory
func configHome() string {
	// Find users home directory
	usr, err := user.Current()
	if err == nil {
		return usr.HomeDir
	}
	// Fall back to reading $HOME - work around user.Current() not
	// working for cross compiled binaries on OSX.
	// https://github.com/golang/go/issues/6376
	home := os.Getenv("HOME")
	if home != "" {
		return home
	}
	log.Printf("Couldn't find home directory or read HOME environment variable.")
	log.Printf("Defaulting to storing config in current directory.")
	log.Printf("Use -config flag to workaround.")
	log.Printf("Error was: %v", err)
	return ""
}

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

	// Allow all the processors
	runtime.GOMAXPROCS(runtime.NumCPU())

	fi, err := os.Stat(configFile)
	if err == nil && fi.Mode().IsRegular() {
		_, err := toml.DecodeFile(configFile, &Config)
		if err != nil {
			log.Fatalf("Bad config file %q: %v", configFile, err)
		}
	}
	overrideConfigFileWithFlags()

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

	needsConnection := true
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
		needsConnection = false
		fn = func() {
			snapshot.Types.List(os.Stdout)
		}
	default:
		needsConnection = false
		fatalf("Command %q not understood", command)
	}

	// Create a v1 auth connection
	c := swift.Connection{
		UserName: Config.User,
		ApiKey:   Config.Password,
		AuthUrl:  Config.AuthUrl,
	}

	// Check connection if required
	if needsConnection {
		if Config.User == "" || Config.Password == "" {
			fatalf(`Flags -user and -password required or config file entries "user" and "password"`)
		}

		// Authenticate
		err = c.Authenticate()
		if err != nil {
			log.Fatalf("Failed to log in to Memstore: %v", err)
		}
	}

	// Create the manager
	sm = &snapshot.Manager{
		Swift:     &c,
		ChunkSize: Config.ChunkSize,
	}
	sm.Init()

	// Run the command
	fn()
}
