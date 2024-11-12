# Built-in Plugins

These plugins are built-in by default.

## alpine_init

This plugin configures services to start at various OpenRC run levels.
It is the equivalent of creating symlinks from `/etc/init.d` to
`/etc/runlevels/<key>/<value>` or running `rc-update`.

This plugin supports config grouping.

The configuration format is, where `runlevel-name` is the name of an
OpenRC run level (such as default) and `service-name` is the name of a
service (such as sshd):

```
{
    "alpine_init": {
        "group-name": {
            "runlevel-name": [
                "service-name"
            ]
        }
    }
}
```

## alpine_keys

This plugin configures trusted keys for APK in `/etc/apk/keys`.

This plugin supports config grouping.

The configuration format is, where `key-name` is the name of the key
file and `key-value` can be either the contents of the key file or a URL
starting with either `http://` or `https://` that will be fetched at OVL
generation time and added as a key file:

```
{
    "alpine_keys": {
        "group-name": {
            "my.key": "https://example.com/keys",
            "my.other.key": "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgk..."
        }
    }
}
```

## alpine_packages

This plugin configures packages to be installed via `apk fix` before the
system boots.

This plugin supports config grouping.

The configuration format is:

```
{
    "alpine_packages": {
        "group-name": [
            "package-a",
            "package-b"
        ]
    }
}
```

## alpine_repos

This plugin configures APK repositories in the `/etc/apk/repositories`
file.

This plugin supports config grouping.

The configuration format is:

```
{
    "alpine_repos": {
        "group-name": [
            "http://dl-cdn.alpinelinux.org/alpine/v3.20/main/",
            "http://dl-cdn.alpinelinux.org/alpine/v3.20/community/"
        ]
    }
}
```

## alpine_start_default_services

This plugin instructs the Alpine init script bootstrap
logic to enable default boot services. The list
of services can be found in Alpine's [mkinitfs
repository](https://gitlab.alpinelinux.org/alpine/mkinitfs/-/blob/HEAD/i
nitramfs-init.in#L793).

This plugin only supports a boolean.

## alpine_syslogd

This plugin configures the Busybox syslogd daemon by writing
`/etc/conf.d/syslog`. Using this plugin does not imply starting the
service. Use `alpine_init` for that.

The configuration is a JSON map containing the following keys:

 * `keep_n_rotated` the number of logs to keep after they are rotated
   (syslog `-b` argument)
 * `max_size` the maximum size of a log before rotation occurs (syslog
   `-s` argument)
 * `strip_client_timestamps` whether to strip the client timestamp from
   log messages (syslog `-t` argument)

## doas

This plugin creates a [doas.conf](https://man.openbsd.org/doas.conf.5)
file in `/etc/doas.d/local.conf`.

The plugin configuration is a list of JSON maps containing the following
fields (see the doas man page for the value and meaning of these
fields):

 * `action`
 * `options` (array of strings)
 * `identity`
 * `as` (optional)
 * `command` (optional)
 * `args` (optional, array of strings)

## hostname

This plugin configures the system hostname as well as the DNS domain
name. This updates `/etc/hosts` and `/etc/hostname`. This plugin is
always enabled and can not be disabled. It requires no configuration.

The hostname is the device name in Netbox.

The DNS domain name is the concatenation of the hostname, a dot ("."),
and the value of the custom field named `site_base_fqdn` from the site
in which the device is located.

## ifupdown_ng

This plugin configures `/etc/network/interfaces`.

The plugin supports the following configuration:

 * `configure_loopback` (bool) generates config for the `lo` interface
 * `generate_interfaces` (bool) generates interfaces based on Netbox
   interfaces
 * `force_primary_dhcp` (bool) forces the first interface listed in
   Netbox to use DHCP to configure itself

If IPv4/6 addresses are specified for the interface in Netbox then those
addresses will be configured to the interface in the config file. IPv6
RA will be enabled for IPv6 default route configuration.

## inittab

This plugin creates or overwrites `/etc/inittab`.

The configuration has the following fields which translate directly to
the [inittab config format](https://manpages.org/inittab/5):

 * `id`
 * `runlevels` (array of strings)
 * `action`
 * `process`

For example:
```
{
    "inittab": [
        {
            "action": "sysinit",
            "process": "/sbin/openrc sysinit"
        },
        {
            "action": "sysinit",
            "process": "/sbin/openrc boot"
        },
        {
            "action": "wait",
            "process": "/sbin/openrc default"
        },
        {
            "action": "respawn",
            "id": "tty1",
            "process": "/sbin/getty 38400 tty1"
        },
        {
            "action": "respawn",
            "id": "ttyS0",
            "process": "/sbin/getty -L 115200 ttyS0 vt100"
        },
        {
            "action": "ctrlaltdel",
            "process": "/sbin/reboot"
        },
        {
            "action": "shutdown",
            "process": "/sbin/openrc shutdown"
        }
    ]
}
```

## load_kernel_modules

This plugin configures kernel modules to be loaded at boot time by
writing files to `/etc/modules-load.d`.

The configuration is a JSON map containing the name of a file and a list
of kernel modules to load. For example:

```
{
    "load_kernel_modules": {
        "01-networking": [
            "bonding",
            "tun",
            "vhost_net"
        ],
        "02-kvm": [
            "kvm-amd"
        ]
    }
}
```

## root_ssh_keys

This plugin writes the root ssh `authorized_keys` file.

The configuration is a list of root SSH keys to be added. For example:

```
{
    "root_ssh_keys": [
        "ssh-ed25519 ... admin-key"
    ]
}
```
