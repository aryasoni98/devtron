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
	"github.com/devtron-labs/devtron/api/helm-app/gRPC"
	openapi "github.com/devtron-labs/devtron/api/helm-app/openapiClient"
	"github.com/devtron-labs/devtron/pkg/appStore/bean"
)

const (
	SOURCE_DEVTRON_APP       SourceAppType = "devtron-app"
	SOURCE_HELM_APP          SourceAppType = "helm-app"
	SOURCE_EXTERNAL_HELM_APP SourceAppType = "external-helm-app"
	SOURCE_LINKED_HELM_APP   SourceAppType = "linked-helm-app"
	SOURCE_UNKNOWN           SourceAppType = "unknown"
	ErrReleaseNotFound       string        = "release: not found"
)

type SourceAppType string

type UpdateApplicationRequestDto struct {
	*openapi.UpdateReleaseRequest
	SourceAppType SourceAppType `json:"-"`
}

type UpdateApplicationWithChartInfoRequestDto struct {
	*gRPC.InstallReleaseRequest
	SourceAppType SourceAppType `json:"-"`
}

func ConvertToInstalledAppInfo(installedApp *appStoreBean.InstallAppVersionDTO) *InstalledAppInfo {
	if installedApp == nil {
		return nil
	}

	chartInfo := installedApp.InstallAppVersionChartDTO

	return &InstalledAppInfo{
		AppId:                 installedApp.AppId,
		EnvironmentName:       installedApp.EnvironmentName,
		AppOfferingMode:       installedApp.AppOfferingMode,
		InstalledAppId:        installedApp.InstalledAppId,
		InstalledAppVersionId: installedApp.InstalledAppVersionId,
		AppStoreChartId:       chartInfo.AppStoreChartId,
		ClusterId:             installedApp.ClusterId,
		EnvironmentId:         installedApp.EnvironmentId,
		AppStoreChartRepoName: chartInfo.InstallAppVersionChartRepoDTO.RepoName,
		AppStoreChartName:     chartInfo.ChartName,
		TeamId:                installedApp.TeamId,
		TeamName:              installedApp.TeamName,
	}
}

type AppDetailAndInstalledAppInfo struct {
	InstalledAppInfo *InstalledAppInfo `json:"installedAppInfo"`
	AppDetail        *gRPC.AppDetail   `json:"appDetail"`
}

type ReleaseAndInstalledAppInfo struct {
	InstalledAppInfo *InstalledAppInfo `json:"installedAppInfo"`
	ReleaseInfo      *gRPC.ReleaseInfo `json:"releaseInfo"`
}

type DeploymentHistoryAndInstalledAppInfo struct {
	InstalledAppInfo  *InstalledAppInfo               `json:"installedAppInfo"`
	DeploymentHistory []*gRPC.HelmAppDeploymentDetail `json:"deploymentHistory"`
}

type InstalledAppInfo struct {
	AppId                 int    `json:"appId"`
	InstalledAppId        int    `json:"installedAppId"`
	InstalledAppVersionId int    `json:"installedAppVersionId"`
	AppStoreChartId       int    `json:"appStoreChartId"`
	EnvironmentName       string `json:"environmentName"`
	AppOfferingMode       string `json:"appOfferingMode"`
	ClusterId             int    `json:"clusterId"`
	EnvironmentId         int    `json:"environmentId"`
	AppStoreChartRepoName string `json:"appStoreChartRepoName"`
	AppStoreChartName     string `json:"appStoreChartName"`
	TeamId                int    `json:"teamId"`
	TeamName              string `json:"teamName"`
	DeploymentType        string `json:"deploymentType"`
	HelmPackageName       string `json:"helmPackageName"`
}
