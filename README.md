# efivar
Temporary repository until merged into u-root

## Prerequisites
The example implementation can be used to list, read and write all
existing efivars, for that the efivarfs needs to be mounted:

```
mount -t efivarfs efivarfs /sys/firmware/efi/efivars
```

## Building
This example tool can either be compiled standalone using 
`go build` or included into u-root using the path to this repo
as an parameter when generating the initramfs as described in
the u-root README.

## Usage
Running `efivar -help` already reveals the available options.
The format needed wenn reading or writing to a var is the same
that `-list` returns, so Name-GUID. If write is called on a not
yet existing variable, it is being created. The data that is supposed
to be written should be specified using `-content` and so far it as
been verified to work using a 16KiB big random textfile but in theory
every decently sized file should be usable.
