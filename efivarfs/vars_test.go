package efivarfs

import (
	"bytes"
	"testing"

	guid "github.com/google/uuid"
)

func TestReadVariable(t *testing.T) {
	for _, tt := range []struct {
		name    string
		vd      VariableDescriptor
		wantErr error
	}{
		{
			name: "no efivarfs",
			vd: VariableDescriptor{
				Name: "TestVar",
				GUID: func() *guid.UUID {
					g := guid.MustParse("bc54d3fb-ed45-462d-9df8-b9f736228350")
					return &g
				}(),
			},
			wantErr: ErrFsNotMounted,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, err := ReadVariable(tt.vd.Name, tt.vd.GUID); err != tt.wantErr {
				t.Errorf("Want: %v, Got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestSimpleReadVariable(t *testing.T) {
	for _, tt := range []struct {
		name    string
		varName string
		wantErr error
	}{
		{
			name:    "no efivarfs",
			varName: "TestVar-bc54d3fb-ed45-462d-9df8-b9f736228350",
			wantErr: ErrFsNotMounted,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, err := SimpleReadVariable(tt.varName); err != tt.wantErr {
				t.Errorf("Want: %v, Got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestWriteVariable(t *testing.T) {
	for _, tt := range []struct {
		name    string
		vd      VariableDescriptor
		attrs   VariableAttributes
		data    []byte
		wantErr error
	}{
		{
			name: "no efivarfs",
			vd: VariableDescriptor{
				Name: "TestVar",
				GUID: func() *guid.UUID {
					g := guid.MustParse("bc54d3fb-ed45-462d-9df8-b9f736228350")
					return &g
				}(),
			},
			attrs:   0,
			data:    nil,
			wantErr: ErrFsNotMounted,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := WriteVariable(tt.vd.Name, tt.vd.GUID, tt.attrs, tt.data); err != tt.wantErr {
				t.Errorf("Want: %v, Got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestSimpleWriteVariable(t *testing.T) {
	for _, tt := range []struct {
		name    string
		varName string
		attrs   VariableAttributes
		data    bytes.Buffer
		wantErr error
	}{
		{
			name:    "no efivarfs",
			varName: "TestVar-bc54d3fb-ed45-462d-9df8-b9f736228350",
			attrs:   0,
			data:    bytes.Buffer{},
			wantErr: ErrFsNotMounted,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := SimpleWriteVariable(tt.varName, tt.attrs, tt.data); err != tt.wantErr {
				t.Errorf("Want: %v, Got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestRemoveVariable(t *testing.T) {
	for _, tt := range []struct {
		name    string
		vd      VariableDescriptor
		wantErr error
	}{
		{
			name: "no efivarfs",
			vd: VariableDescriptor{
				Name: "TestVar",
				GUID: func() *guid.UUID {
					g := guid.MustParse("bc54d3fb-ed45-462d-9df8-b9f736228350")
					return &g
				}(),
			},
			wantErr: ErrFsNotMounted,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := RemoveVariable(tt.vd.Name, tt.vd.GUID); err != tt.wantErr {
				t.Errorf("Want: %v, Got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestSimpleRemoveVariable(t *testing.T) {
	for _, tt := range []struct {
		name    string
		varName string
		wantErr error
	}{
		{
			name:    "no efivarfs",
			varName: "TestVar-bc54d3fb-ed45-462d-9df8-b9f736228350",
			wantErr: ErrFsNotMounted,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := SimpleRemoveVariable(tt.varName); err != tt.wantErr {
				t.Errorf("Want: %v, Got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestListVariable(t *testing.T) {
	for _, tt := range []struct {
		name    string
		vd      []VariableDescriptor
		wantErr error
	}{
		{
			name: "no efivarfs",
			vd: []VariableDescriptor{
				{
					Name: "TestVar",
					GUID: func() *guid.UUID {
						g := guid.MustParse("bc54d3fb-ed45-462d-9df8-b9f736228350")
						return &g
					}(),
				},
			},
			wantErr: ErrFsNotMounted,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ListVariables(); err != tt.wantErr {
				t.Errorf("Want: %v, Got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestSimpleListVariable(t *testing.T) {
	for _, tt := range []struct {
		name    string
		result  []string
		wantErr error
	}{
		{
			name: "no efivarfs",
			result: []string{
				"TestVar-bc54d3fb-ed45-462d-9df8-b9f736228350",
			},
			wantErr: ErrFsNotMounted,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := SimpleListVariables(); err != tt.wantErr {
				t.Errorf("Want: %v, Got: %v", tt.wantErr, err)
			}
		})
	}
}
