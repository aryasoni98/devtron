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

package pipeline

import (
	"context"
	"fmt"
	bean5 "github.com/devtron-labs/devtron/api/bean"
	"github.com/devtron-labs/devtron/api/bean/gitOps"
	"github.com/devtron-labs/devtron/api/helm-app/service"
	helmBean "github.com/devtron-labs/devtron/api/helm-app/service/bean"
	"github.com/devtron-labs/devtron/client/argocdServer"
	"github.com/devtron-labs/devtron/internal/sql/repository/app"
	"github.com/devtron-labs/devtron/internal/sql/repository/appStatus"
	"github.com/devtron-labs/devtron/internal/sql/repository/pipelineConfig"
	"github.com/devtron-labs/devtron/internal/sql/repository/pipelineConfig/bean/workflow/cdWorkflow"
	"github.com/devtron-labs/devtron/internal/util"
	app2 "github.com/devtron-labs/devtron/pkg/app"
	userBean "github.com/devtron-labs/devtron/pkg/auth/user/bean"
	"github.com/devtron-labs/devtron/pkg/bean"
	chartService "github.com/devtron-labs/devtron/pkg/chart"
	"github.com/devtron-labs/devtron/pkg/chart/read"
	"github.com/devtron-labs/devtron/pkg/deployment/common"
	"github.com/devtron-labs/devtron/pkg/deployment/common/adapter"
	bean4 "github.com/devtron-labs/devtron/pkg/deployment/common/bean"
	read2 "github.com/devtron-labs/devtron/pkg/deployment/common/read"
	commonBean "github.com/devtron-labs/devtron/pkg/deployment/gitOps/common/bean"
	"github.com/devtron-labs/devtron/pkg/deployment/gitOps/config"
	bean3 "github.com/devtron-labs/devtron/pkg/deployment/trigger/devtronApps/bean"
	clientErrors "github.com/devtron-labs/devtron/pkg/errors"
	"github.com/devtron-labs/devtron/pkg/eventProcessor/out"
	bean2 "github.com/devtron-labs/devtron/pkg/eventProcessor/out/bean"
	"github.com/juju/errors"
	"go.uber.org/zap"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	"strconv"
)

type AppDeploymentTypeChangeManager interface {
	// ChangeDeploymentType : takes in DeploymentAppTypeChangeRequest struct and
	// deletes all the cd pipelines for that deployment type in all apps that belongs to
	// that environment and updates the db with desired deployment app type
	ChangeDeploymentType(ctx context.Context, request *bean.DeploymentAppTypeChangeRequest, userMetadata *userBean.UserMetadata) (*bean.DeploymentAppTypeChangeResponse, error)
	// ChangePipelineDeploymentType : takes in DeploymentAppTypeChangeRequest struct and
	// deletes all the cd pipelines for that deployment type in all apps that belongs to
	// that environment and updates the db with desired deployment app type
	ChangePipelineDeploymentType(ctx context.Context, request *bean.DeploymentAppTypeChangeRequest, userMetadata *userBean.UserMetadata) (*bean.DeploymentAppTypeChangeResponse, error)
	// TriggerDeploymentAfterTypeChange : triggers a new deployment after type change
	TriggerDeploymentAfterTypeChange(ctx context.Context, request *bean.DeploymentAppTypeChangeRequest, userMetadata *userBean.UserMetadata) (*bean.DeploymentAppTypeChangeResponse, error)
	// DeleteDeploymentApps : takes in a list of pipelines and delete the applications
	DeleteDeploymentApps(ctx context.Context, pipelines []*pipelineConfig.Pipeline, deploymentConfig []*bean4.DeploymentConfig, userId int32) *bean.DeploymentAppTypeChangeResponse
	// DeleteDeploymentAppsForEnvironment : takes in environment id and current deployment app type
	// and deletes all the cd pipelines for that deployment type in all apps that belongs to
	// that environment.
	DeleteDeploymentAppsForEnvironment(ctx context.Context, environmentId int, currentDeploymentAppType bean3.DeploymentType, exclusionList []int, includeApps []int, userId int32) (*bean.DeploymentAppTypeChangeResponse, error)
}

type AppDeploymentTypeChangeManagerImpl struct {
	logger                      *zap.SugaredLogger
	pipelineRepository          pipelineConfig.PipelineRepository
	appService                  app2.AppService
	appStatusRepository         appStatus.AppStatusRepository
	helmAppService              service.HelmAppService
	appArtifactManager          AppArtifactManager
	cdPipelineConfigService     CdPipelineConfigService
	gitOpsConfigReadService     config.GitOpsConfigReadService
	chartService                chartService.ChartService
	workflowEventPublishService out.WorkflowEventPublishService
	deploymentConfigService     common.DeploymentConfigService
	ArgoClientWrapperService    argocdServer.ArgoClientWrapperService
	chartReadService            read.ChartReadService
	DeploymentConfigReadService read2.DeploymentConfigReadService
}

func NewAppDeploymentTypeChangeManagerImpl(
	logger *zap.SugaredLogger,
	pipelineRepository pipelineConfig.PipelineRepository,
	appService app2.AppService,
	appStatusRepository appStatus.AppStatusRepository,
	helmAppService service.HelmAppService,
	appArtifactManager AppArtifactManager,
	cdPipelineConfigService CdPipelineConfigService,
	gitOpsConfigReadService config.GitOpsConfigReadService,
	chartService chartService.ChartService,
	workflowEventPublishService out.WorkflowEventPublishService,
	deploymentConfigService common.DeploymentConfigService,
	chartReadService read.ChartReadService,
	DeploymentConfigReadService read2.DeploymentConfigReadService) *AppDeploymentTypeChangeManagerImpl {
	return &AppDeploymentTypeChangeManagerImpl{
		logger:                      logger,
		pipelineRepository:          pipelineRepository,
		appService:                  appService,
		appStatusRepository:         appStatusRepository,
		helmAppService:              helmAppService,
		appArtifactManager:          appArtifactManager,
		cdPipelineConfigService:     cdPipelineConfigService,
		gitOpsConfigReadService:     gitOpsConfigReadService,
		chartService:                chartService,
		workflowEventPublishService: workflowEventPublishService,
		deploymentConfigService:     deploymentConfigService,
		chartReadService:            chartReadService,
		DeploymentConfigReadService: DeploymentConfigReadService,
	}
}

func (impl *AppDeploymentTypeChangeManagerImpl) ChangeDeploymentType(ctx context.Context,
	request *bean.DeploymentAppTypeChangeRequest, userMetadata *userBean.UserMetadata) (*bean.DeploymentAppTypeChangeResponse, error) {

	var response *bean.DeploymentAppTypeChangeResponse
	var deleteDeploymentType bean3.DeploymentType
	var err error

	if request.DesiredDeploymentType == bean3.ArgoCd {
		deleteDeploymentType = bean3.Helm
	} else {
		deleteDeploymentType = bean3.ArgoCd
	}

	// Force delete apps
	response, err = impl.DeleteDeploymentAppsForEnvironment(ctx,
		request.EnvId, deleteDeploymentType, request.ExcludeApps, request.IncludeApps, request.UserId)

	if err != nil {
		return nil, err
	}

	// Updating the env id and desired deployment app type received from request in the response
	response.EnvId = request.EnvId
	response.DesiredDeploymentType = request.DesiredDeploymentType
	response.TriggeredPipelines = make([]*bean.CdPipelineTrigger, 0)

	// Update the deployment app type to Helm and toggle deployment_app_created to false in db
	pipelineIds := make([]int, 0, len(response.SuccessfulPipelines))

	for _, item := range response.SuccessfulPipelines {
		pipelineIds = append(pipelineIds, item.PipelineId)
	}

	// If nothing to update in db
	if len(pipelineIds) == 0 {
		return response, nil
	}

	// Get all pipelines
	pipelines, err := impl.pipelineRepository.FindByIdsIn(pipelineIds)
	if err != nil {
		impl.logger.Errorw("failed to fetch pipeline details",
			"ids", pipelineIds,
			"err", err)

		return response, nil
	}

	//Update in db
	err = impl.pipelineRepository.UpdateCdPipelineDeploymentAppInFilter(string(request.DesiredDeploymentType),
		pipelineIds, request.UserId, false, false)
	if err != nil {
		impl.logger.Errorw("failed to update deployment app type in db",
			"pipeline ids", pipelineIds,
			"desired deployment type", request.DesiredDeploymentType,
			"err", err)

		return response, nil
	}

	for _, pipeline := range pipelines {
		deploymentConfig, err := impl.deploymentConfigService.GetAndMigrateConfigIfAbsentForDevtronApps(pipeline.AppId, pipeline.EnvironmentId)
		if err != nil {
			impl.logger.Errorw("error in getting deployment config by appId and envId", "appId", pipeline.AppId, "envId", pipeline.EnvironmentId, "err", err)
			return nil, err
		}
		deploymentConfig.DeploymentAppType = request.DesiredDeploymentType
		deploymentConfig.ReleaseMode = util.PIPELINE_RELEASE_MODE_CREATE //now pipeline release mode will be create
		releaseConfig, err := impl.DeploymentConfigReadService.ParseEnvLevelReleaseConfigForDevtronApp(deploymentConfig, pipeline.AppId, pipeline.EnvironmentId)
		if err != nil {
			impl.logger.Errorw("error in parsing release config", "err", err)
			return response, err
		}
		deploymentConfig.ReleaseConfiguration = releaseConfig
		deploymentConfig, err = impl.deploymentConfigService.CreateOrUpdateConfig(nil, deploymentConfig, request.UserId)
		if err != nil {
			impl.logger.Errorw("error in updating configs", "err", err)
			return nil, err
		}
	}

	if !request.AutoTriggerDeployment {
		return response, nil
	}

	// Bulk trigger all the successfully changed pipelines (async)
	bulkTriggerRequest := make([]*bean2.BulkTriggerRequest, 0)

	for _, pipeline := range pipelines {

		artifactsListingFilterOptions := &bean5.ArtifactsListFilterOptions{
			Limit:        10,
			Offset:       0,
			SearchString: "",
		}
		artifactDetails, err := impl.appArtifactManager.RetrieveArtifactsByCDPipelineV2(pipeline, bean5.CD_WORKFLOW_TYPE_DEPLOY, artifactsListingFilterOptions)
		if err != nil {
			impl.logger.Errorw("failed to fetch artifact details for cd pipeline",
				"pipelineId", pipeline.Id,
				"appId", pipeline.AppId,
				"envId", pipeline.EnvironmentId,
				"err", err)

			return response, nil
		}

		if artifactDetails.LatestWfArtifactId == 0 || artifactDetails.LatestWfArtifactStatus == "" {
			continue
		}

		bulkTriggerRequest = append(bulkTriggerRequest, &bean2.BulkTriggerRequest{
			CiArtifactId: artifactDetails.LatestWfArtifactId,
			PipelineId:   pipeline.Id,
		})
		response.TriggeredPipelines = append(response.TriggeredPipelines, &bean.CdPipelineTrigger{
			CiArtifactId: artifactDetails.LatestWfArtifactId,
			PipelineId:   pipeline.Id,
		})
	}

	// pg panics if empty slice is passed as an argument
	if len(bulkTriggerRequest) == 0 {
		return response, nil
	}

	// Trigger
	_, err = impl.workflowEventPublishService.TriggerBulkDeploymentAsync(bulkTriggerRequest, request.UserId)

	if err != nil {
		impl.logger.Errorw("failed to bulk trigger cd pipelines with error: "+err.Error(),
			"err", err)
	}
	return response, nil
}

func (impl *AppDeploymentTypeChangeManagerImpl) ChangePipelineDeploymentType(ctx context.Context,
	request *bean.DeploymentAppTypeChangeRequest, userMetadata *userBean.UserMetadata) (*bean.DeploymentAppTypeChangeResponse, error) {

	response := &bean.DeploymentAppTypeChangeResponse{
		EnvId:                 request.EnvId,
		DesiredDeploymentType: request.DesiredDeploymentType,
		TriggeredPipelines:    make([]*bean.CdPipelineTrigger, 0),
		FailedPipelines:       make([]*bean.DeploymentChangeStatus, 0),
	}

	var deleteDeploymentType string

	if request.DesiredDeploymentType == bean3.ArgoCd {
		deleteDeploymentType = bean3.Helm
	} else {
		deleteDeploymentType = bean3.ArgoCd
	}

	pipelines, err := impl.pipelineRepository.FindActiveByEnvIdAndDeploymentType(request.EnvId,
		deleteDeploymentType, request.ExcludeApps, request.IncludeApps)

	if err != nil {
		impl.logger.Errorw("Error fetching cd pipelines",
			"environmentId", request.EnvId,
			"currentDeploymentAppTypes", deleteDeploymentType,
			"err", err)
		return response, err
	}

	var allPipelines []int
	for _, item := range pipelines {
		allPipelines = append(allPipelines, item.Id)
	}

	if len(allPipelines) == 0 {
		return response, nil
	}

	pipelineIds := make([]int, 0)
	if len(allPipelines) == 0 {
		return response, nil
	}

	deploymentConfigs := make([]*bean4.DeploymentConfig, 0)
	for _, p := range pipelines {
		envDeploymentConfig, err := impl.deploymentConfigService.GetAndMigrateConfigIfAbsentForDevtronApps(p.AppId, p.EnvironmentId)
		if err != nil {
			impl.logger.Errorw("error in fetching environment deployment config by appId and envId", "appId", p.AppId, "envId", p.EnvironmentId, "err", err)
			return response, err
		}
		if !envDeploymentConfig.IsLinkedRelease() {
			pipelineIds = append(pipelineIds, p.Id)
			deploymentConfigs = append(deploymentConfigs, envDeploymentConfig)
		} else {
			response.FailedPipelines = append(response.FailedPipelines,
				&bean.DeploymentChangeStatus{
					PipelineId: p.Id,
					AppId:      p.AppId,
					AppName:    p.App.AppName,
					EnvId:      p.EnvironmentId,
					EnvName:    p.Environment.Name,
					Error:      "Deployment app type cannot be changed because this app is linked with external application",
					Status:     bean.Failed,
				})
		}
	}

	deleteResponse := impl.DeleteDeploymentApps(ctx, pipelines, deploymentConfigs, request.UserId)

	response.SuccessfulPipelines = deleteResponse.SuccessfulPipelines
	response.FailedPipelines = deleteResponse.FailedPipelines

	var cdPipelineIds []int
	for _, item := range response.SuccessfulPipelines {
		cdPipelineIds = append(cdPipelineIds, item.PipelineId)
	}

	if len(cdPipelineIds) == 0 {
		return response, nil
	}

	err = impl.pipelineRepository.UpdateCdPipelineDeploymentAppInFilter(string(request.DesiredDeploymentType),
		cdPipelineIds, request.UserId, false, false)

	if err != nil {
		impl.logger.Errorw("failed to update deployment app type in db",
			"pipeline ids", cdPipelineIds,
			"desired deployment type", request.DesiredDeploymentType,
			"err", err)

		return response, nil
	}

	for _, item := range response.SuccessfulPipelines {
		envDeploymentConfig, err := impl.deploymentConfigService.GetConfigForDevtronApps(item.AppId, item.EnvId)
		if err != nil {
			impl.logger.Errorw("error in fetching environment deployment config by appId and envId", "appId", item.AppId, "envId", item.EnvId, "err", err)
			return response, err
		}
		envDeploymentConfig.ReleaseMode = util.PIPELINE_RELEASE_MODE_CREATE // now pipeline release mode will be create
		envDeploymentConfig.DeploymentAppType = request.DesiredDeploymentType
		releaseConfig, err := impl.DeploymentConfigReadService.ParseEnvLevelReleaseConfigForDevtronApp(envDeploymentConfig, item.AppId, item.EnvId)
		if err != nil {
			impl.logger.Errorw("error in parsing release config", "err", err)
			return response, err
		}
		envDeploymentConfig.ReleaseConfiguration = releaseConfig
		envDeploymentConfig, err = impl.deploymentConfigService.CreateOrUpdateConfig(nil, envDeploymentConfig, request.UserId)
		if err != nil {
			impl.logger.Errorw("error in updating deployment config", "err", err)
			return nil, err
		}
	}

	return response, nil
}

func (impl *AppDeploymentTypeChangeManagerImpl) TriggerDeploymentAfterTypeChange(ctx context.Context,
	request *bean.DeploymentAppTypeChangeRequest, userMetadata *userBean.UserMetadata) (*bean.DeploymentAppTypeChangeResponse, error) {

	response := &bean.DeploymentAppTypeChangeResponse{
		EnvId:                 request.EnvId,
		DesiredDeploymentType: request.DesiredDeploymentType,
		TriggeredPipelines:    make([]*bean.CdPipelineTrigger, 0),
	}
	var err error

	cdPipelines, err := impl.pipelineRepository.FindActiveByEnvIdAndDeploymentType(request.EnvId,
		request.DesiredDeploymentType, request.ExcludeApps, request.IncludeApps)

	if err != nil {
		impl.logger.Errorw("Error fetching cd pipelines",
			"environmentId", request.EnvId,
			"desiredDeploymentAppType", string(request.DesiredDeploymentType),
			"err", err)
		return response, err
	}

	var cdPipelineIds []int
	for _, item := range cdPipelines {
		cdPipelineIds = append(cdPipelineIds, item.Id)
	}

	if len(cdPipelineIds) == 0 {
		return response, nil
	}

	deleteResponse := impl.fetchDeletedApp(ctx, cdPipelines)

	response.SuccessfulPipelines = deleteResponse.SuccessfulPipelines
	response.FailedPipelines = deleteResponse.FailedPipelines

	var successPipelines []int
	for _, item := range response.SuccessfulPipelines {
		successPipelines = append(successPipelines, item.PipelineId)
	}

	bulkTriggerRequest := make([]*bean2.BulkTriggerRequest, 0)

	pipelineIds := make([]int, 0, len(response.SuccessfulPipelines))
	for _, item := range response.SuccessfulPipelines {
		pipelineIds = append(pipelineIds, item.PipelineId)
	}

	pipelines, err := impl.pipelineRepository.FindByIdsIn(pipelineIds)
	if err != nil {
		impl.logger.Errorw("failed to fetch pipeline details",
			"ids", pipelineIds,
			"err", err)

		return response, nil
	}

	for _, pipeline := range pipelines {

		artifactsListingFilterOptions := &bean5.ArtifactsListFilterOptions{
			Limit:        10,
			Offset:       0,
			SearchString: "",
		}
		artifactDetails, err := impl.appArtifactManager.RetrieveArtifactsByCDPipelineV2(pipeline, bean5.CD_WORKFLOW_TYPE_DEPLOY, artifactsListingFilterOptions)
		if err != nil {
			impl.logger.Errorw("failed to fetch artifact details for cd pipeline",
				"pipelineId", pipeline.Id,
				"appId", pipeline.AppId,
				"envId", pipeline.EnvironmentId,
				"err", err)

			return response, nil
		}

		if artifactDetails.LatestWfArtifactId == 0 || artifactDetails.LatestWfArtifactStatus == "" {
			continue
		}

		bulkTriggerRequest = append(bulkTriggerRequest, &bean2.BulkTriggerRequest{
			CiArtifactId: artifactDetails.LatestWfArtifactId,
			PipelineId:   pipeline.Id,
		})
		response.TriggeredPipelines = append(response.TriggeredPipelines, &bean.CdPipelineTrigger{
			CiArtifactId: artifactDetails.LatestWfArtifactId,
			PipelineId:   pipeline.Id,
		})
	}

	if len(bulkTriggerRequest) == 0 {
		return response, nil
	}

	_, err = impl.workflowEventPublishService.TriggerBulkDeploymentAsync(bulkTriggerRequest, request.UserId)

	if err != nil {
		impl.logger.Errorw("failed to bulk trigger cd pipelines with error: "+err.Error(),
			"err", err)
	}

	for _, p := range pipelines {
		envDeploymentConfig, err := impl.deploymentConfigService.GetAndMigrateConfigIfAbsentForDevtronApps(p.AppId, p.EnvironmentId)
		if err != nil {
			impl.logger.Errorw("error in fetching environment deployment config by appId and envId", "appId", p.AppId, "envId", p.EnvironmentId, "err", err)
			return response, err
		}
		envDeploymentConfig.DeploymentAppType = request.DesiredDeploymentType
		envDeploymentConfig, err = impl.deploymentConfigService.CreateOrUpdateConfig(nil, envDeploymentConfig, request.UserId)
		if err != nil {
			impl.logger.Errorw("error in updating configs", "err", err)
			return nil, err
		}
	}

	err = impl.pipelineRepository.UpdateCdPipelineAfterDeployment(string(request.DesiredDeploymentType),
		successPipelines, request.UserId, false)

	if err != nil {
		impl.logger.Errorw("failed to update cd pipelines with error: : "+err.Error(),
			"err", err)
	}

	return response, nil
}

func (impl *AppDeploymentTypeChangeManagerImpl) DeleteDeploymentApps(ctx context.Context,
	pipelines []*pipelineConfig.Pipeline, deploymentConfig []*bean4.DeploymentConfig, userId int32) *bean.DeploymentAppTypeChangeResponse {

	successfulPipelines := make([]*bean.DeploymentChangeStatus, 0)
	failedPipelines := make([]*bean.DeploymentChangeStatus, 0)

	gitOpsConfigurationStatus, gitOpsConfigErr := impl.gitOpsConfigReadService.IsGitOpsConfigured()

	deploymentConfigMap := make(map[bean4.UniqueDeploymentConfigIdentifier]*bean4.DeploymentConfig)
	for _, c := range deploymentConfig {
		deploymentConfigMap[bean4.GetConfigUniqueIdentifier(c.AppId, c.EnvironmentId)] = c
	}

	// Iterate over all the pipelines in the environment for given deployment app type
	for _, pipeline := range pipelines {

		envDeploymentConfig := deploymentConfigMap[bean4.GetConfigUniqueIdentifier(pipeline.AppId, pipeline.EnvironmentId)]

		var isValid bool
		// check if pipeline info like app name and environment is empty or not
		if failedPipelines, isValid = impl.isPipelineInfoValid(pipeline, failedPipelines); !isValid {
			continue
		}

		var healthChkErr error
		// check health of the app if it is argocd deployment type
		if _, healthChkErr = impl.handleNotDeployedAppsIfArgoDeploymentType(pipeline, failedPipelines, envDeploymentConfig); healthChkErr != nil {

			// cannot delete unhealthy app
			continue
		}

		// delete request
		var err error
		if envDeploymentConfig.DeploymentAppType == bean3.ArgoCd {
			err = impl.deleteArgoCdApp(ctx, pipeline, envDeploymentConfig, true)

		} else {

			// For converting from Helm to ArgoCD, GitOps should be configured
			if gitOpsConfigErr != nil || !gitOpsConfigurationStatus.IsGitOpsConfiguredAndArgoCdInstalled() {
				err = errors.New("GitOps integration is not installed/configured. Please install/configure GitOps.")

			} else {
				// Register app in ACD
				var (
					chartGitAttr *commonBean.ChartGitAttribute
					AcdRegisterErr, RepoURLUpdateErr,
					createGitRepoErr, gitOpsRepoNotFound error
				)
				chart, chartServiceErr := impl.chartReadService.FindLatestChartForAppByAppId(pipeline.AppId)
				if chartServiceErr != nil {
					impl.logger.Errorw("Error in fetching latest chart for pipeline", "err", err, "appId", pipeline.AppId)
				}
				if chartServiceErr == nil {
					if gitOps.IsGitOpsRepoNotConfigured(chart.GitRepoUrl) {
						if gitOpsConfigurationStatus.AllowCustomRepository || chart.IsCustomGitRepository {
							gitOpsRepoNotFound = fmt.Errorf(cdWorkflow.GITOPS_REPO_NOT_CONFIGURED)
						} else {
							targetRevision := chart.TargetRevision
							_, chartGitAttr, createGitRepoErr = impl.appService.CreateGitOpsRepo(&app.App{Id: pipeline.AppId, AppName: pipeline.App.AppName}, targetRevision, userId)
							if createGitRepoErr == nil {
								AcdRegisterErr = impl.cdPipelineConfigService.RegisterInACD(ctx, chartGitAttr, userId)
								if AcdRegisterErr != nil {
									impl.logger.Errorw("error in registering acd app", "err", AcdRegisterErr)
								}
								if AcdRegisterErr == nil {
									_, RepoURLUpdateErr = impl.chartService.ConfigureGitOpsRepoUrlForApp(pipeline.AppId, chartGitAttr.RepoUrl, chartGitAttr.ChartLocation, false, userId)
									if RepoURLUpdateErr != nil {
										impl.logger.Errorw("error in updating git repo url in charts", "err", RepoURLUpdateErr)
									}
									envDeploymentConfig.ConfigType = adapter.GetDeploymentConfigType(chart.IsCustomGitRepository)
									envDeploymentConfig.SetRepoURL(chartGitAttr.RepoUrl)

									envDeploymentConfig, RepoURLUpdateErr = impl.deploymentConfigService.CreateOrUpdateConfig(nil, envDeploymentConfig, userId)
									if RepoURLUpdateErr != nil {
										impl.logger.Errorw("error in saving deployment config for app", "appId", pipeline.AppId, "envId", pipeline.EnvironmentId, "err", err)
									}
								}
							}
						}
					} else {
						// in this case user has already created an empty git repository and provided us gitRepoUrl
						chartGitAttr = &commonBean.ChartGitAttribute{
							RepoUrl:        chart.GitRepoUrl,
							TargetRevision: chart.TargetRevision,
						}
					}
				}

				if gitOpsRepoNotFound != nil {
					impl.logger.Errorw("error no GitOps repository configured for the app", "err", gitOpsRepoNotFound)
				}
				if createGitRepoErr != nil {
					impl.logger.Errorw("error in creating git repo", "err", createGitRepoErr)
				}

				if chartServiceErr != nil {
					err = chartServiceErr
				} else if gitOpsRepoNotFound != nil {
					err = gitOpsRepoNotFound
				} else if createGitRepoErr != nil {
					err = createGitRepoErr
				} else if AcdRegisterErr != nil {
					err = AcdRegisterErr
				} else if RepoURLUpdateErr != nil {
					err = RepoURLUpdateErr
				}
			}
			if err != nil {
				impl.logger.Errorw("error registering app on ACD with error: "+err.Error(),
					"deploymentAppName", pipeline.DeploymentAppName,
					"envId", pipeline.EnvironmentId,
					"appId", pipeline.AppId,
					"err", err)

				// deletion failed, append to the list of failed pipelines
				failedPipelines = impl.handleFailedDeploymentAppChange(pipeline, failedPipelines,
					"failed to register app on ACD with error: "+err.Error())

				continue
			}
			err = impl.deleteHelmApp(ctx, pipeline, envDeploymentConfig.DeploymentAppType)
		}

		if err != nil {
			impl.logger.Errorw("error deleting app on "+envDeploymentConfig.DeploymentAppType,
				"deployment app name", pipeline.DeploymentAppName,
				"err", err)

			// deletion failed, append to the list of failed pipelines
			failedPipelines = impl.handleFailedDeploymentAppChange(pipeline, failedPipelines,
				"error deleting app with error: "+err.Error())

			continue
		}

		// deletion successful, append to the list of successful pipelines
		successfulPipelines = impl.appendToDeploymentChangeStatusList(
			successfulPipelines,
			pipeline,
			"",
			bean.INITIATED)
	}

	return &bean.DeploymentAppTypeChangeResponse{
		SuccessfulPipelines: successfulPipelines,
		FailedPipelines:     failedPipelines,
	}
}

func (impl *AppDeploymentTypeChangeManagerImpl) DeleteDeploymentAppsForEnvironment(ctx context.Context, environmentId int,
	currentDeploymentAppType bean3.DeploymentType, exclusionList []int, includeApps []int, userId int32) (*bean.DeploymentAppTypeChangeResponse, error) {

	// fetch active pipelines from database for the given environment id and current deployment app type
	allPipelines, err := impl.pipelineRepository.FindActiveByEnvIdAndDeploymentType(environmentId,
		currentDeploymentAppType, exclusionList, includeApps)

	pipelines := make([]*pipelineConfig.Pipeline, 0)
	deploymentConfigs := make([]*bean4.DeploymentConfig, 0)
	for _, p := range allPipelines {
		envDeploymentConfig, err := impl.deploymentConfigService.GetAndMigrateConfigIfAbsentForDevtronApps(p.AppId, p.EnvironmentId)
		if err != nil {
			impl.logger.Errorw("error in fetching environment deployment config by appId and envId", "appId", p.AppId, "envId", p.EnvironmentId, "err", err)
			return nil, err
		}
		if !envDeploymentConfig.IsLinkedRelease() {
			pipelines = append(pipelines, p)
			deploymentConfigs = append(deploymentConfigs, envDeploymentConfig)
		}
	}

	if err != nil {
		impl.logger.Errorw("Error fetching cd pipelines",
			"environmentId", environmentId,
			"currentDeploymentAppType", currentDeploymentAppType,
			"err", err)

		return &bean.DeploymentAppTypeChangeResponse{
			EnvId:               environmentId,
			SuccessfulPipelines: []*bean.DeploymentChangeStatus{},
			FailedPipelines:     []*bean.DeploymentChangeStatus{},
		}, err
	}

	// Currently deleting apps only in argocd is supported
	return impl.DeleteDeploymentApps(ctx, pipelines, deploymentConfigs, userId), nil
}

func (impl *AppDeploymentTypeChangeManagerImpl) isPipelineInfoValid(pipeline *pipelineConfig.Pipeline,
	failedPipelines []*bean.DeploymentChangeStatus) ([]*bean.DeploymentChangeStatus, bool) {

	if len(pipeline.App.AppName) == 0 || len(pipeline.Environment.Name) == 0 {
		impl.logger.Errorw("app name or environment name is not present",
			"pipeline id", pipeline.Id)

		failedPipelines = impl.handleFailedDeploymentAppChange(pipeline, failedPipelines,
			"could not fetch app name or environment name")

		return failedPipelines, false
	}
	return failedPipelines, true
}

func (impl *AppDeploymentTypeChangeManagerImpl) handleNotHealthyAppsIfArgoDeploymentType(pipeline *pipelineConfig.Pipeline, deploymentConfig *bean4.DeploymentConfig,
	failedPipelines []*bean.DeploymentChangeStatus) ([]*bean.DeploymentChangeStatus, error) {

	if deploymentConfig.DeploymentAppType == bean3.ArgoCd {
		// check if app status is Healthy
		status, err := impl.appStatusRepository.Get(pipeline.AppId, pipeline.EnvironmentId)

		// case: missing status row in db
		if len(status.Status) == 0 {
			return failedPipelines, nil
		}

		// cannot delete the app from argocd if app status is Progressing
		if err != nil || status.Status == "Progressing" {

			healthCheckErr := errors.New("unable to fetch app status or app status is progressing")

			impl.logger.Errorw(healthCheckErr.Error(),
				"appId", pipeline.AppId,
				"environmentId", pipeline.EnvironmentId,
				"err", err)

			failedPipelines = impl.handleFailedDeploymentAppChange(pipeline, failedPipelines, healthCheckErr.Error())

			return failedPipelines, healthCheckErr
		}
		return failedPipelines, nil
	}
	return failedPipelines, nil
}

func (impl *AppDeploymentTypeChangeManagerImpl) handleNotDeployedAppsIfArgoDeploymentType(pipeline *pipelineConfig.Pipeline,
	failedPipelines []*bean.DeploymentChangeStatus, deploymentConfig *bean4.DeploymentConfig) ([]*bean.DeploymentChangeStatus, error) {

	if deploymentConfig.DeploymentAppType == string(bean3.ArgoCd) {
		// check if app status is Healthy
		status, err := impl.appStatusRepository.Get(pipeline.AppId, pipeline.EnvironmentId)

		// case: missing status row in db
		if len(status.Status) == 0 {
			return failedPipelines, nil
		}

		// cannot delete the app from argocd if app status is Progressing
		if err != nil {

			healthCheckErr := errors.New("unable to fetch app status")

			impl.logger.Errorw(healthCheckErr.Error(),
				"appId", pipeline.AppId,
				"environmentId", pipeline.EnvironmentId,
				"err", err)

			failedPipelines = impl.handleFailedDeploymentAppChange(pipeline, failedPipelines, healthCheckErr.Error())

			return failedPipelines, healthCheckErr
		}
		return failedPipelines, nil
	}
	return failedPipelines, nil
}

func (impl *AppDeploymentTypeChangeManagerImpl) handleFailedDeploymentAppChange(pipeline *pipelineConfig.Pipeline,
	failedPipelines []*bean.DeploymentChangeStatus, err string) []*bean.DeploymentChangeStatus {

	return impl.appendToDeploymentChangeStatusList(
		failedPipelines,
		pipeline,
		err,
		bean.Failed)
}

func (impl *AppDeploymentTypeChangeManagerImpl) fetchDeletedApp(ctx context.Context,
	pipelines []*pipelineConfig.Pipeline) *bean.DeploymentAppTypeChangeResponse {

	successfulPipelines := make([]*bean.DeploymentChangeStatus, 0)
	failedPipelines := make([]*bean.DeploymentChangeStatus, 0)
	// Iterate over all the pipelines in the environment for given deployment app type
	for _, pipeline := range pipelines {
		envDeploymentConfig, err := impl.deploymentConfigService.GetConfigForDevtronApps(pipeline.AppId, pipeline.EnvironmentId)
		if err != nil {
			impl.logger.Errorw("error in fetching environment deployment config by appId and envId", "appId", pipeline.AppId, "envId", pipeline.EnvironmentId, "err", err)
		}
		if envDeploymentConfig.DeploymentAppType == bean3.ArgoCd {
			appIdentifier := &helmBean.AppIdentifier{
				ClusterId:   pipeline.Environment.ClusterId,
				ReleaseName: pipeline.DeploymentAppName,
				Namespace:   pipeline.Environment.Namespace,
			}
			_, err = impl.helmAppService.GetApplicationDetail(ctx, appIdentifier)
		} else {
			_, err = impl.ArgoClientWrapperService.GetArgoAppByName(ctx, pipeline.DeploymentAppName)
		}
		if err != nil {
			impl.logger.Errorw("error in getting application detail", "err", err, "deploymentAppName", pipeline.DeploymentAppName)
		}

		if err != nil && CheckAppReleaseNotExist(err) {
			successfulPipelines = impl.appendToDeploymentChangeStatusList(
				successfulPipelines,
				pipeline,
				"",
				bean.Success)
		} else {
			failedPipelines = impl.appendToDeploymentChangeStatusList(
				failedPipelines,
				pipeline,
				"App Not Yet Deleted.",
				bean.NOT_YET_DELETED)
		}
	}

	return &bean.DeploymentAppTypeChangeResponse{
		SuccessfulPipelines: successfulPipelines,
		FailedPipelines:     failedPipelines,
	}
}

// deleteArgoCdApp takes context and deployment app name used in argo cd and deletes
// the application in argo cd.
func (impl *AppDeploymentTypeChangeManagerImpl) deleteArgoCdApp(ctx context.Context, pipeline *pipelineConfig.Pipeline, envDeploymentConfig *bean4.DeploymentConfig,
	cascadeDelete bool) error {
	if !pipeline.DeploymentAppCreated {
		return nil
	}
	var err error
	applicationObjectClusterId := envDeploymentConfig.GetApplicationObjectClusterId()
	applicationNamespace := envDeploymentConfig.GetApplicationObjectNamespace()
	err = impl.ArgoClientWrapperService.DeleteArgoAppWithK8sClient(ctx, applicationObjectClusterId, applicationNamespace, pipeline.DeploymentAppName, cascadeDelete)
	if err != nil {
		impl.logger.Errorw("error in deleting argocd application", "err", err)
		// Possible that argocd app got deleted but db updation failed
		if errors2.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

func (impl *AppDeploymentTypeChangeManagerImpl) appendToDeploymentChangeStatusList(pipelines []*bean.DeploymentChangeStatus,
	pipeline *pipelineConfig.Pipeline, error string, status bean.Status) []*bean.DeploymentChangeStatus {

	return append(pipelines, &bean.DeploymentChangeStatus{
		PipelineId: pipeline.Id,
		AppId:      pipeline.AppId,
		AppName:    pipeline.App.AppName,
		EnvId:      pipeline.EnvironmentId,
		EnvName:    pipeline.Environment.Name,
		Error:      error,
		Status:     status,
	})
}

// deleteHelmApp takes in context and pipeline object and deletes the release in helm
func (impl *AppDeploymentTypeChangeManagerImpl) deleteHelmApp(ctx context.Context, pipeline *pipelineConfig.Pipeline, deploymentAppType string) error {

	if !pipeline.DeploymentAppCreated {
		return nil
	}

	// validation
	if !util.IsHelmApp(deploymentAppType) {
		return errors.New("unable to delete pipeline with id: " + strconv.Itoa(pipeline.Id) + ", not a helm app")
	}

	// create app identifier
	appIdentifier := &helmBean.AppIdentifier{
		ClusterId:   pipeline.Environment.ClusterId,
		ReleaseName: pipeline.DeploymentAppName,
		Namespace:   pipeline.Environment.Namespace,
	}

	// call for delete resource
	deleteResponse, err := impl.helmAppService.DeleteApplication(ctx, appIdentifier)

	if err != nil {
		impl.logger.Errorw("error in deleting helm application", "error", err, "appIdentifier", appIdentifier)
		apiError := clientErrors.ConvertToApiError(err)
		if apiError != nil {
			err = apiError
		}
		return err
	}

	if deleteResponse == nil || !deleteResponse.GetSuccess() {
		return errors.New("helm delete application response unsuccessful")
	}
	return nil
}
