// This code is a somewhat simplified and changed up version of
// github.com/canonical/go-efilib credits go their authors.
// It allows interaction with the efivarfs via u-root which means
// both read and write support.

package efivarfs

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"syscall"

	guid "github.com/google/uuid"
	"golang.org/x/sys/unix"
)

// EfiVarFs is the path to the efivarfs mount point
var EfiVarFs = "/sys/firmware/efi/efivars/"

type backend interface {
	// Get reads the contents of an efivar if it exists and has the necessary permission
	Get(name string, guid *guid.UUID) (VariableAttributes, []byte, error)
	// Set modifies a given efivar with the provided contents
	Set(name string, guid *guid.UUID, attrs VariableAttributes, data []byte) error
	// Remove makes the specified EFI var mutable and then deletes it
	Remove(name string, guid *guid.UUID) error
	// List returns the VariableDescriptor for each efivar in the system
	List() ([]VariableDescriptor, error)
}

type varfs struct{}

var vars backend = varfs{}

// Get reads the contents of an efivar if it exists and has the necessary permission
func (v varfs) Get(name string, guid *guid.UUID) (VariableAttributes, []byte, error) {
	// Check if there is an efivarfs present
	if !probeEfivarfs(EfiVarFs) {
		return 0, nil, ErrFsNotMounted
	}

	path := filepath.Join(EfiVarFs, fmt.Sprintf("%s-%s", name, guid.String()))
	f, err := openFile(path, os.O_RDONLY, 0)
	switch {
	case os.IsNotExist(err):
		return 0, nil, ErrVarNotExist
	case os.IsPermission(err):
		return 0, nil, ErrVarPermission
	case err != nil:
		return 0, nil, err
	}
	defer f.Close()

	var attrs VariableAttributes
	if err := binary.Read(f, binary.LittleEndian, &attrs); err != nil {
		if err == io.EOF {
			return 0, nil, ErrVarNotExist
		}
		return 0, nil, err
	}

	data, err := io.ReadAll(f)
	if err != nil {
		return 0, nil, err
	}
	return attrs, data, nil
}

// Set modifies a given efivar with the provided contents
func (v varfs) Set(name string, guid *guid.UUID, attrs VariableAttributes, data []byte) error {
	// Check if there is an efivarfs present
	if !probeEfivarfs(EfiVarFs) {
		return ErrFsNotMounted
	}

	path := filepath.Join(EfiVarFs, fmt.Sprintf("%s-%s", name, guid.String()))
	flags := os.O_WRONLY | os.O_CREATE
	if attrs&AttributeAppendWrite != 0 {
		flags |= os.O_APPEND
	}

	read, err := openFile(path, os.O_RDONLY, 0)
	switch {
	case os.IsNotExist(err):
	case os.IsPermission(err):
		return ErrVarPermission
	case err != nil:
		return err
	default:
		defer read.Close()

		restoreImmutable, err := makeVarFileMutable(read)
		switch {
		case os.IsPermission(err):
			return ErrVarPermission
		case err != nil:
			return err
		}
		defer restoreImmutable()
	}

	write, err := openFile(path, flags, 0644)
	switch {
	case os.IsPermission(err):
		pe, ok := err.(*os.PathError)
		if !ok {
			return err
		}
		if pe.Err == syscall.EACCES {
			// open will fail with EACCES if we lack the privileges
			// to write to the file or the parent directory in the
			// case where we need to create a new file. Don't retry
			// in this case.
			return ErrVarPermission
		}

		// open will fail with EPERM if the file exists but we can't
		// write to it because it is immutable. This might happen as a
		// result of a race with another process that might have been
		// writing to the variable or may have deleted and recreated
		// it, making the underlying inode immutable again. Retry in
		// this case.
		return ErrVarRetry
	case err != nil:
		return err
	}
	defer write.Close()

	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, attrs); err != nil {
		return err
	}
	for len(data)%8 != 0 {
		data = append(data, 0)
	}
	size, err := buf.Write(data)
	if err != nil {
		return err
	}
	if (size-4)%8 == 0 {
		return fmt.Errorf("data misaligned")
	}
	if _, err := buf.WriteTo(write); err != nil {
		return err
	}
	return nil
}

// Remove makes the specified EFI var mutable and then deletes it
func (v varfs) Remove(name string, guid *guid.UUID) error {
	// Check if there is an efivarfs present
	if !probeEfivarfs(EfiVarFs) {
		return ErrFsNotMounted
	}

	path := filepath.Join(EfiVarFs, fmt.Sprintf("%s-%s", name, guid.String()))
	f, err := openFile(path, os.O_WRONLY, 0)
	switch {
	case os.IsNotExist(err):
		return ErrVarNotExist
	case os.IsPermission(err):
		return ErrVarPermission
	case err != nil:
		return err
	default:
		_, err := makeVarFileMutable(f)
		switch {
		case os.IsPermission(err):
			return ErrVarPermission
		case err != nil:
			return err
		default:
			f.Close()
		}
	}
	return os.Remove(path)
}

// List returns the VariableDescriptor for each efivar in the system
func (v varfs) List() ([]VariableDescriptor, error) {
	// Check if there is an efivarfs present
	if !probeEfivarfs(EfiVarFs) {
		return nil, ErrFsNotMounted
	}

	const guidLength = 36
	f, err := openFile(EfiVarFs, os.O_RDONLY, 0)
	switch {
	case os.IsNotExist(err):
		return nil, ErrVarNotExist
	case os.IsPermission(err):
		return nil, ErrVarPermission
	case err != nil:
		return nil, err
	}
	defer f.Close()

	dirents, err := f.Readdir(-1)
	if err != nil {
		return nil, err
	}
	var entries []VariableDescriptor
	for _, dirent := range dirents {
		if !dirent.Mode().IsRegular() {
			// Skip non-regular files
			continue
		}
		if len(dirent.Name()) < guidLength+1 {
			// Skip files with a basename that isn't long enough
			// to contain a GUID and a hyphen
			continue
		}
		if dirent.Name()[len(dirent.Name())-guidLength-1] != '-' {
			// Skip files where the basename doesn't contain a
			// hyphen between the name and GUID
			continue
		}
		if dirent.Size() == 0 {
			// Skip files with zero size. These are variables that
			// have been deleted by writing an empty payload
			continue
		}

		name := dirent.Name()[:len(dirent.Name())-guidLength-1]
		guid, err := guid.Parse(dirent.Name()[len(name)+1:])
		if err != nil {
			continue
		}

		entries = append(entries, VariableDescriptor{Name: name, GUID: &guid})
	}

	sort.Slice(entries, func(i, j int) bool {
		return fmt.Sprintf("%s-%v", entries[i].Name, entries[i].GUID) < fmt.Sprintf("%s-%v", entries[j].Name, entries[j].GUID)
	})
	return entries, nil
}

func probeEfivarfs(path string) bool {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return false
	}
	if uint(stat.Type) != uint(unix.EFIVARFS_MAGIC) {
		return false
	}
	return true
}
