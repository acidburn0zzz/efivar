// This is a small example implementation of a tool that
// uses the efivarfs package in this repo.
// The file referenced by -content was tested with different
// amounts of text data in it only.

package main

import (
	"bytes"
	"flag"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/system-transparency/efivar/efivarfs"
)

var (
	list    = flag.Bool("list", false, "List all efivars")
	read    = flag.String("read", "", "Read specified efivar")
	delete  = flag.String("delete", "", "Delete specified var")
	write   = flag.String("write", "", "Write to specified efivar")
	content = flag.String("content", "", "Path to file to write to efivar")
)

func main() {
	flag.Parse()

	if *list {
		l, err := efivarfs.SimpleListVariables()
		if err != nil {
			log.Fatalf("List failed: %v", err)
		}
		for _, s := range l {
			log.Println(s)
		}
	}

	if *read != "" {
		attr, data, err := efivarfs.SimpleReadVariable(*read)
		if err != nil {
			log.Fatalf("Read failed: %v", err)
		}
		b := make([]byte, data.Len())
		_, err = data.Read(b)
		if err != nil {
			log.Fatalf("Reading buffer failed: %v", err)
		}
		log.Printf("Name: %s, Attributes: %d, Data: %s", *read, attr, b)
	}

	if *delete != "" {
		err := efivarfs.SimpleDeleteVariable(*delete)
		if err != nil {
			log.Fatalf("Delete failed: %v", err)
		}
	}

	if *write != "" {
		if strings.ContainsAny(*write, "-") {
			v := strings.SplitN(*write, "-", 2)
			_, err := efivarfs.DecodeGUIDString(v[1])
			if err != nil {
				log.Fatal("Var name malformed: Must be either Name-GUID or just Name")
			}
		}
		path, err := filepath.Abs(*content)
		if err != nil {
			log.Fatalf("Could not resolve path: %v", err)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			log.Fatalf("Failed to read file: %v", err)
		}
		if !strings.ContainsAny(*write, "-") {
			var array [6]uint8
			for i := range array {
				array[i] = uint8(rand.Uint32())
			}
			*write = *write + "-" + efivarfs.MakeGUID(rand.Uint32(), uint16(rand.Uint32()), uint16(rand.Uint32()), uint16(rand.Uint32()), array).String()
		}
		err = efivarfs.SimpleWriteVariable(*write, 7, *bytes.NewBuffer(b))
		if err != nil {
			log.Fatalf("Write failed: %v", err)
		}
	}
}
