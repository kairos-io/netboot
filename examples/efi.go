package main

import (
	"fmt"
	"github.com/kairos-io/netboot/booters"
	"github.com/kairos-io/netboot/log"
	"github.com/kairos-io/netboot/server"
	"github.com/kairos-io/netboot/types"
	"github.com/rs/zerolog"
	"time"
)

// This runs a quick server that serves an efi file directly, good for UKI.
func main() {
	log.Log = zerolog.New(zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		w.TimeFormat = time.RFC3339
	})).With().Timestamp().Logger().Level(zerolog.DebugLevel)
	ret := &server.Server{
		Log:        func(subsystem, msg string) { log.Log.Info().Str("subsystem", subsystem).Msgf(msg) },
		Debug:      func(subsystem, msg string) { log.Log.Debug().Str("subsystem", subsystem).Msgf(msg) },
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
