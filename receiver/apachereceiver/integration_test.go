// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build integration
// +build integration

package apachereceiver

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.opentelemetry.io/collector/component"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/scraperinttest"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/pmetrictest"
)

const apachePort = "80"

func TestApacheIntegration(t *testing.T) {
	scraperinttest.NewIntegrationTest(
		NewFactory(),
		testcontainers.ContainerRequest{
			FromDockerfile: testcontainers.FromDockerfile{
				Context:    path.Join("testdata", "integration"),
				Dockerfile: "Dockerfile.apache",
			},
			ExposedPorts: []string{"80"},
			WaitingFor:   waitStrategy{},
		},
		scraperinttest.WithCustomConfig(
			func(cfg component.Config, host string, mappedPort scraperinttest.MappedPortFunc) {
				port := mappedPort(apachePort)
				rCfg := cfg.(*Config)
				rCfg.ScraperControllerSettings.CollectionInterval = 100 * time.Millisecond
				rCfg.Endpoint = fmt.Sprintf("http://%s:%s/server-status?auto", host, port)
			}),
		scraperinttest.WithCompareOptions(
			pmetrictest.IgnoreResourceAttributeValue("apache.server.port"),
			pmetrictest.IgnoreMetricValues(),
			pmetrictest.IgnoreMetricDataPointsOrder(),
			pmetrictest.IgnoreStartTimestamp(),
			pmetrictest.IgnoreTimestamp(),
		),
	).Run(t)
}

type waitStrategy struct{}

func (ws waitStrategy) WaitUntilReady(ctx context.Context, st wait.StrategyTarget) error {
	if err := wait.ForListeningPort(apachePort).
		WithStartupTimeout(time.Minute).
		WaitUntilReady(ctx, st); err != nil {
		return err
	}

	hostname, err := st.Host(ctx)
	if err != nil {
		return err
	}
	port, err := st.MappedPort(ctx, apachePort)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
			return fmt.Errorf("server startup problem")
		case <-time.After(100 * time.Millisecond):
			resp, err := http.Get(fmt.Sprintf("http://%s:%s/server-status?auto", hostname, port.Port()))
			if err != nil {
				continue
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				continue
			}

			if resp.Body.Close() != nil {
				continue
			}

			// The server needs a moment to generate some stats
			if strings.Contains(string(body), "ReqPerSec") {
				return nil
			}
		}
	}
}
