// Copyright 2024 Kairos contributors

package types

import (
	"io"
	"net"
	"time"

	"github.com/kairos-io/netboot/constants"
)

// A Booter provides boot instructions and files for machines.
//
// Due to the stateless nature of various boot protocols, BootSpec()
// will be called multiple times in the course of a single boot
// attempt.
type Booter interface {
	// BootSpec The given MAC address wants to know what it should
	// boot. What should Pixiecore make it boot?
	//
	// Returning an error or a nil BootSpec will make Pixiecore ignore
	// the client machine's request.
	BootSpec(m Machine) (*Spec, error)
	// ReadBootFile Get the bytes corresponding to an ID given in Spec.
	//
	// Additionally returns the total number of bytes in the
	// ReadCloser, or -1 if the size is unknown. Be warned, returning
	// -1 will make the boot process orders of magnitude slower due to
	// poor ipxe behavior.
	ReadBootFile(id ID) (io.ReadCloser, int64, error)
	// WriteBootFile Write the given Reader to an ID given in Spec.
	WriteBootFile(id ID, body io.Reader) error
}

// An ID is an identifier used by Booters to reference files.
type ID string

// A Machine describes a machine that is attempting to boot.
type Machine struct {
	MAC  net.HardwareAddr
	Arch constants.Architecture
}

// A Spec describes a kernel and associated configuration.
type Spec struct {
	// The kernel to boot
	Kernel ID
	// Optional init ramdisks for linux kernels
	Initrd []ID

	// Optional efi binary to boot
	// Either Efi or Kernel must be set
	Efi ID
	// Optional kernel commandline. This string is evaluated as a
	// text/template template, in which "ID(x)" function is
	// available. Invoking ID(x) returns a URL that will call
	// Booter.ReadBootFile(x) when fetched.
	Cmdline string
	// Message to print on the client machine before booting.
	Message string

	// A raw iPXE script to run. Overrides all of the above.
	//
	// THIS IS NOT A STABLE INTERFACE. This will only work for
	// machines that get booted via iPXE. Currently, that is all of
	// them, but there is no guarantee that this will remain
	// true. When passing a custom iPXE script, it is your
	// responsibility to make the boot succeed, Pixiecore's
	// involvement ends when it serves your script.
	IpxeScript string
}

// IPV6

// IdentityAssociation associates an ip address with a network interface of a client
type IdentityAssociation struct {
	IPAddress   net.IP
	ClientID    []byte
	InterfaceID []byte
	CreatedAt   time.Time
}

// AddressPool keeps track of assigned and available ip address in an address pool
type AddressPool interface {
	ReserveAddresses(clientID []byte, interfaceIds [][]byte) ([]*IdentityAssociation, error)
	ReleaseAddresses(clientID []byte, interfaceIds [][]byte)
}
