/*
 * Copyright (c) 2024. Devtron Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package bean

import (
	"encoding/base64"
	"encoding/json"
	"github.com/caarlos0/env"
	"github.com/docker/cli/cli/config/types"
	"time"
)

const (
	YamlSeparator       string = "---\n"
	RegistryTypeGcr            = "gcr"
	RegistryTypeEcr            = "ecr"
	GcrRegistryUsername        = "oauth2accesstoken"
	GcrRegistryScope           = "https://www.googleapis.com/auth/cloud-platform"
)

type DockerAuthConfig struct {
	RegistryType          string // can be ecr, gcr, docker-hub, harbor etc.
	Username              string
	Password              string
	AccessKeyEcr          string // used for pulling from private ecr registry
	SecretAccessKeyEcr    string // used for pulling from private ecr registry
	EcrRegion             string // used for pulling from private ecr registry
	CredentialFileJsonGcr string // used for pulling from private gcr registry
	IsRegistryPrivate     bool
}

func (r *DockerAuthConfig) GetEncodedRegistryAuth() (string, error) {
	// Create and encode the auth config
	authConfig := types.AuthConfig{
		Username: r.Username,
		Password: r.Password,
	}
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encodedJSON), nil
}

type DockerRegistryInfo struct {
	DockerImageTag     string `json:"dockerImageTag"`
	DockerRegistryId   string `json:"dockerRegistryId"`
	DockerRegistryType string `json:"dockerRegistryType"`
	DockerRegistryURL  string `json:"dockerRegistryURL"`
	DockerRepository   string `json:"dockerRepository"`
}

type PgQueryMonitoringConfig struct {
	LogSlowQuery           bool  `env:"PG_LOG_SLOW_QUERY" envDefault:"true"`
	LogAllQuery            bool  `env:"PG_LOG_ALL_QUERY" envDefault:"false"`
	LogAllFailureQueries   bool  `env:"PG_LOG_ALL_FAILURE_QUERIES" envDefault:"true"`
	ExportPromMetrics      bool  `env:"PG_EXPORT_PROM_METRICS" envDefault:"true"`
	QueryDurationThreshold int64 `env:"PG_QUERY_DUR_THRESHOLD" envDefault:"5000"`
	ServiceName            string
}

func GetPgQueryMonitoringConfig(serviceName string) (PgQueryMonitoringConfig, error) {
	cfg := &PgQueryMonitoringConfig{}
	err := env.Parse(cfg)
	return *cfg, err
}

type PgQueryEvent struct {
	StartTime time.Time
	Error     error
	Query     string
	FuncName  string
}

type TargetPlatform struct {
	Name string `json:"name"`
}
