package main

import (
	"fmt"
	"github.com/kairos-io/netboot/booters"
	"github.com/kairos-io/netboot/server"
	"github.com/kairos-io/netboot/types"
)

// This runs a quick server that serves an efi file directly, good for UKI.
func main() {
	ret := &server.Server{
		Log:        func(subsystem, msg string) { fmt.Printf("subsystem: %s, msg: %s\n", subsystem, msg) },
		Debug:      func(subsystem, msg string) { fmt.Printf("subsystem: %s, msg: %s\n", subsystem, msg) },
		DHCPNoBind: true,
	}

	ret.SetDefaultFirmwares()
	booterSpec := &types.Spec{
		Efi: types.ID("https://boot.netboot.xyz/ipxe/netboot.xyz.efi"),
	}
	b, _ := booters.StaticBooter(booterSpec)
	ret.Booter = b
	err := ret.Serve()
	if err != nil {
		fmt.Println(err)
	}
}
