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

package service

import (
	"github.com/devtron-labs/devtron/internal/util"
	appStoreBean "github.com/devtron-labs/devtron/pkg/appStore/bean"
	appStoreDiscoverRepository "github.com/devtron-labs/devtron/pkg/appStore/discover/repository"
	"go.uber.org/zap"
)

type AppStoreService interface {
	FindAllApps(filter *appStoreBean.AppStoreFilter) ([]appStoreBean.AppStoreWithVersion, error)
	FindChartDetailsById(id int) (appStoreBean.AppStoreApplicationVersionResponse, error)
	FindChartVersionsByAppStoreId(appStoreId int) ([]appStoreBean.AppStoreVersionsResponse, error)
	GetChartInfoByAppStoreApplicationVersionId(id int) (*appStoreBean.ChartInfoRes, error)
	SearchAppStoreChartByName(chartName string) ([]*appStoreBean.ChartRepoSearch, error)
}

type AppStoreServiceImpl struct {
	logger                        *zap.SugaredLogger
	appStoreApplicationRepository appStoreDiscoverRepository.AppStoreApplicationVersionRepository
}

func NewAppStoreServiceImpl(logger *zap.SugaredLogger,
	appStoreApplicationRepository appStoreDiscoverRepository.AppStoreApplicationVersionRepository) *AppStoreServiceImpl {
	return &AppStoreServiceImpl{
		logger:                        logger,
		appStoreApplicationRepository: appStoreApplicationRepository,
	}
}

func (impl *AppStoreServiceImpl) FindAllApps(filter *appStoreBean.AppStoreFilter) ([]appStoreBean.AppStoreWithVersion, error) {
	appStoreApplications, err := impl.appStoreApplicationRepository.FindWithFilter(filter)
	if err != nil && !util.IsErrNoRows(err) {
		return nil, err
	}
	return appStoreApplications, nil
}

func (impl *AppStoreServiceImpl) FindChartDetailsById(id int) (appStoreBean.AppStoreApplicationVersionResponse, error) {
	chartDetails, err := impl.appStoreApplicationRepository.FindById(id)
	if err != nil && !util.IsErrNoRows(err) {
		impl.logger.Error(err)
		return appStoreBean.AppStoreApplicationVersionResponse{}, err
	}
	appStoreApplicationVersion := appStoreBean.AppStoreApplicationVersionResponse{
		Id:                      chartDetails.Id,
		Version:                 chartDetails.Version,
		AppVersion:              chartDetails.AppVersion,
		Created:                 chartDetails.Created,
		Deprecated:              chartDetails.Deprecated,
		Description:             chartDetails.Description,
		Digest:                  chartDetails.Digest,
		Icon:                    chartDetails.Icon,
		Name:                    chartDetails.Name,
		AppStoreApplicationName: chartDetails.AppStore.Name,
		Home:                    chartDetails.Home,
		Source:                  chartDetails.Source,
		ValuesYaml:              chartDetails.ValuesYaml,
		ChartYaml:               chartDetails.ChartYaml,
		AppStoreId:              chartDetails.AppStoreId,
		CreatedOn:               chartDetails.CreatedOn,
		UpdatedOn:               chartDetails.UpdatedOn,
		RawValues:               chartDetails.RawValues,
		Readme:                  chartDetails.Readme,
		ValuesSchemaJson:        chartDetails.ValuesSchemaJson,
		Notes:                   chartDetails.Notes,
	}
	if chartDetails.AppStore.DockerArtifactStore != nil {
		appStoreApplicationVersion.ChartName = chartDetails.AppStore.DockerArtifactStore.Id
		appStoreApplicationVersion.IsChartRepoActive = chartDetails.AppStore.DockerArtifactStore.Active
		appStoreApplicationVersion.IsOCICompliantChart = true
	}
	if chartDetails.AppStore.ChartRepo != nil {
		appStoreApplicationVersion.ChartName = chartDetails.AppStore.ChartRepo.Name
		appStoreApplicationVersion.IsChartRepoActive = chartDetails.AppStore.ChartRepo.Active
		appStoreApplicationVersion.IsOCICompliantChart = false
	}
	return appStoreApplicationVersion, nil
}

func (impl *AppStoreServiceImpl) FindChartVersionsByAppStoreId(appStoreId int) ([]appStoreBean.AppStoreVersionsResponse, error) {
	appStoreVersions, err := impl.appStoreApplicationRepository.FindVersionsByAppStoreId(appStoreId)
	if err != nil && !util.IsErrNoRows(err) {
		impl.logger.Error(err)
		return nil, err
	}
	var appStoreVersionsResponse []appStoreBean.AppStoreVersionsResponse
	for _, a := range appStoreVersions {
		res := appStoreBean.AppStoreVersionsResponse{
			Id:      a.Id,
			Version: a.Version,
		}
		appStoreVersionsResponse = append(appStoreVersionsResponse, res)
	}
	return appStoreVersionsResponse, nil
}

func (impl *AppStoreServiceImpl) GetChartInfoByAppStoreApplicationVersionId(id int) (*appStoreBean.ChartInfoRes, error) {
	appVersion, err := impl.appStoreApplicationRepository.GetChartInfoById(id)
	if err != nil {
		return nil, err
	}
	readme := &appStoreBean.ChartInfoRes{
		AppStoreApplicationVersionId: appVersion.Id,
		Readme:                       appVersion.Readme,
		ValuesSchemaJson:             appVersion.ValuesSchemaJson,
		Notes:                        appVersion.Notes,
	}
	return readme, nil
}

func (impl *AppStoreServiceImpl) SearchAppStoreChartByName(chartName string) ([]*appStoreBean.ChartRepoSearch, error) {
	appStoreApplications, err := impl.appStoreApplicationRepository.SearchAppStoreChartByName(chartName)
	if err != nil && !util.IsErrNoRows(err) {
		return nil, err
	}
	for _, appStore := range appStoreApplications {
		if len(appStore.ChartRepoName) == 0 && len(appStore.DockerArtifactStoreId) != 0 {
			appStore.ChartRepoName = appStore.DockerArtifactStoreId
		}
	}
	return appStoreApplications, nil
}
