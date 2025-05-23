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

package dashboard

import (
	"github.com/caarlos0/env"
)

// CATEGORY=INFRA_SETUP
type Config struct {
	Host      string `env:"DASHBOARD_HOST" envDefault:"localhost" description:"Dashboard micro-service URL"`
	Port      string `env:"DASHBOARD_PORT" envDefault:"3000" description:"Port for dashboard micro-service"`
	Namespace string `env:"DASHBOARD_NAMESPACE" envDefault:"devtroncd" description:"Dashboard micro-service namespace"`
}

func GetConfig() (*Config, error) {
	cfg := &Config{}
	err := env.Parse(cfg)
	return cfg, err
}
