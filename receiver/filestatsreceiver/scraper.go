// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package filestatsreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/filestatsreceiver"

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver/scrapererror"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/filestatsreceiver/internal/metadata"
)

type scraper struct {
	include        string
	logger         *zap.Logger
	metricsBuilder *metadata.MetricsBuilder
}

func (s *scraper) scrape(ctx context.Context) (pmetric.Metrics, error) {
	matches, err := doublestar.FilepathGlob(s.include)
	if err != nil {
		return pmetric.NewMetrics(), err
	}

	var scrapeErrors []error

	now := pcommon.NewTimestampFromTime(time.Now())

	for _, match := range matches {
		fileinfo, err := os.Stat(match)
		if err != nil {
			scrapeErrors = append(scrapeErrors, err)
			continue
		}
		path, err := filepath.Abs(fileinfo.Name())
		if err != nil {
			scrapeErrors = append(scrapeErrors, err)
			continue
		}
		s.metricsBuilder.RecordFileSizeDataPoint(now, fileinfo.Size())
		s.metricsBuilder.RecordFileMtimeDataPoint(now, fileinfo.ModTime().Unix())
		collectStats(now, fileinfo, s.metricsBuilder, s.logger)
		s.metricsBuilder.EmitForResource(metadata.WithFileName(fileinfo.Name()), metadata.WithFilePath(path))
	}

	if len(scrapeErrors) > 0 {
		return s.metricsBuilder.Emit(), scrapererror.NewPartialScrapeError(multierr.Combine(scrapeErrors...), len(scrapeErrors))
	}
	return s.metricsBuilder.Emit(), nil
}

func newScraper(metricsBuilder *metadata.MetricsBuilder, cfg *Config, logger *zap.Logger) *scraper {
	return &scraper{
		include:        cfg.Include,
		logger:         logger,
		metricsBuilder: metricsBuilder,
	}
}
