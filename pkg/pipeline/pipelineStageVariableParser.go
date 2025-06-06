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

package pipeline

import (
	"errors"
	"fmt"
	commonBean "github.com/devtron-labs/common-lib/workflow"
	dockerRegistryRepository "github.com/devtron-labs/devtron/internal/sql/repository/dockerRegistry"
	"github.com/devtron-labs/devtron/internal/util"
	bean2 "github.com/devtron-labs/devtron/pkg/plugin/bean"
	"github.com/go-pg/pg"
	errors1 "github.com/juju/errors"
	"go.uber.org/zap"
	"strings"
)

type copyContainerImagePluginInputVariable = string

const (
	EMPTY_STRING = " "
)

const (
	DESTINATION_INFO                copyContainerImagePluginInputVariable = "DESTINATION_INFO"
	SOURCE_REGISTRY_CREDENTIALS_KEY                                       = "SOURCE_REGISTRY_CREDENTIAL"
)

type PluginInputVariableParser interface {
	HandleCopyContainerImagePluginInputVariables(inputVariables []*commonBean.VariableObject, dockerImageTag string, pluginTriggerImage string, sourceImageDockerRegistry string) (registryDestinationImageMap map[string][]string, registryCredentials map[string]bean2.RegistryCredentials, err error)
}

type PluginInputVariableParserImpl struct {
	logger               *zap.SugaredLogger
	dockerRegistryConfig DockerRegistryConfig
	customTagService     CustomTagService
}

func NewPluginInputVariableParserImpl(
	logger *zap.SugaredLogger,
	dockerRegistryConfig DockerRegistryConfig,
	customTagService CustomTagService,
) *PluginInputVariableParserImpl {
	return &PluginInputVariableParserImpl{
		logger:               logger,
		dockerRegistryConfig: dockerRegistryConfig,
		customTagService:     customTagService,
	}
}

func (impl *PluginInputVariableParserImpl) HandleCopyContainerImagePluginInputVariables(inputVariables []*commonBean.VariableObject,
	dockerImageTag string,
	pluginTriggerImage string,
	sourceImageDockerRegistry string) (registryDestinationImageMap map[string][]string, registryCredentials map[string]bean2.RegistryCredentials, err error) {

	var DestinationInfo string
	for _, ipVariable := range inputVariables {
		if ipVariable.Name == DESTINATION_INFO {
			DestinationInfo = ipVariable.Value
		}
	}

	if len(pluginTriggerImage) == 0 {
		return nil, nil, errors.New("no image provided during trigger time")
	}

	if len(DestinationInfo) == 0 {
		return nil, nil, errors.New("destination info now")
	}

	if len(dockerImageTag) == 0 {
		// case when custom tag is not configured - source image tag will be taken as docker image tag
		pluginTriggerImageSplit := strings.Split(pluginTriggerImage, ":")
		dockerImageTag = pluginTriggerImageSplit[len(pluginTriggerImageSplit)-1]
	}

	registryRepoMapping, err := impl.getRegistryRepoMapping(DestinationInfo)
	if err != nil {
		impl.logger.Errorw("error in getting registry repo mapping", "DestinationInfo", DestinationInfo, "err", err)
		return nil, nil, err
	}
	registryCredentials, err = impl.getRegistryDetails(registryRepoMapping, sourceImageDockerRegistry)
	if err != nil {
		impl.logger.Errorw("error in getting registry details", "err", err)
		return nil, nil, err
	}
	registryDestinationImageMap = impl.getRegistryDestinationImageMapping(registryRepoMapping, dockerImageTag, registryCredentials)

	err = impl.createEcrRepoIfRequired(registryCredentials, registryRepoMapping)
	if err != nil {
		impl.logger.Errorw("error in creating ecr repo", "err", err)
		return registryDestinationImageMap, registryCredentials, err
	}

	return registryDestinationImageMap, registryCredentials, nil
}

func (impl *PluginInputVariableParserImpl) getRegistryRepoMapping(destinationInfo string) (map[string][]string, error) {
	/*
		creating map with registry as key and list of repositories in that registry where we need to copy image
			destinationInfo format (each registry detail is separated by new line) :
				<registryName1> | <comma separated repoNames>
				<registryName2> | <comma separated repoNames>
	*/
	destinationRegistryRepositoryMap := make(map[string][]string)
	destinationRegistryRepoDetails := strings.Split(destinationInfo, "\n")
	for _, detail := range destinationRegistryRepoDetails {
		registryRepoSplit := strings.Split(detail, "|")
		if len(registryRepoSplit) != 2 {
			impl.logger.Errorw("invalid destination info format", "destinationInfo", destinationInfo)
			// skipping for invalid format
			return destinationRegistryRepositoryMap, errors.New("invalid destination info format. Please provide it in <registry-1> | <repo1>,<repo2>")
		}
		registryName := strings.Trim(registryRepoSplit[0], EMPTY_STRING)
		repositoryValuesSplit := strings.Split(registryRepoSplit[1], ",")
		var repositories []string
		for _, repositoryName := range repositoryValuesSplit {
			repositoryName = strings.Trim(repositoryName, EMPTY_STRING)
			repositories = append(repositories, repositoryName)
		}
		destinationRegistryRepositoryMap[registryName] = repositories
	}
	return destinationRegistryRepositoryMap, nil
}

func (impl *PluginInputVariableParserImpl) getRegistryDetails(destinationRegistryRepositoryMap map[string][]string, sourceRegistry string) (map[string]bean2.RegistryCredentials, error) {
	registryCredentialsMap := make(map[string]bean2.RegistryCredentials)
	//saving source registry credentials
	sourceRegistry = strings.Trim(sourceRegistry, " ")
	sourceRegistryCredentials, err := impl.getPluginRegistryCredentialsByRegistryName(sourceRegistry)
	if err != nil {
		return nil, err
	}
	registryCredentialsMap[SOURCE_REGISTRY_CREDENTIALS_KEY] = *sourceRegistryCredentials

	// saving destination registry credentials; destinationRegistryRepositoryMap -> map[registryName]= [<repo1>, <repo2>]
	for registry, _ := range destinationRegistryRepositoryMap {
		destinationRegistryCredential, err := impl.getPluginRegistryCredentialsByRegistryName(registry)
		if err != nil {
			return nil, err
		}
		registryCredentialsMap[registry] = *destinationRegistryCredential
	}
	return registryCredentialsMap, nil
}

func (impl *PluginInputVariableParserImpl) getPluginRegistryCredentialsByRegistryName(registryName string) (*bean2.RegistryCredentials, error) {
	registryCredentials, err := impl.dockerRegistryConfig.FetchOneDockerAccount(registryName)
	if err != nil {
		impl.logger.Errorw("error in fetching registry details by registry name", "err", err)
		if err == pg.ErrNoRows {
			return nil, fmt.Errorf("invalid registry name: registry details not found in global container registries")
		}
		return nil, err
	}
	return &bean2.RegistryCredentials{
		RegistryType:       string(registryCredentials.RegistryType),
		RegistryURL:        registryCredentials.RegistryURL,
		Username:           registryCredentials.Username,
		Password:           registryCredentials.Password,
		AWSRegion:          registryCredentials.AWSRegion,
		AWSSecretAccessKey: registryCredentials.AWSSecretAccessKey,
		AWSAccessKeyId:     registryCredentials.AWSAccessKeyId,
	}, nil
}

func (impl *PluginInputVariableParserImpl) getRegistryDestinationImageMapping(
	registryRepoMapping map[string][]string,
	dockerImageTag string,
	registryCredentials map[string]bean2.RegistryCredentials) map[string][]string {

	// creating map with registry as key and list of destination images in that registry
	registryDestinationImageMapping := make(map[string][]string)
	for registry, destinationRepositories := range registryRepoMapping {
		registryCredential := registryCredentials[registry]
		var destinationImages []string
		for _, repo := range destinationRepositories {
			destinationImage := fmt.Sprintf("%s/%s:%s", registryCredential.RegistryURL, repo, dockerImageTag)
			destinationImages = append(destinationImages, destinationImage)
		}
		registryDestinationImageMapping[registry] = destinationImages
	}

	return registryDestinationImageMapping
}

func (impl *PluginInputVariableParserImpl) createEcrRepoIfRequired(registryCredentials map[string]bean2.RegistryCredentials, registryRepoMapping map[string][]string) error {
	for registry, registryCredential := range registryCredentials {
		if registryCredential.RegistryType == dockerRegistryRepository.REGISTRYTYPE_ECR {
			repositories := registryRepoMapping[registry]
			for _, dockerRepo := range repositories {
				err := util.CreateEcrRepo(dockerRepo, registryCredential.AWSRegion, registryCredential.AWSAccessKeyId, registryCredential.AWSSecretAccessKey)
				if err != nil {
					if errors1.IsAlreadyExists(err) {
						impl.logger.Warnw("this repo already exists!!, skipping repo creation", "repo", dockerRepo)
					} else {
						impl.logger.Errorw("ecr repo creation failed, it might be due to authorization or any other external "+
							"dependency. please create repo manually before triggering ci", "repo", dockerRepo, "err", err)
						return err
					}
				}
			}
		}
	}
	return nil
}
