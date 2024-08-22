package main

import (
	"embed"

	"code.crute.us/mcrute/netboot-server/cmd"
	_ "code.crute.us/mcrute/netboot-server/netboxconfig/plugins"
)

// This is the most minimal possible main.go file to allow for easy
// extension in adding additional plugins that are out of tree. Just
// import them in this file and rebuild.

//go:embed tftpboot
var tftpboot embed.FS

//go:embed boot.ipxe.tpl
var ipxeTemplate string

func main() {
	cmd.CmdMain(tftpboot, ipxeTemplate)
}
