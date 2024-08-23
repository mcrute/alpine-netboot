package app

import (
	"context"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"sync"
	"syscall"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

var (
	scanHupMetric = promauto.NewCounter(prometheus.CounterOpts{
		Name: "netboot_scan_hup_count",
		Help: "Number of rescan events triggered by SIGHUP",
	})
	scanTimerMetric = promauto.NewCounter(prometheus.CounterOpts{
		Name: "netboot_scan_timer_count",
		Help: "Number of rescan events triggered by the timer",
	})
	scanCountMetric = promauto.NewCounter(prometheus.CounterOpts{
		Name: "netboot_scan_count",
		Help: "Number of rescan events",
	})
	scanSoftFailureMetric = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "netboot_scan_soft_failure",
		Help: "Number of failures during scan that did not abort the scan",
	}, []string{"reason"})
	scanHardFailureMetric = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "netboot_scan_hard_failure",
		Help: "Number of failures during scan that aborted the scan",
	}, []string{"reason"})
	scanFoundDistroSuccessMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "netboot_scan_distro_success",
		Help: "Number of successfully found distributions in last scan",
	})
)

type DistributionCatalog struct {
	files       fs.FS
	logger      *zap.Logger
	distros     DistroList
	watchers    []chan<- DistroList
	watchErrors chan<- error
	httpHandler http.Handler
	sync.Mutex
}

func LoadDistributionCatalog(files fs.FS, errors chan<- error, logger *zap.Logger) (*DistributionCatalog, error) {
	c := &DistributionCatalog{
		files:       files,
		logger:      logger,
		watchers:    []chan<- DistroList{},
		watchErrors: errors,
		httpHandler: http.StripPrefix("/distros/", http.FileServerFS(files)),
	}

	// Do the initial scan on startup
	if err := c.scanFiles(); err != nil {
		return nil, err
	}

	return c, nil
}

func fileSet(entries []fs.DirEntry) mapset.Set[string] {
	files := mapset.NewSet[string]()
	for _, e := range entries {
		if !e.IsDir() {
			files.Add(e.Name())
		}
	}
	return files
}

func (c *DistributionCatalog) scanVersions(root string, versionCandidates []fs.DirEntry, distro Distribution) (DistroList, error) {
	validDistros := DistroList{}

	// Walk through version candidates
	for _, versionCandidate := range versionCandidates {
		if !versionCandidate.IsDir() {
			continue
		}

		// Enumerate architecture candidates
		versionName := versionCandidate.Name()
		versionPath := filepath.Join(root, versionName)
		archCandidates, err := fs.ReadDir(c.files, versionPath)
		if err != nil {
			scanSoftFailureMetric.WithLabelValues("reason", "arch_candidate_read_failed").Inc()
			c.logger.Debug("Error reading architecture candidates",
				zap.String("path", versionPath),
				zap.Error(err),
			)
			continue
		}

		// Walk through architecture candidates
		for _, archCandidate := range archCandidates {
			if !archCandidate.IsDir() {
				continue
			}

			// Enumerate files
			archName := archCandidate.Name()
			archPath := filepath.Join(versionPath, archName)
			entries, err := fs.ReadDir(c.files, archPath)
			if err != nil {
				scanSoftFailureMetric.WithLabelValues("reason", "list_files_read_failed").Inc()
				c.logger.Debug("Error reading architecture files",
					zap.String("path", archPath),
					zap.Error(err),
				)
				continue
			}

			if distro.FilesContainDistro(fileSet(entries)) {
				newDistro := distro
				newDistro.Architecture = archName
				newDistro.FullVersion = versionName
				validDistros = append(validDistros, &newDistro)

				c.logger.Debug("Found valid distribution",
					zap.String("name", newDistro.ShortName),
					zap.String("version", versionName),
					zap.String("architecture", archName),
				)
			}
		}
	}

	return validDistros, nil
}

func (c *DistributionCatalog) scanFiles() error {
	// File system layout is:
	// <short_name>/<full_version>/<architecture>/<files>
	//
	// There must be a distro.yaml in <sort_name>/ for it to be considered
	// a distribution, othewise it's skipped.
	//
	// The kernel and initrd files named in distro.yaml must exist in
	// <files> to be considered a valid distro, otherwise it's skipped.

	distros := DistroList{}

	// Fetch distribution candidates from the filesystem root
	root, err := fs.ReadDir(c.files, ".")
	if err != nil {
		c.logger.Error("Error reading root distro candidates", zap.Error(err))
		scanHardFailureMetric.WithLabelValues("reason", "root_read_failed").Inc()
		c.watchErrors <- err
		return err
	}

	// Walk through distribution candidates
	for _, distroCandidate := range root {
		if !distroCandidate.IsDir() {
			continue
		}

		// Fetch version candidates
		versionCandidateFiles, err := fs.ReadDir(c.files, distroCandidate.Name())
		if err != nil {
			scanSoftFailureMetric.WithLabelValues("reason", "distro_candidate_read_failed").Inc()
			c.logger.Debug("Error reading distro candidate files",
				zap.String("distro", distroCandidate.Name()),
				zap.Error(err),
			)
			continue
		}

		// Walk through version candidates
		var distro *Distribution
		versionCandidates := []fs.DirEntry{}

		for _, item := range versionCandidateFiles {
			if item.IsDir() {
				// Any directory could be a candidate version if it eventually
				// passes validation in scanVersions.
				versionCandidates = append(versionCandidates, item)
			} else if !item.IsDir() && item.Name() == "distro.yaml" {
				// A distribution must have a valid distro.yaml file to be
				// considered for any further processing.
				distro, err = DistributionFromYaml(c.files, filepath.Join(distroCandidate.Name(), item.Name()))
				if err != nil {
					scanSoftFailureMetric.WithLabelValues("reason", "distro_yaml_read_failed").Inc()
					c.logger.Debug("Error loading distro.yaml",
						zap.String("distro", distroCandidate.Name()),
						zap.Error(err),
					)
					continue
				}
				// The short name of the distribution is the name of the directory
				// in which it's located
				distro.ShortName = distroCandidate.Name()
			}
		}

		// If we found a valid distro then scan all of its versions
		if distro != nil {
			scanned, err := c.scanVersions(distroCandidate.Name(), versionCandidates, *distro)
			if err != nil {
				scanHardFailureMetric.WithLabelValues("reason", "version_scan_failed").Inc()
				c.watchErrors <- err
				return err
			}
			distros = append(distros, scanned...)
		}
	}

	sort.Stable(sort.Reverse(distros))

	// Only the first distribution for an architecture that has the default
	// flag can be considered default. Unset default flags on everything
	// else.
	archHasDefault := mapset.NewSet[string]()
	for _, d := range distros {
		if d.Default {
			if archHasDefault.Contains(d.Architecture) {
				d.Default = false
			} else {
				archHasDefault.Add(d.Architecture)
			}
		}
	}

	// Flip the current set of distros to this new set...
	c.Lock()
	c.distros = distros
	c.Unlock()

	// Log some metrics
	scanCountMetric.Inc()
	scanFoundDistroSuccessMetric.Set(float64(len(c.distros)))

	// ... and notify all of our watchers
	for _, watcher := range c.watchers {
		watcher <- c.distros
	}

	return nil
}

func (c *DistributionCatalog) Watch(notify chan<- DistroList) {
	c.watchers = append(c.watchers, notify)
	notify <- c.distros // Always give new watchers current catalog
}

func (c *DistributionCatalog) ManageAsync(ctx context.Context, wg *sync.WaitGroup) {
	go func() {
		wg.Add(1)
		defer wg.Done()

		hupChan := make(chan os.Signal)
		signal.Notify(hupChan, syscall.SIGHUP)

		t := time.NewTicker(time.Hour)

		c.logger.Info("Starting distribution scanner")

		for {
			select {
			case <-ctx.Done():
				c.logger.Info("Application finished, stopping distribution scanner")
				return
			case <-hupChan:
				c.logger.Info("Got SIGHUP, re-scanning distributions")
				scanHupMetric.Inc()
				c.scanFiles()
			case <-t.C:
				c.logger.Debug("Performing periodic scan of distributions")
				scanTimerMetric.Inc()
				c.scanFiles()
			}
		}
	}()
}

func (c *DistributionCatalog) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.httpHandler.ServeHTTP(w, r)
}
