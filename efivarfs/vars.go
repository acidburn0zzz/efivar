package efivarfs

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"

	guid "github.com/google/uuid"
	"golang.org/x/sys/unix"
)

// VariableAttributes uint32 identifying the variables attributes
type VariableAttributes uint32

const (
	// Variable is non volatile
	AttributeNonVolatile VariableAttributes = 0x00000001
	// Variable is accessible during boot service
	AttributeBootserviceAccess VariableAttributes = 0x00000002
	//Variable is accessible during runtime
	AttributeRuntimeAccess VariableAttributes = 0x00000004
	// Variable holds hardware error records
	AttributeHardwareErrorRecord VariableAttributes = 0x00000008
	// Variable needs authentication before write access
	AttributeAuthenticatedWriteAccess VariableAttributes = 0x00000010
	// Variable needs time based authentication before write access
	AttributeTimeBasedAuthenticatedWriteAccess VariableAttributes = 0x00000020
	// Data written to this variable is appended
	AttributeAppendWrite VariableAttributes = 0x00000040
	// Variable uses the new authentication format
	AttributeEnhancedAuthenticatedAccess VariableAttributes = 0x00000080
)

var (
	// ErrFsNotMounted is caused if no vailed efivarfs magic is found
	ErrFsNotMounted = errors.New("no efivarfs magic found, is it mounted?")

	// ErrVarsUnavailable is caused by not having a valid backend
	ErrVarsUnavailable = errors.New("no variable backend is available")

	// ErrVarNotExist is caused by accessing a non-existing variable
	ErrVarNotExist = errors.New("variable does not exist")

	// ErrVarPermission is caused by not haven the right permissions either
	// because of not being root or xattrs not allowing changes
	ErrVarPermission = errors.New("permission denied")

	// ErrVarRetry is caused if the previous action failed under a condition
	// that indicates a retry might be necessary to fullfill the action
	ErrVarRetry = errors.New("retry needed")
)

// VariableDescriptor contains the name and GUID identifying a variable
type VariableDescriptor struct {
	Name string
	GUID *guid.UUID
}

// File represents a file inside the efivarfs
type File interface {
	io.ReadWriteCloser

	// Readdir is analog to fs.ReadDir
	Readdir(n int) ([]os.FileInfo, error)

	// GetInodeFlags returns the extended attributes of a file
	GetInodeFlags() (int, error)

	// SetInodeFlags sets the extended attributes of a file
	SetInodeFlags(flags int) error
}

type file struct {
	*os.File
}

// ReadVariable calls Get() on the current efivarfs backend
func ReadVariable(name string, guid *guid.UUID) (VariableAttributes, []byte, error) {
	e, err := ProbeAndReturn()
	if err != nil {
		return 0, nil, err
	}
	return e.Get(name, guid)
}

// SimpleReadVariable is like ReadVariables but takes the combined name and guid string
// of the form name-guid and returns a bytes.Reader instead of a []byte
func SimpleReadVariable(v string) (VariableAttributes, *bytes.Reader, error) {
	e, err := ProbeAndReturn()
	if err != nil {
		return 0, nil, err
	}
	vs := strings.SplitN(v, "-", 2)
	g, err := guid.Parse(vs[1])
	if err != nil {
		return 0, nil, err
	}
	attrs, data, err := e.Get(vs[0], &g)
	return attrs, bytes.NewReader(data), err
}

// WriteVariable calls Set() on the current efivarfs backend
func WriteVariable(name string, guid *guid.UUID, attrs VariableAttributes, data []byte) error {
	e, err := ProbeAndReturn()
	if err != nil {
		return err
	}
	return maybeRetry(4, func() error { return e.Set(name, guid, attrs, data) })
}

// SimpleWriteVariable is like WriteVariables but takes the combined name and guid string
// of the form name-guid and returns a bytes.Buffer instead of a []byte
func SimpleWriteVariable(v string, attrs VariableAttributes, data bytes.Buffer) error {
	e, err := ProbeAndReturn()
	if err != nil {
		return err
	}
	vs := strings.SplitN(v, "-", 2)
	g, err := guid.Parse(vs[1])
	if err != nil {
		return err
	}
	return e.Set(vs[0], &g, attrs, data.Bytes())
}

// RemoveVariable calls Remove() on the current efivarfs backend
func RemoveVariable(name string, guid *guid.UUID) error {
	e, err := ProbeAndReturn()
	if err != nil {
		return err
	}
	return e.Remove(name, guid)
}

// SimpleRemoveVariable is like RemoveVariable but takes the combined name and guid string
// of the form name-guid
func SimpleRemoveVariable(v string) error {
	e, err := ProbeAndReturn()
	if err != nil {
		return err
	}
	vs := strings.SplitN(v, "-", 2)
	g, err := guid.Parse(vs[1])
	if err != nil {
		return err
	}
	return e.Remove(vs[0], &g)
}

// ListVariables calls List() on the current efivarfs backend
func ListVariables() ([]VariableDescriptor, error) {
	e, err := ProbeAndReturn()
	if err != nil {
		return nil, err
	}
	return e.List()
}

// SimpleListVariables is like ListVariables but returns a []string instead of a []VariableDescriptor
func SimpleListVariables() ([]string, error) {
	e, err := ProbeAndReturn()
	if err != nil {
		return nil, err
	}
	list, err := e.List()
	if err != nil {
		return nil, err
	}
	var out []string
	for _, v := range list {
		out = append(out, v.Name+"-"+v.GUID.String())
	}
	return out, nil
}

func openFile(path string, flags int, perm os.FileMode) (File, error) {
	f, err := os.OpenFile(path, flags, perm)
	if err != nil {
		return nil, err
	}
	return &file{f}, nil
}

// GetInodeFlags returns the extended attributes of a file
func (f *file) GetInodeFlags() (int, error) {
	// If I knew how unix.Getxattr works I'd use that...
	flags, err := unix.IoctlGetInt(int(f.Fd()), unix.FS_IOC_GETFLAGS)
	if err != nil {
		return 0, &os.PathError{Op: "ioctl", Path: f.Name(), Err: err}
	}
	return flags, nil
}

// SetInodeFlags sets the extended attributes of a file
func (f *file) SetInodeFlags(flags int) error {
	// If I knew how unix.Setxattr works I'd use that...
	if err := unix.IoctlSetPointerInt(int(f.Fd()), unix.FS_IOC_SETFLAGS, flags); err != nil {
		return &os.PathError{Op: "ioctl", Path: f.Name(), Err: err}
	}
	return nil
}

func makeVarFileMutable(f File) (restore func(), err error) {
	flags, err := f.GetInodeFlags()
	if err != nil {
		return nil, err
	}
	if flags&unix.STATX_ATTR_IMMUTABLE == 0 {
		return func() {}, nil
	}

	if err := f.SetInodeFlags(flags &^ unix.STATX_ATTR_IMMUTABLE); err != nil {
		return nil, err
	}
	return func() {
		if err := f.SetInodeFlags(flags); err != nil {
			// If setting the immutable did
			// not work it's alright to do nothing
			// because after a reboot the flag is
			// automatically reapplied
			return
		}
	}, nil
}

func maybeRetry(n int, fn func() error) error {
	for i := 1; ; i++ {
		err := fn()
		switch {
		case i > n:
			return err
		case err != ErrVarRetry:
			return err
		case err == nil:
			return nil
		}
	}
}
