package main

import (
	"fmt"
	"go.universe.tf/netboot/booters"
	"go.universe.tf/netboot/server"
	"go.universe.tf/netboot/types"
)

// This runs a quick server that serves iPXE and Debian netboot files.
func main() {
	version := "stable"
	arch := "amd64"
	mirror := "http://deb.debian.org/debian"

	kernel := fmt.Sprintf("%s/dists/%s/main/installer-%s/current/images/netboot/debian-installer/%s/linux", mirror, version, arch, arch)
	initrd := fmt.Sprintf("%s/dists/%s/main/installer-%s/current/images/netboot/debian-installer/%s/initrd.gz", mirror, version, arch, arch)

	ret := &server.Server{
		Log:        func(subsystem, msg string) { fmt.Printf("subsystem: %s, msg: %s\n", subsystem, msg) },
		Debug:      func(subsystem, msg string) { fmt.Printf("subsystem: %s, msg: %s\n", subsystem, msg) },
		DHCPNoBind: true,
	}

	ret.SetDefaultFirmwares()
	booterSpec := &types.Spec{
		Kernel:  types.ID(kernel),
		Cmdline: "",
		Initrd:  []types.ID{types.ID(initrd)},
	}
	b, _ := booters.StaticBooter(booterSpec)
	ret.Booter = b
	err := ret.Serve()
	if err != nil {
		fmt.Println(err)
	}
}
