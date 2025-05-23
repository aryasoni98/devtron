/*
 * Copyright (c) 2020-2024. Devtron Inc.
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
	"encoding/json"
	"github.com/devtron-labs/devtron/util"
	"github.com/devtron-labs/devtron/util/sliceUtil"
)

type ConfigMapRootJson struct {
	ConfigMapJson ConfigMapJson `json:"ConfigMaps"`
}
type ConfigMapJson struct {
	Enabled bool              `json:"enabled"`
	Maps    []ConfigSecretMap `json:"maps"`
}

type ConfigSecretRootJson struct {
	ConfigSecretJson ConfigSecretJson `json:"ConfigSecrets"`
}

type ConfigSecretJson struct {
	Enabled bool               `json:"enabled"`
	Secrets []*ConfigSecretMap `json:"secrets"`
}

type ConfigMapAndSecretJson struct {
	ConfigMapJson    ConfigMapJson    `json:"configMapJson"`
	ConfigSecretJson ConfigSecretJson `json:"configSecretJson"`
}

type ConfigSecretMap struct {
	Name           string          `json:"name"`
	Type           string          `json:"type"`
	External       bool            `json:"external"`
	MountPath      string          `json:"mountPath"`
	Data           json.RawMessage `json:"data,omitempty"`
	ESOSecretData  json.RawMessage `json:"esoSecretData,omitempty"`
	ExternalType   string          `json:"externalType"`
	RoleARN        string          `json:"roleARN"`
	SecretData     json.RawMessage `json:"secretData,omitempty"`
	SubPath        bool            `json:"subPath"`
	ESOSubPath     []string        `json:"esoSubPath"`
	FilePermission string          `json:"filePermission"`
	ConfigSecretMapEnt
}

func (configSecret *ConfigSecretMap) GetDataMap() (map[string]string, error) {
	if len(configSecret.Data) == 0 {
		return make(map[string]string), nil
	}
	var dataMap map[string]string
	err := json.Unmarshal(configSecret.Data, &dataMap)
	return dataMap, err
}

func (configSecretJson *ConfigSecretJson) GetDereferencedSecrets() []ConfigSecretMap {
	return sliceUtil.GetDeReferencedSlice(configSecretJson.Secrets)
}

func (configSecretJson *ConfigSecretJson) SetReferencedSecrets(secrets []ConfigSecretMap) {
	configSecretJson.Secrets = sliceUtil.GetReferencedSlice(secrets)
}

func GetTransformedDataForSecretRootJsonData(data string, mode util.SecretTransformMode) (string, error) {
	secretsJson := ConfigSecretRootJson{}
	err := json.Unmarshal([]byte(data), &secretsJson)
	if err != nil {
		return "", err
	}

	for _, configData := range secretsJson.ConfigSecretJson.Secrets {
		if configData.Data == nil || configData.External {
			continue
		}
		configData.Data, err = util.GetDecodedAndEncodedData(configData.Data, mode)
		if err != nil {
			return "", err
		}
	}

	marshal, err := json.Marshal(secretsJson)
	if err != nil {
		return "", err
	}
	return string(marshal), nil
}

type ConfigType string

func (c ConfigType) String() string {
	return string(c)
}

const (
	ConfigMap ConfigType = "cm"
	Secret    ConfigType = "cs"
)
