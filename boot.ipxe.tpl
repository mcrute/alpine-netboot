#!ipxe

set http_server {{ .HttpServer }}

{{ range $k, $v := .DefaultVars }}
set {{ $k }} {{ $v }}
{{- end }}

{{ range $prod, $vars := .ProductVars }}
{{- range $k, $v := $vars -}}
iseq ${product} {{ $prod }} && set {{ $k }} {{ $v }} ||
{{ end -}}
{{ end }}

ntp {{ .NTP }} ||

iseq ${buildarch} arm64 && goto menu-arm64 ||
iseq ${buildarch} x86_64 && goto menu-x86_64 ||

:menu
set space:hex 20:20
set space ${space:string}
menu Boot Menu
item --gap Operating Systems

{{ range .X86Distros }}
item {{ .Slug }} ${space} {{ .Name }} {{ .FullVersion }} ({{ .Architecture }})
{{- end }}
{{- range .ARM64Distros }}
item {{ .Slug }} ${space} {{ .Name }} {{ .FullVersion }} ({{ .Architecture }})
{{- end }}

item --gap Utilities
item show-config ${space} PXE Config
item shell ${space} iPXE Shell
item reboot ${space} Reboot system
item poweroff ${space} Power off system

choose --timeout 10000 item && goto ${item}








:menu-arm64
set space:hex 20:20
set space ${space:string}
menu Boot Menu
item --gap Operating Systems

{{- range .ARM64Distros }}
item {{ if .Default }}--default{{ end }} {{ .Slug }} ${space} {{ .Name }} {{ .FullVersion }} ({{ .Architecture }})
{{- end }}

item --gap Utilities
item show-config ${space} PXE Config
item shell ${space} iPXE Shell
item reboot ${space} Reboot system
item poweroff ${space} Power off system

choose --timeout 10000 item && goto ${item}








:menu-x86_64
set space:hex 20:20
set space ${space:string}
menu Boot Menu
item --gap Operating Systems

{{ range .X86Distros }}
item {{ if .Default }}--default{{ end }} {{ .Slug }} ${space} {{ .Name }} {{ .FullVersion }} ({{ .Architecture }})
{{- end }}

item --gap Utilities
item show-config ${space} PXE Config
item shell ${space} iPXE Shell
item reboot ${space} Reboot system
item poweroff ${space} Power off system

choose --timeout 10000 item && goto ${item}























{{ range .X86Distros }}
:{{ .Slug }}
imgfree
kernel {{ .DistroPath }}/{{ .KernelName }} {{ .KernelCommandLine }}
initrd {{ .DistroPath }}/{{ .InitrdName }}
boot
clear menu
exit 0
{{ end -}}

{{ range .ARM64Distros }}
:{{ .Slug }}
imgfree
kernel {{ .DistroPath }}/{{ .KernelName }} {{ .KernelCommandLine }}
initrd {{ .DistroPath }}/{{ .InitrdName }}
boot
clear menu
exit 0
{{ end }}

:show-config
config
goto menu

:shell
echo Type "exit" to return to menu.
shell
goto menu

:reboot
reboot

:poweroff
poweroff
