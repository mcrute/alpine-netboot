# Alpine Netboot Server

This is a combination HTTP and TFTP server that will serve any Linux
distribution but has special functionality for configuring Alpine Linux
based on config context data within a Netbox instance.

The TFTP server exists purely to bootstrap a system into an iPXE boot
loader. Once built the server embeds iPXE payloads for arm64 and x86_64
as well as the legacy undionly.pxe boot loader to support systems
running BIOS. When chain loading systems your DHCP server must be
configured to send PXE clients to the TFTP server but should send iPXE
clients to the HTTP server.

The HTTP server will serve a custom iPXE script to clients based on
a template which is built-into the binary. The daemon will walk a
directory of Linux distributions that also contain YAML formatted
configuration files. From this walk the daemon will maintain a catalog
of available Linux distributions that will be rendered as options within
the iPXE template. The daemon will re-scan the directory every hour for
new distributions or when delivered the HUP signal.

During iPXE bootstrapping the server will issue a script to the client
that will chainload to a customized script based on the client's MAC
address. This MAC address will be used to look up a device record in
Netbox. If the user then boots into an Alpine Linux distribution the
configuration context from Netbox will be used to generate an apkovl
file dynamically for the boot system using a set of plugins built into
the server.

## Distribution Catalog

The distribution catalog is configured by passing the `--distro-files`
flag to the server (default: `/netboot`). The directory must contain a
file called `vars.yaml`, which may be empty. If the vars file is missing
then the server will fail to start.

The catalog expects a specific directory hierarchy, which is:

```
/
 vars.yaml
 /<distribution-name>
  distro.yaml
  /<distribution-version>
   /<distribution-architecture>
    /...<distro files>...
```

for example:

```
/
 /alpine
  distro.yaml
  /3.20.2
   /x86_64
   /aarch64
    /initramfs-lts
    /vmlinuz-lts
 /fedora
  distro.yaml
  /37
   /x86_64
```

Provided that the distribution is configured correctly, adding a new
distribution or pruning an old one is as simple as adding or removing
the directory sub-tree from the filesystem and sending a HUP signal
to the process. While the configuration changes are atomic within the
daemon be careful of race conditions with currently booting clients
which may fail if the distribution files are removed underneath of them.
The filesystem view is not atomic.

### vars.yaml

The `vars.yaml` file exists to provide default iPXE variables as well
as product specific variable overrides. Default variables are in a YAML
map named `default_vars` and are rendered as iPXE `set` statements. For
example, the configuration:

```yaml
default_vars:
  alpine_iparg: "dhcp"
```

will render in the iPXE script as:

```ipxe
set alpine_iparg "dhcp"
```

Product specific variables are stored in a two-layer YAML map named `product_vars`. The second level of the map
is the product name as understood by iPXE `${product}`. This is rendered as a series of `iseq` and `set` statements in
the iPXE script. For example, the configuration:

```yaml
product_vars:
  SYS-5018D-FN8T:
    alpine_iparg: "dhcp:::::eth0"
```

will render in the iPXE script as:

```ipxe
iseq ${product} SYS-5018D-FN8T && set alpine_iparg "dhcp:::::eth0"
```

### distro.yaml

The `distro.yaml` file configures an entire distribution tree for
booting if this file is missing then the distribution will not be loaded
into the catalog. All fields are required unless a default is specified.
The format is:

 * `name` - the friendly name of the distribution, as rendered in iPXE menus
 * `default` (bool, default: false) - if the distribution is candidate
   for being selected as the default boot during iPXE (useful for headless
   booting)
 * `kernel` - the name of the kernel image file, this is expected to be
   consistent for all versions and architectures of a distribution
 * `initrd` - the name of the initrd file, this is expected to be
   consistent for all versions and architectures of a distribution
 * `kernel_args` - a list of key/values which support templating and
   hold the kernel command-line arguments

Kernel arguments always have a `key` field but may optionally have one
of these fields:

 * No other field, for unary kernel arguments (for example: `rhgb`
   requires no arguments)
 * `value`, a static string value that will be rendered as `key=value`
    when rendering the kernel command line.
 * `template` which is a string that can contain Go template
   replacements for rendering. The template can reference any field on the
   Distribution structure.

```go
type Distribution struct {
	ShortName    string
	Name         string
	Default      bool
	FullVersion  string
	Architecture string
	KernelName   string
	InitrdName   string
}
```

Note that iPXE variables for the format `${name}` are supported anywhere
in kernel arguments.

For example:

```
name: Alpine Linux
default: true
kernel: vmlinuz-lts
initrd: initramfs-lts
kernel_args:
- key: ip
  value: "${alpine_iparg}"
- key: apkovl
  template: "${http_server}/${net0/mac}/apkovl.tar.gz"
- key: modloop
  template: "${http_server}/{{ .DistroPath }}/modloop-lts"
- key: ixgbe.allow_unsupported_sfp
  value: "1"
- key: intel_iommu
  value: "on"
- key: iommu
  value: "pt"
```

## APKOVL Rendering

For Alpine Linux based distributions including an `apkovl`
argument to the kernel that references the server path
`/{mac_address}/apkovl.tar.gz` will cause the server to lookup the
device record from Netbox using the MAC address provided. If a device
record is returned then that is parsed as JSON and a set of plugins
built-in to the daemon are run to generate configuration files that are
packed into the OVL file and delivered to the client.

Read more about [Netbox Context
Data](https://netboxlabs.com/docs/netbox/en/stable/features/context-data
/) to gain a better understanding of the context hierarchy and rendering
works within Netbox. This daemon consumes the fully rendered context
data from Netbox.

If an APKOVL is requested but no device is found the default config will
be used to render the APKOVL. This is specified as a record ID using
the `--default-config-id` command line flag. This should be the ID of a
non-empty config context that is not targeted at any Netbox entity.

### Adding Plugins

The config context is treated as a one-level map from the perspective
of a plugin (despite the fact that it is in-fact a JSON document of
arbitary complexity). For each key in the map a plugin will be loaded
from the plugin registry and passed the raw JSON value, which it is
expected to unmarshal and handle as it will. If no plugin is found
for the key then that key is ignored; this allows mixing Alpine and
non-Alpine configuration in the context.

To add a plugin either create a new folder in the source tree under
`netboxconfig/plugins` and register the plugin or make a copy of
`main.go` and import your plugin. Make sure that the plugin calls the
correct registration API so that it can be found at runtime.

See the existing plugins for examples. The plugin API is specified in
`netboxconfig/coordinator.go`.

### Config Grouping

Netbox configuration contexts are hierarchical but values are not
additive. Keys higher up the context hierarchy will completely
overwrite keys lower in the hierarchy. Some built-in plugins handle
this by supporting groups within their configuration which are sorted
lexicographically and then merged before rendering configuration. Empty
groups are discarded, which can be useful for overriding and removing
groups.

As an example, consider the following configuration context objects:

```
{
    "name": "one",
    "alpine_repos": {
        "3.20_group": [
            "http://dl-cdn.alpinelinux.org/alpine/v3.20/main/",
            "http://dl-cdn.alpinelinux.org/alpine/v3.20/community/"
        ]
    }
}

{
    "name": "two",
    "alpine_repos": {
        "default": [
            "http://dl-cdn.alpinelinux.org/alpine/edge/main/",
            "http://dl-cdn.alpinelinux.org/alpine/edge/community/"
        ]
    }
}

{
    "name": "three",
    "alpine_repos": {
        "corp": [
            "http://example.com/alpine/corp/main/",
        ]
    }
}
```

Assuming these are stacked as two -> three -> one in the context
hierarchy, the `alpine_repos` plugin will receive and render the package
configuration as:

```
http://dl-cdn.alpinelinux.org/alpine/edge/main/
http://dl-cdn.alpinelinux.org/alpine/edge/community/
http://example.com/alpine/corp/main/
http://dl-cdn.alpinelinux.org/alpine/v3.20/main/
http://dl-cdn.alpinelinux.org/alpine/v3.20/community/
```

Introducing a fourth context with the value of:

```
{
    "name": "four",
    "alpine_repos": {
        "corp": [],
        "other": [
            "http://example.com/alpine/other-corp/main/",
        ]
    }
}
```

which results in a hierarchy of four -> two -> three -> one; the
`alpine_repos` plugin will receive and render the package configuration
as:

```
http://example.com/alpine/other-corp/main/
http://dl-cdn.alpinelinux.org/alpine/edge/main/
http://dl-cdn.alpinelinux.org/alpine/edge/community/
http://dl-cdn.alpinelinux.org/alpine/v3.20/main/
http://dl-cdn.alpinelinux.org/alpine/v3.20/community/
```

Plugins that support this grouping method are indicated in their
documentation.

### Built-in Plugins

See [PLUGINS.md](PLUGINS.md) for more details about the built-in
plugins.

## Building

This can be built pretty simply by checking out the code and running
`make`.

There is some default configuration for the command line arguments
of the application that have generic defaults in the Makefile. To
customize these export the variables before running make and your local
configuration will be embedded in the resulting binary. The variables
are:

 * `VAULT_PATH` - a path to a Vault Key/Value material that contains the
   Netbox secret. It is expected that the material contains an `key`. This
   assumes that the Key/Value store is mounted to the path kv/. This is
   optional but if it is not specified then password must be specified.
 * `NETBOX_HOST` - a full URL to the Netbox host
 * `HTTP_SERVER` - the path to the HTTP server running on this host for
   making self-referential links
 * `DEFAULT_CONFIG` - the ID of the default configuration context used
   when APKOVL files are requested for a non-existing device

If these fields are not specified at build time they can be overriden as
command line flags.

## Running

In practice, if you have set the defaults properly during the build
stage then running the application with no arguments will start a
functional bootstrapping server.

### Requirements

 * Hashicorp Vault
 * Netbox
 * DHCP server

### Environment

The following environment variables must be exported before starting the
process.

* `VAULT_ADDR` the HTTP/S address the Vault server. 
* `VAULT_TOKEN` (optional) a Vault token to use for authentication
* `VAULT_ROLE_ID` and `VAULT_SECRET_ID` (optional) used to authenticate
  to Vault using the AppRole backend. Either these or `VAULT_TOKEN` must
  be specified otherwise Vault will fail to initialize.

### Command Line

The following command line flags are supported and these ones are mandatory:

 * `--netbox-host` a full URL to Netbox
 * `--http-server` the path to the HTTP server running on this host for
   making self-referential links
 * `--vault-netbox-path` the path to a Key/Value material in Vault that
   contains a `key` used for authenticating to the Netbox API. This assumes
   that Vault has a KV backend mounted at `kv/`
 * `--default-config-id` the ID of the default configuration context
   used when APKOVL files are requested for a non-existing device

The following flags are optional:

 * `--debug` enables debug logging
 * `--bind-http` (default: `:80`) the address and port to which the HTTP
   server will bind
 * `--bind-tftp` (default: `:69`) the address and port to which the TFTP
   server will bind
 * `--distro-files` (default: `/netboot`) filesystem path to the
   distribution catalog
 * `--ntp-server` (default: `0.pool.ntp.org`) the NTP server iPXE
   clients are configured to use
 * `--vars-config` (default: `vars.yaml`) the name of the YAML vars file
   for the distribution catalog

### Configuring DHCP

Clients requesting IP addresses from the DHCP server that are not iPXE
should be redirected to the TFTP server so that they can bootstrap iPXE.
iPXE clients should be directed to the HTTP server `/boot.ixpe` URL
for chainloading their configuration. This configuration is specific
to whatever DHCP server you're running. Here are two examples of
configuration for common DHCP servers.

#### ISC DHCPD

For DHCPD use an if statement in the subnet to sort out hosts with
different architectures and redirect them to the correct iPXE payload.
The `next-server` is the IP address of the TFTP server.

```
shared-network LAN-subnet {
  subnet 10.0.0.0 netmask 255.255.255.0 {
    next-server 10.0.0.1;
    
    if exists user-class and option user-class = "iPXE" {
      filename "http://10.0.0.1/boot.ipxe";
    # 00:00    Intel x86PC
    } elsif option arch = 00:00 {
      filename "undionly.kpxe";
    # 00:0b    ARM 64-bit UEFI
    } elsif option arch = 00:0b {
      filename "ipxe-arm64.efi";
    } else {
      filename "ipxe-x86_64.efi";
    }
  }
}
```

#### dnsmasq

For dnsmasq it's a little bit more complicated. First you must tag
the client architecture, then you can map it to a boot argument. The
example below shows a dnsmasq configuration for a server that is
serving multiple networks. The first match tags an architecture, the
`tag-if` statement ensures that boot arguments are only sent for hosts
in a specific subnet, and finally the `dhcp-boot` selects the correct
boot image based on the tags. A more simple setup with only a single
subnet could skip the second configuration block and instead match the
architecture directly using `dhcp-boot`.

```
# First match the client architecture
dhcp-match=set:arch-x86-bios,option:client-arch,0
dhcp-match=set:arch-x86-efi,option:client-arch,6
dhcp-match=set:arch-x86_64-efi,option:client-arch,7
dhcp-match=set:arch-arm32-efi,option:client-arch,10
dhcp-match=set:arch-arm64-efi,option:client-arch,11
dhcp-userclass=set:class-ipxe,iPXE

# The tag with the correct boot path
tag-if=tag:net-bootstrap,tag:class-ipxe,set:bootstrap-ipxe
tag-if=tag:net-bootstrap,tag:!class-ipxe,tag:arch-x86-bios,set:bootstrap-x86-bios
tag-if=tag:net-bootstrap,tag:!class-ipxe,tag:arch-arm64-efi,set:bootstrap-arm64-efi
tag-if=tag:net-bootstrap,tag:!class-ipxe,tag:arch-x86-efi,set:bootstrap-x86-efi
tag-if=tag:net-bootstrap,tag:!class-ipxe,tag:arch-x86_64-efi,set:bootstrap-x86_64-efi

# Send the correct boot argument to the client
dhcp-boot=tag:bootstrap-ipxe,"http://10.0.0.1/boot.ipxe"
dhcp-boot=tag:bootstrap-arm64-efi,"ipxe-arm64.efi"
dhcp-boot=tag:bootstrap-x86-bios,"undionly.kpxe"
dhcp-boot=tag:bootstrap-x86_64-efi,"ipxe.efi"
dhcp-boot=tag:bootstrap-x86-efi,"ipxe.efi"
```
