// This is a small example implementation of a tool that
// uses the efivarfs package in this repo.
// The file referenced by -content was tested with different
// amounts of text data in it only.

package main

import (
	"bytes"
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/u-root/u-root/pkg/efivarfs"
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
		path, err := filepath.Abs(*content)
		if err != nil {
			log.Fatalf("Could not resolve path: %v", err)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			log.Fatalf("Failed to read file: %v", err)
		}
		err = efivarfs.SimpleWriteVariable(*write, 7, *bytes.NewBuffer(b))
		if err != nil {
			log.Fatalf("Write failed: %v", err)
		}
	}
}
