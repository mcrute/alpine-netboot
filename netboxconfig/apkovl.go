package netboxconfig

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

const defaultMaxSize = 1_000_000_000

type APKOVL struct {
	MaxHTTPFileSize int64
	gzipWriter      *gzip.Writer
	tarWriter       *tar.Writer
}

func NewAPKOVLFromWriter(out io.Writer) *APKOVL {
	gw := gzip.NewWriter(out)
	return &APKOVL{
		gzipWriter: gw,
		tarWriter:  tar.NewWriter(gw),
	}
}

func (a *APKOVL) Close() {
	a.tarWriter.Close()
	a.gzipWriter.Close()
}

// AddEmptyFile adds an empty file
func (a *APKOVL) AddEmptyFile(name string, mode int64) error {
	return a.tarWriter.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     name,
		Size:     0,
		Mode:     mode,
		Uid:      0,
		Gid:      0,
		ModTime:  time.Now(),
	})
}

// AddStringFile adds a string to a file and appends a newline
// terminator if one isn't passed in contents
func (a *APKOVL) AddStringFile(contents, name string, mode int64) error {
	newlineLen := 0
	if !strings.HasSuffix(contents, "\n") {
		newlineLen = 1
	}

	if err := a.tarWriter.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     name,
		Size:     int64(len(contents) + newlineLen),
		Mode:     mode,
		Uid:      0,
		Gid:      0,
		ModTime:  time.Now(),
	}); err != nil {
		return err
	}

	if _, err := a.tarWriter.Write([]byte(contents)); err != nil {
		return err
	}

	if newlineLen > 0 {
		if _, err := a.tarWriter.Write([]byte("\n")); err != nil {
			return err
		}
	}

	return nil
}

// AddRCLink creates a link from a service in /etc/init.d to a named
// runlevel
func (a *APKOVL) AddRCLink(service, runlevel string) error {
	return a.tarWriter.WriteHeader(&tar.Header{
		Typeflag: tar.TypeSymlink,
		Name:     filepath.Join("etc/runlevels", runlevel, service),
		Linkname: filepath.Join("/etc/init.d", service),
		Uid:      0,
		Gid:      0,
		ModTime:  time.Now(),
	})
}

// AddStringListFile adds a file with a list of newline terminated
// strings. This is a convenience method for doing string joining
// elsewhere.
func (a *APKOVL) AddStringListFile(contents []string, name string, mode int64) error {
	return a.AddStringFile(strings.Join(contents, "\n"), name, mode)
}

// AddHTTPFile adds a file by first fetching it from an HTTP URL.
// It does not manipulate the file in any way. The size of the HTTP
// file is limited by default to 1GiB. This can be changed by setting
// MaxHTTPFileSize in APKOVL. Setting MaxHTTPFileSize to -1 disables
// this behavior.
func (a *APKOVL) AddHTTPFile(ctx context.Context, url, name string, mode int64) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("Invalid status code from HTTP server %d", res.StatusCode)
	}

	var reader io.Reader
	switch a.MaxHTTPFileSize {
	case 0: // Unset, use default
		reader = &io.LimitedReader{R: res.Body, N: defaultMaxSize}
	case -1: // Unlimited
		reader = res.Body
	default: // Set, use value
		reader = &io.LimitedReader{R: res.Body, N: a.MaxHTTPFileSize}
	}

	contents, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	if err := a.tarWriter.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     name,
		Size:     int64(len(contents)),
		Mode:     mode,
		Uid:      0,
		Gid:      0,
		ModTime:  time.Now(),
	}); err != nil {
		return err
	}

	_, err = a.tarWriter.Write(contents)
	return err
}
