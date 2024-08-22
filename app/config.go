package app

import "strconv"

// These variables get set at build time from the Makefile and exist
// to make it easier to open-source this code without passing out a
// bunch of internal information about the backup system.
var (
	defaultNetboxHost      string
	defaultHttpServer      string
	defaultVaultNetboxPath string
	defaultNetboxConfigId  string
)

func mustAtoi(s string) int {
	o, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return o
}

type Config struct {
	Debug                 bool   `flag:"debug" flag-help:"Enable debug mode"`
	BindHttp              string `flag:"bind-http" flag-help:"Address and port to bind http server"`
	BindTftp              string `flag:"bind-tftp" flag-help:"Address and port to bind tftp server"`
	NetboxHost            string `flag:"netbox-host" flag-help:"Full URL to Netbox"`
	DistroFilesPath       string `flag:"distro-files" flag-help:"Path to distribution file tree"`
	NtpServer             string `flag:"ntp-server" flag-help:"Address of NTP server"`
	HttpServer            string `flag:"http-server" flag-help:"HTTP/S URL to this server"`
	VarsConfigFile        string `flag:"vars-config" flag-help:"Path to variables config file"`
	VaultNetboxPath       string `flag:"vault-netbox-path" flag-help:"Path in Vault KV store for Netbox credential"`
	NetboxDefaultConfigId int    `flag:"default-config-id" flag-help:"ID for default config context"`
}

var DefaultConfig = &Config{
	Debug:                 false,
	BindHttp:              ":80",
	BindTftp:              ":69",
	NetboxHost:            defaultNetboxHost,
	DistroFilesPath:       "./html",
	NtpServer:             "0.pool.ntp.org",
	HttpServer:            defaultHttpServer,
	VarsConfigFile:        "./html/vars.yaml",
	VaultNetboxPath:       defaultVaultNetboxPath,
	NetboxDefaultConfigId: mustAtoi(defaultNetboxConfigId),
}
