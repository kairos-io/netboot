package constants

// Firmware describes a kind of firmware attempting to boot.
//
// This should only be used for selecting the right bootloader within
// Pixiecore, kernel selection should key off the more generic
// Architecture.
type Firmware int

// The bootloaders that Pixiecore knows how to handle.
const (
	FirmwareX86PC         Firmware = iota // "Classic" x86 BIOS with PXE/UNDI support
	FirmwareEFI32                         // 32-bit x86 processor running EFI
	FirmwareEFI64                         // 64-bit x86 processor running EFI
	FirmwareEFIBC                         // 64-bit x86 processor running EFI
	FirmwareX86Ipxe                       // "Classic" x86 BIOS running iPXE (no UNDI support)
	FirmwarePixiecoreIpxe                 // Pixiecore's iPXE, which has replaced the underlying firmware
	FirmwareEfiArm64                      // 64-bit ARM processor running EFI
)

// Architecture describes a kind of CPU architecture.
type Architecture int

// Architecture types that Pixiecore knows how to boot.
//
// These architectures are self-reported by the booting machine. The
// machine may support additional execution modes. For example, legacy
// PC BIOS reports itself as an ArchIA32, but may also support ArchX64
// execution.
const (
	// ArchIA32 is a 32-bit x86 machine. It _may_ also support X64
	// execution, but Pixiecore has no way of knowing.
	ArchIA32 Architecture = iota
	// ArchX64 is a 64-bit x86 machine (aka amd64 aka X64).
	ArchX64
	ArchArm64
)

const X86HTTPClient = 0x10

const (
	PortDHCP   = 67
	PortDHCPv6 = 547
	PortTFTP   = 69
	PortHTTP   = 80
	PortPXE    = 4011
)
