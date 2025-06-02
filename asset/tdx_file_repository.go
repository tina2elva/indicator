// Copyright (c) 2021-2024 Onur Cinar.
// The source code is provided under GNU AGPLv3 License.
// https://github.com/cinar/indicator

package asset

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cinar/indicator/v2/helper"
)

// TdxFileRepository stores and retrieves asset snapshots using
// the local file system.
type TdxFileRepository struct {
	// base is the root directory where asset snapshots are stored.
	base string
	ext  string // file extension
}

// NewTdxFileRepository initializes a file system repository with
// the given base directory.
func NewTdxFileRepository(base string, ext string) *TdxFileRepository {
	return &TdxFileRepository{
		base: base,
	}
}

// Assets returns the names of all assets in the repository.
func (r *TdxFileRepository) Assets() ([]string, error) {
	files, err := os.ReadDir(r.base)
	if err != nil {
		return nil, err
	}

	var assets []string

	suffix := r.ext

	for _, file := range files {
		name := file.Name()

		if strings.HasSuffix(name, suffix) {
			assets = append(assets, strings.TrimSuffix(name, suffix))
		}
	}

	return assets, nil
}

// Get attempts to return a channel of snapshots for the asset with the given name.
func (r *TdxFileRepository) Get(name string) (<-chan *Snapshot, error) {
	bars, err := helper.ReadFromTdxFile[Snapshot](r.getTdxFileName(name))
	if err != nil {
		return nil, err
	}

	snapshots := make(chan *Snapshot)

	go func() {
		for bar := range bars {
			snapshot := &Snapshot{
				Date:   bar.Time(),
				Open:   float64(bar.Open()),
				High:   float64(bar.High()),
				Low:    float64(bar.Low()),
				Close:  float64(bar.Close()),
				Volume: float64(bar.Volume()),
			}

			snapshots <- snapshot
		}

		close(snapshots)
	}()

	return snapshots, nil
}

// GetSince attempts to return a channel of snapshots for the asset with the given name since the given date.
func (r *TdxFileRepository) GetSince(name string, date time.Time) (<-chan *Snapshot, error) {
	snapshots, err := r.Get(name)
	if err != nil {
		return nil, err
	}

	snapshots = helper.Filter(snapshots, func(s *Snapshot) bool {
		return s.Date.Equal(date) || s.Date.After(date)
	})

	return snapshots, nil
}

// LastDate returns the date of the last snapshot for the asset with the given name.
func (r *TdxFileRepository) LastDate(name string) (time.Time, error) {
	var last time.Time

	snapshots, err := r.Get(name)
	if err != nil {
		return last, err
	}

	snapshot, ok := <-helper.Last(snapshots, 1)
	if !ok {
		return last, errors.New("empty asset")
	}

	return snapshot.Date, nil
}

// Append adds the given snapshows to the asset with the given name.
func (r *TdxFileRepository) Append(name string, snapshots <-chan *Snapshot) error {
	return errors.ErrUnsupported
}

// getTdxFileName gets the CSV file name for the given asset name.
func (r *TdxFileRepository) getTdxFileName(name string) string {
	return filepath.Join(r.base, fmt.Sprintf("%s%s", name, r.ext))
}
