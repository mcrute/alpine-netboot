package app

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	mapset "github.com/deckarep/golang-set/v2"
	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v2"
)

type DistroList []*Distribution

func (l DistroList) Len() int      { return len(l) }
func (l DistroList) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
func (l DistroList) Less(i, j int) bool {
	if cmp := semver.Compare(l[i].FullVersion, l[j].FullVersion); cmp != 0 {
		return cmp < 0
	}
	return l[i].FullVersion < l[j].FullVersion
}

type Distribution struct {
	ShortName    string
	Name         string `yaml:"name"`
	Default      bool   `yaml:"default"`
	FullVersion  string
	Architecture string
	KernelName   string           `yaml:"kernel"`
	InitrdName   string           `yaml:"initrd"`
	KernelParams []KernelArgument `yaml:"kernel_args"`
	Files        map[string]any
}

func DistributionFromYaml(f fs.FS, path string) (*Distribution, error) {
	fd, err := f.Open(path)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	d := &Distribution{}
	if err := yaml.NewDecoder(fd).Decode(&d); err != nil {
		return nil, err
	}

	return d, nil
}

func (d Distribution) DistroPath() string {
	return filepath.Join("/distros", d.ShortName, d.FullVersion, d.Architecture)
}

func (d Distribution) Slug() string {
	return strings.Join([]string{
		d.ShortName,
		d.FullVersion,
		d.Architecture,
	}, "-")
}

func (d Distribution) BaseVersion() string {
	parts := strings.Split(d.FullVersion, ".")
	if len(parts) > 2 {
		return strings.Join(parts[:2], ".")
	}
	return d.FullVersion
}

func (d Distribution) KernelCommandLine() string {
	out := []string{
		// Should always be first
		fmt.Sprintf("initrd=%s", d.InitrdName),
	}

	for _, a := range d.KernelParams {
		// Skip any arguments that fail to render
		if arg, err := a.Render(&d); err == nil {
			out = append(out, arg)
		}
	}

	// Should always be last
	out = append(out, "console=ttyS0,115200n8")

	return strings.Join(out, " ")
}

func (d Distribution) FilesContainDistro(files mapset.Set[string]) bool {
	return files.Contains(d.KernelName) && files.Contains(d.InitrdName)
}
