package snapshot

import (
	"fmt"
	"log"
	"path"
	"strings"

	"github.com/ncw/swift"
)

// Manages snapshots in the container
type Manager struct {
	Swift     *swift.Connection
	ChunkSize int
	Container string
}

// Init makes the Manager object ready, setting default items
func (sm *Manager) Init() {
	if sm.ChunkSize == 0 {
		sm.ChunkSize = 64 * 1024 * 1024
	}
	if sm.Container == "" {
		sm.Container = DefaultContainer
	}
}

// Check the Container exists
func (sm *Manager) Check() (bool, error) {
	_, _, err := sm.Swift.Container(sm.Container)
	if err == swift.ContainerNotFound {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("error for container %q: %v", sm.Container, err)
	}
	return true, nil
}

// Create the container if it doesn't exist
func (sm *Manager) CreateContainer() error {
	ok, err := sm.Check()
	if err != nil {
		return err
	}
	if !ok {
		err = sm.Swift.ContainerCreate(sm.Container, nil)
		if err != nil {
			return fmt.Errorf("failed to create container %q: %v", sm.Container, err)
		}
	}
	return nil
}

// Read the objects in the snapshot
func (sm *Manager) Objects(name string) ([]swift.Object, error) {
	objects, err := sm.Swift.Objects(sm.Container, &swift.ObjectsOpts{
		Prefix:    name + "/",
		Delimiter: '/',
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read snapshot %q: %v", name, err)
	}
	return objects, nil
}

// ReadSnapshot gets info about snapshot from container
func (sm *Manager) ReadSnapshot(name string) (*Snapshot, error) {
	s := &Snapshot{
		Manager: sm,
		Name:    name,
	}

	objects, err := sm.Objects(name)
	if err != nil {
		return nil, err
	}

	// check for README.txt for the user comment
	for _, object := range objects {
		if strings.HasSuffix(object.Name, "README.txt") {
			readme, err := sm.Swift.ObjectGetString(sm.Container, object.Name)
			if err != nil {
				log.Printf("Couldn't read %q - ignoring: %v", object.Name, err)
				continue
			}
			s.ParseReadme(readme)
		}
	}

	// we could get these from the README.txt, but currently this is
	// easier/more reliable than parsing the .txt file
	for _, object := range objects {
		Type := Types.Find(object.Name)
		if Type != nil {
			s.Path = object.Name
			if s.Date.IsZero() {
				s.Date = object.LastModified
			}
			break
		}
	}

	// it might be a broken or active snapshot
	if s.Path == "" {
		s.Broken = true
		s.Comment = "The snapshot probably failed and some files were left behind."
	}

	return s, nil
}

// NewSnapshot makes an empty snapshot from a name
func (sm *Manager) NewSnapshot(name string) *Snapshot {
	return &Snapshot{
		Manager: sm,
		Name:    name,
	}
}

// NewSnapshot makes an empty snapshot from a name and a file
func (sm *Manager) NewSnapshotForUpload(name, file string) *Snapshot {
	s := sm.NewSnapshot(name)
	leaf := strings.ToLower(path.Base(file))
	Path := name + "/" + leaf

	s.Path = Path
	s.Comment = fmt.Sprintf("Uploaded from original file '%s'", file)
	s.Broken = false
	s.ImageLeaf = leaf
	s.Miniserver = "uploaded"
	return s
}

// List all snapshots in the container
func (sm *Manager) List() ([]*Snapshot, error) {
	ok, err := sm.Check()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	objects, err := sm.Swift.Objects(sm.Container, &swift.ObjectsOpts{
		Prefix:    "",
		Delimiter: '/',
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots: %v", err)
	}
	if len(objects) == 0 {
		return nil, nil
	}
	var snapshots []*Snapshot
	for _, obj := range objects {
		if obj.PseudoDirectory {
			name := strings.TrimRight(obj.Name, "/")
			s, err := sm.ReadSnapshot(name)
			if err != nil {
				return nil, err
			}
			snapshots = append(snapshots, s)
		}
	}
	return snapshots, nil
}
