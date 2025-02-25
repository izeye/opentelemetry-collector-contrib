// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build integration
// +build integration

package nginxreceiver

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.opentelemetry.io/collector/component"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/scraperinttest"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/pmetrictest"
)

const nginxPort = "80"

func TestNginxIntegration(t *testing.T) {
	scraperinttest.NewIntegrationTest(
		NewFactory(),
		testcontainers.ContainerRequest{
			FromDockerfile: testcontainers.FromDockerfile{
				Context:    filepath.Join("testdata", "integration"),
				Dockerfile: "Dockerfile.nginx",
			},
			ExposedPorts: []string{nginxPort},
			WaitingFor:   wait.ForListeningPort(nginxPort),
		},
		scraperinttest.WithCustomConfig(
			func(cfg component.Config, host string, mappedPort scraperinttest.MappedPortFunc) {
				port := mappedPort(nginxPort)
				rCfg := cfg.(*Config)
				rCfg.ScraperControllerSettings.CollectionInterval = 100 * time.Millisecond
				rCfg.Endpoint = fmt.Sprintf("http://%s:%s/status", host, port)
			}),
		scraperinttest.WithCompareOptions(
			pmetrictest.IgnoreMetricValues(),
			pmetrictest.IgnoreStartTimestamp(),
			pmetrictest.IgnoreTimestamp(),
		),
	).Run(t)
}
