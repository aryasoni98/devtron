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

package configure

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/devtron-labs/devtron/pkg/build/artifacts/imageTagging"
	imageTaggingRead "github.com/devtron-labs/devtron/pkg/build/artifacts/imageTagging/read"
	read2 "github.com/devtron-labs/devtron/pkg/build/git/gitMaterial/read"
	gitProviderRead "github.com/devtron-labs/devtron/pkg/build/git/gitProvider/read"
	bean3 "github.com/devtron-labs/devtron/pkg/build/pipeline/bean"
	"github.com/devtron-labs/devtron/pkg/build/trigger"
	"github.com/devtron-labs/devtron/pkg/chart/gitOpsConfig"
	read5 "github.com/devtron-labs/devtron/pkg/chart/read"
	repository2 "github.com/devtron-labs/devtron/pkg/cluster/environment/repository"
	"github.com/devtron-labs/devtron/pkg/deployment/manifest/deployedAppMetrics"
	"github.com/devtron-labs/devtron/pkg/deployment/manifest/deploymentTemplate/chartRef"
	validator2 "github.com/devtron-labs/devtron/pkg/deployment/manifest/deploymentTemplate/validator"
	"github.com/devtron-labs/devtron/pkg/deployment/trigger/devtronApps"
	"github.com/devtron-labs/devtron/pkg/pipeline/draftAwareConfigService"
	security2 "github.com/devtron-labs/devtron/pkg/policyGovernance/security/imageScanning"
	"github.com/devtron-labs/devtron/pkg/policyGovernance/security/imageScanning/read"
	read3 "github.com/devtron-labs/devtron/pkg/team/read"
	"github.com/devtron-labs/devtron/util/beHelper"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/caarlos0/env"
	"github.com/devtron-labs/devtron/api/restHandler/common"
	"github.com/devtron-labs/devtron/client/gitSensor"
	"github.com/devtron-labs/devtron/pkg/auth/authorisation/casbin"
	"github.com/devtron-labs/devtron/pkg/auth/user"
	"github.com/devtron-labs/devtron/pkg/chart"
	"github.com/devtron-labs/devtron/pkg/generateManifest"
	resourceGroup2 "github.com/devtron-labs/devtron/pkg/resourceGroup"
	"github.com/go-pg/pg"
	"go.opentelemetry.io/otel"

	"github.com/devtron-labs/devtron/internal/sql/repository"
	"github.com/devtron-labs/devtron/internal/sql/repository/pipelineConfig"
	"github.com/devtron-labs/devtron/internal/util"
	"github.com/devtron-labs/devtron/pkg/appClone"
	"github.com/devtron-labs/devtron/pkg/appWorkflow"
	"github.com/devtron-labs/devtron/pkg/bean"
	"github.com/devtron-labs/devtron/pkg/pipeline"
	"github.com/devtron-labs/devtron/pkg/team"
	"github.com/devtron-labs/devtron/util/rbac"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"gopkg.in/go-playground/validator.v9"
)

type PipelineRestHandlerEnvConfig struct {
	UseArtifactListApiV2 bool `env:"USE_ARTIFACT_LISTING_API_V2" envDefault:"true" description:"To use the V2 API for listing artifacts in Listing the images in pipeline"` //deprecated
}

type DevtronAppRestHandler interface {
	CreateApp(w http.ResponseWriter, r *http.Request)
	DeleteApp(w http.ResponseWriter, r *http.Request)
	DeleteACDAppWithNonCascade(w http.ResponseWriter, r *http.Request)
	GetApp(w http.ResponseWriter, r *http.Request)
	FindAppsByTeamId(w http.ResponseWriter, r *http.Request)
	FindAppsByTeamName(w http.ResponseWriter, r *http.Request)
	GetEnvironmentListWithAppData(w http.ResponseWriter, r *http.Request)
	GetApplicationsByEnvironment(w http.ResponseWriter, r *http.Request)
}

type DevtronAppWorkflowRestHandler interface {
	FetchAppWorkflowStatusForTriggerView(w http.ResponseWriter, r *http.Request)
	FetchAppWorkflowStatusForTriggerViewByEnvironment(w http.ResponseWriter, r *http.Request)
	FetchAppDeploymentStatusForEnvironments(w http.ResponseWriter, r *http.Request)
}

type PipelineConfigRestHandler interface {
	DevtronAppRestHandler
	DevtronAppWorkflowRestHandler
	DevtronAppBuildRestHandler
	DevtronAppBuildMaterialRestHandler
	DevtronAppBuildHistoryRestHandler
	DevtronAppDeploymentRestHandler
	DevtronAppDeploymentHistoryRestHandler
	DevtronAppPrePostDeploymentRestHandler
	DevtronAppDeploymentConfigRestHandler
	ImageTaggingRestHandler
	PipelineNameSuggestion(w http.ResponseWriter, r *http.Request)
}

type PipelineConfigRestHandlerImpl struct {
	pipelineBuilder                     pipeline.PipelineBuilder
	ciPipelineRepository                pipelineConfig.CiPipelineRepository
	ciPipelineMaterialRepository        pipelineConfig.CiPipelineMaterialRepository
	ciHandler                           pipeline.CiHandler
	Logger                              *zap.SugaredLogger
	deploymentTemplateValidationService validator2.DeploymentTemplateValidationService
	chartService                        chart.ChartService
	devtronAppGitOpConfigService        gitOpsConfig.DevtronAppGitOpConfigService
	propertiesConfigService             pipeline.PropertiesConfigService
	userAuthService                     user.UserService
	validator                           *validator.Validate
	teamService                         team.TeamService
	enforcer                            casbin.Enforcer
	gitSensorClient                     gitSensor.Client
	pipelineRepository                  pipelineConfig.PipelineRepository
	appWorkflowService                  appWorkflow.AppWorkflowService
	enforcerUtil                        rbac.EnforcerUtil
	dockerRegistryConfig                pipeline.DockerRegistryConfig
	cdHandler                           pipeline.CdHandler
	appCloneService                     appClone.AppCloneService
	gitMaterialReadService              read2.GitMaterialReadService
	policyService                       security2.PolicyService
	imageScanResultReadService          read.ImageScanResultReadService
	gitProviderReadService              gitProviderRead.GitProviderReadService
	imageTaggingReadService             imageTaggingRead.ImageTaggingReadService
	imageTaggingService                 imageTagging.ImageTaggingService
	deploymentTemplateService           generateManifest.DeploymentTemplateService
	pipelineRestHandlerEnvConfig        *PipelineRestHandlerEnvConfig
	ciArtifactRepository                repository.CiArtifactRepository
	deployedAppMetricsService           deployedAppMetrics.DeployedAppMetricsService
	chartRefService                     chartRef.ChartRefService
	ciCdPipelineOrchestrator            pipeline.CiCdPipelineOrchestrator
	teamReadService                     read3.TeamReadService
	environmentRepository               repository2.EnvironmentRepository
	chartReadService                    read5.ChartReadService
	draftAwareResourceService           draftAwareConfigService.DraftAwareConfigService
	ciHandlerService                    trigger.HandlerService
	cdHandlerService                    devtronApps.HandlerService
}

func NewPipelineRestHandlerImpl(pipelineBuilder pipeline.PipelineBuilder, Logger *zap.SugaredLogger,
	deploymentTemplateValidationService validator2.DeploymentTemplateValidationService,
	chartService chart.ChartService,
	devtronAppGitOpConfigService gitOpsConfig.DevtronAppGitOpConfigService,
	propertiesConfigService pipeline.PropertiesConfigService,
	userAuthService user.UserService,
	teamService team.TeamService,
	enforcer casbin.Enforcer,
	ciHandler pipeline.CiHandler,
	validator *validator.Validate,
	gitSensorClient gitSensor.Client,
	ciPipelineRepository pipelineConfig.CiPipelineRepository,
	pipelineRepository pipelineConfig.PipelineRepository,
	enforcerUtil rbac.EnforcerUtil,
	dockerRegistryConfig pipeline.DockerRegistryConfig,
	cdHandler pipeline.CdHandler,
	appCloneService appClone.AppCloneService,
	deploymentTemplateService generateManifest.DeploymentTemplateService,
	appWorkflowService appWorkflow.AppWorkflowService,
	gitMaterialReadService read2.GitMaterialReadService, policyService security2.PolicyService,
	imageScanResultReadService read.ImageScanResultReadService,
	ciPipelineMaterialRepository pipelineConfig.CiPipelineMaterialRepository,
	imageTaggingReadService imageTaggingRead.ImageTaggingReadService,
	imageTaggingService imageTagging.ImageTaggingService,
	ciArtifactRepository repository.CiArtifactRepository,
	deployedAppMetricsService deployedAppMetrics.DeployedAppMetricsService,
	chartRefService chartRef.ChartRefService,
	ciCdPipelineOrchestrator pipeline.CiCdPipelineOrchestrator,
	gitProviderReadService gitProviderRead.GitProviderReadService,
	teamReadService read3.TeamReadService,
	EnvironmentRepository repository2.EnvironmentRepository,
	chartReadService read5.ChartReadService,
	draftAwareResourceService draftAwareConfigService.DraftAwareConfigService,
	ciHandlerService trigger.HandlerService,
	cdHandlerService devtronApps.HandlerService,
) *PipelineConfigRestHandlerImpl {
	envConfig := &PipelineRestHandlerEnvConfig{}
	err := env.Parse(envConfig)
	if err != nil {
		Logger.Errorw("error in parsing PipelineRestHandlerEnvConfig", "err", err)
	}
	return &PipelineConfigRestHandlerImpl{
		pipelineBuilder:                     pipelineBuilder,
		Logger:                              Logger,
		deploymentTemplateValidationService: deploymentTemplateValidationService,
		chartService:                        chartService,
		devtronAppGitOpConfigService:        devtronAppGitOpConfigService,
		propertiesConfigService:             propertiesConfigService,
		userAuthService:                     userAuthService,
		validator:                           validator,
		teamService:                         teamService,
		enforcer:                            enforcer,
		ciHandler:                           ciHandler,
		gitSensorClient:                     gitSensorClient,
		ciPipelineRepository:                ciPipelineRepository,
		pipelineRepository:                  pipelineRepository,
		enforcerUtil:                        enforcerUtil,
		dockerRegistryConfig:                dockerRegistryConfig,
		cdHandler:                           cdHandler,
		appCloneService:                     appCloneService,
		appWorkflowService:                  appWorkflowService,
		gitMaterialReadService:              gitMaterialReadService,
		policyService:                       policyService,
		imageScanResultReadService:          imageScanResultReadService,
		ciPipelineMaterialRepository:        ciPipelineMaterialRepository,
		imageTaggingReadService:             imageTaggingReadService,
		imageTaggingService:                 imageTaggingService,
		deploymentTemplateService:           deploymentTemplateService,
		pipelineRestHandlerEnvConfig:        envConfig,
		ciArtifactRepository:                ciArtifactRepository,
		deployedAppMetricsService:           deployedAppMetricsService,
		chartRefService:                     chartRefService,
		ciCdPipelineOrchestrator:            ciCdPipelineOrchestrator,
		gitProviderReadService:              gitProviderReadService,
		teamReadService:                     teamReadService,
		environmentRepository:               EnvironmentRepository,
		chartReadService:                    chartReadService,
		draftAwareResourceService:           draftAwareResourceService,
		ciHandlerService:                    ciHandlerService,
		cdHandlerService:                    cdHandlerService,
	}
}

const (
	devtron             = "DEVTRON"
	SSH_URL_PREFIX      = "git@"
	HTTPS_URL_PREFIX    = "https://"
	HTTP_URL_PREFIX     = "http://"
	argoWFLogIdentifier = "argo=true"
)

func (handler *PipelineConfigRestHandlerImpl) DeleteApp(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("token")
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	appId, err := strconv.Atoi(vars["appId"])
	if err != nil {
		handler.Logger.Errorw("request err, delete app", "err", err, "appId", appId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.Logger.Infow("request payload, delete app", "appId", appId)
	wfs, err := handler.appWorkflowService.FindAppWorkflows(appId)
	if err != nil {
		handler.Logger.Errorw("could not fetch wfs", "err", err, "appId", appId)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	if len(wfs) != 0 {
		handler.Logger.Info("cannot delete app with workflow's")
		err = &util.ApiError{Code: "400", HttpStatusCode: 400, UserMessage: "cannot delete app having workflow's"}
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	resourceObject := handler.enforcerUtil.GetAppRBACNameByAppId(appId)
	ok := handler.enforcerUtil.CheckAppRbacForAppOrJob(token, resourceObject, casbin.ActionDelete)
	if !ok {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusForbidden)
		return
	}
	err = handler.pipelineBuilder.DeleteApp(appId, userId)
	if err != nil {
		handler.Logger.Errorw("service error, delete app", "err", err, "appId", appId)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, nil, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) DeleteACDAppWithNonCascade(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("token")
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	appId, err := strconv.Atoi(vars["appId"])
	if err != nil {
		handler.Logger.Errorw("request err, delete app", "err", err, "appId", appId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	envId, err := strconv.Atoi(vars["envId"])
	if err != nil {
		handler.Logger.Errorw("request err, delete app", "err", err, "envId", envId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.Logger.Infow("request payload, delete app", "appId", appId)

	v := r.URL.Query()
	forceDelete := false
	force := v.Get("force")
	if len(force) > 0 {
		forceDelete, err = strconv.ParseBool(force)
		if err != nil {
			handler.Logger.Errorw("request err, NonCascadeDeleteCdPipeline", "err", err)
			common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
			return
		}
	}
	app, err := handler.pipelineBuilder.GetApp(appId)
	if err != nil {
		handler.Logger.Infow("service error, NonCascadeDeleteCdPipeline", "err", err, "appId", appId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	// rbac enforcer applying
	resourceName := handler.enforcerUtil.GetAppRBACName(app.AppName)
	if ok := handler.enforcer.Enforce(token, casbin.ResourceApplications, casbin.ActionGet, resourceName); !ok {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusForbidden)
		return
	}
	object := handler.enforcerUtil.GetEnvRBACNameByAppId(appId, envId)
	if ok := handler.enforcer.Enforce(token, casbin.ResourceEnvironment, casbin.ActionDelete, object); !ok {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusForbidden)
		return
	}
	// rbac enforcer ends

	pipelines, err := handler.pipelineRepository.FindActiveByAppIdAndEnvironmentId(appId, envId)
	if err != nil && err != pg.ErrNoRows {
		handler.Logger.Errorw("error in fetching pipelines from db", "appId", appId, "envId", envId)
		common.WriteJsonResp(w, err, "error in fetching pipelines from db", http.StatusInternalServerError)
		return
	} else if len(pipelines) == 0 {
		common.WriteJsonResp(w, err, "deployment not found, unable to fetch resource tree", http.StatusNotFound)
		return
	} else if len(pipelines) > 1 {
		common.WriteJsonResp(w, err, "multiple pipelines found for an envId", http.StatusBadRequest)
		return
	}
	cdPipeline := pipelines[0]
	err = handler.pipelineBuilder.DeleteACDAppCdPipelineWithNonCascade(cdPipeline, r.Context(), forceDelete, userId)
	if err != nil {
		handler.Logger.Errorw("service err, NonCascadeDeleteCdPipeline", "err", err, "payload", cdPipeline)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, nil, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) CreateApp(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("token")
	decoder := json.NewDecoder(r.Body)
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	var createRequest bean.CreateAppDTO
	err = decoder.Decode(&createRequest)
	createRequest.UserId = userId
	if err != nil {
		handler.Logger.Errorw("request err, CreateApp", "err", err, "CreateApp", createRequest)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.Logger.Infow("request payload, CreateApp", "CreateApp", createRequest)
	err = handler.validator.Struct(createRequest)
	if err != nil {
		handler.Logger.Errorw("validation err, CreateApp", "err", err, "CreateApp", createRequest)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}

	project, err := handler.teamReadService.FindOne(createRequest.TeamId)
	if err != nil {
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}

	// with admin roles, you have to access for all the apps of the project to create new app. (admin or manager with specific app permission can't create app.)
	object := fmt.Sprintf("%s/%s", project.Name, "*")
	isAuthorised := handler.enforcerUtil.CheckAppRbacForAppOrJob(token, object, casbin.ActionCreate)
	if !isAuthorised {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusForbidden)
		return
	}
	// Validation For appName
	if ok := strings.Contains(createRequest.AppName, bean3.UniquePlaceHolderForAppName); ok {
		common.WriteJsonResp(w, err, "app creation failed due to validation on app-name as it contains not allowed place-holder in name", http.StatusBadRequest)
		return
	}

	var createResp *bean.CreateAppDTO
	err = nil
	if createRequest.TemplateId == 0 {
		createResp, err = handler.pipelineBuilder.CreateApp(&createRequest)
	} else {
		ctx, cancel := context.WithCancel(r.Context())
		if cn, ok := w.(http.CloseNotifier); ok {
			go func(done <-chan struct{}, closed <-chan bool) {
				select {
				case <-done:
				case <-closed:
					cancel()
				}
			}(ctx.Done(), cn.CloseNotify())
		}
		createResp, err = handler.appCloneService.CloneApp(&createRequest, ctx)
	}
	if err != nil {
		handler.Logger.Errorw("service err, CreateApp", "err", err, "CreateApp", createRequest)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, createResp, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) GetApp(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("token")
	vars := mux.Vars(r)
	appId, err := strconv.Atoi(vars["appId"])
	if err != nil {
		handler.Logger.Errorw("request err, get app", "err", err, "appId", appId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.Logger.Infow("request payload, get app", "appId", appId)
	ciConf, err := handler.pipelineBuilder.GetApp(appId)
	if err != nil {
		handler.Logger.Errorw("service err, get app", "err", err, "appId", appId)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}

	//rbac implementation starts here
	object := handler.enforcerUtil.GetAppRBACNameByAppId(appId)
	ok := handler.enforcerUtil.CheckAppRbacForAppOrJob(token, object, casbin.ActionGet)
	if !ok {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusForbidden)
		return
	}
	//rbac implementation ends here

	common.WriteJsonResp(w, err, ciConf, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) FindAppsByTeamId(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamId, err := strconv.Atoi(vars["teamId"])
	if err != nil {
		handler.Logger.Errorw("request err, FindAppsByTeamId", "err", err, "teamId", teamId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.Logger.Infow("request payload, FindAppsByTeamId", "teamId", teamId)
	project, err := handler.pipelineBuilder.FindAppsByTeamId(teamId)
	if err != nil {
		handler.Logger.Errorw("service err, FindAppsByTeamId", "err", err, "teamId", teamId)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, project, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) FindAppsByTeamName(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamName := vars["teamName"]
	handler.Logger.Infow("request payload, FindAppsByTeamName", "teamName", teamName)
	project, err := handler.pipelineBuilder.FindAppsByTeamName(teamName)
	if err != nil {
		handler.Logger.Errorw("service err, FindAppsByTeamName", "err", err, "teamName", teamName)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, project, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) streamOutput(w http.ResponseWriter, reader *bufio.Reader, lastSeenMsgId int) {
	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "unexpected server doesnt support streaming", http.StatusInternalServerError)
	}

	// Important to make it work in browsers
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	//var wroteHeader bool
	startOfStream := []byte("START_OF_STREAM")
	endOfStreamEvent := []byte("END_OF_STREAM")
	reconnectEvent := []byte("RECONNECT_STREAM")
	unexpectedEndOfStreamEvent := []byte("UNEXPECTED_END_OF_STREAM")
	streamStarted := false
	msgCounter := 0
	if lastSeenMsgId == -1 {
		handler.sendData(startOfStream, w, msgCounter)
		handler.sendEvent(startOfStream, w)
		f.Flush()
	} else {
		handler.sendEvent(reconnectEvent, w)
		f.Flush()
	}

	for {
		data, err := reader.ReadBytes('\n')
		if err == io.EOF {
			if streamStarted {
				handler.sendData(endOfStreamEvent, w, msgCounter)
				handler.sendEvent(endOfStreamEvent, w)
				f.Flush()
				return
			}
			return
		}
		if err != nil {
			//TODO handle error
			handler.sendData(unexpectedEndOfStreamEvent, w, msgCounter)
			handler.sendEvent(unexpectedEndOfStreamEvent, w)
			f.Flush()
			return
		}
		msgCounter = msgCounter + 1
		//skip for seen msg
		if msgCounter <= lastSeenMsgId {
			continue
		}

		// only skip the logs of argo-wf if found at starting
		isAWFLog := msgCounter == 1 && strings.Contains(string(data), argoWFLogIdentifier)
		if strings.Contains(string(data), devtron) || isAWFLog {
			continue
		}

		var res []byte
		res = append(res, "id:"...)
		res = append(res, fmt.Sprintf("%d\n", msgCounter)...)
		res = append(res, "data:"...)
		res = append(res, data...)
		res = append(res, '\n')

		if _, err = w.Write(res); err != nil {
			//TODO handle error
			handler.Logger.Errorw("Failed to send response chunk, streamOutput", "err", err)
			handler.sendData(unexpectedEndOfStreamEvent, w, msgCounter)
			handler.sendEvent(unexpectedEndOfStreamEvent, w)
			f.Flush()
			return
		}
		streamStarted = true
		f.Flush()
	}
}

func (handler *PipelineConfigRestHandlerImpl) sendEvent(event []byte, w http.ResponseWriter) {
	var res []byte
	res = append(res, "event:"...)
	res = append(res, event...)
	res = append(res, '\n')
	res = append(res, "data:"...)
	res = append(res, '\n', '\n')

	if _, err := w.Write(res); err != nil {
		handler.Logger.Debugf("Failed to send response chunk: %v", err)
		return
	}

}
func (handler *PipelineConfigRestHandlerImpl) sendData(event []byte, w http.ResponseWriter, msgId int) {
	var res []byte
	res = append(res, "id:"...)
	res = append(res, fmt.Sprintf("%d\n", msgId)...)
	res = append(res, "data:"...)
	res = append(res, event...)
	res = append(res, '\n', '\n')
	if _, err := w.Write(res); err != nil {
		handler.Logger.Errorw("Failed to send response chunk, sendData", "err", err)
		return
	}
}

func (handler *PipelineConfigRestHandlerImpl) FetchAppWorkflowStatusForTriggerView(w http.ResponseWriter, r *http.Request) {
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	token := r.Header.Get("token")
	vars := mux.Vars(r)
	appId, err := strconv.Atoi(vars["appId"])
	if err != nil {
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.Logger.Infow("request payload, FetchAppWorkflowStatusForTriggerView", "appId", appId)
	//RBAC CHECK
	resourceName := handler.enforcerUtil.GetAppRBACNameByAppId(appId)
	ok := handler.enforcerUtil.CheckAppRbacForAppOrJob(token, resourceName, casbin.ActionGet)
	if !ok {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusForbidden)
		return
	}
	//RBAC CHECK

	apiVersion := vars["version"]
	triggerWorkflowStatus := pipelineConfig.TriggerWorkflowStatus{}
	var ciWorkflowStatus []*pipelineConfig.CiWorkflowStatus
	var err1 error
	var cdWorkflowStatus []*pipelineConfig.CdWorkflowStatus

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		if apiVersion == "v2" {
			ciWorkflowStatus, err = handler.ciHandler.FetchCiStatusForTriggerViewV1(appId)
		} else {
			ciWorkflowStatus, err = handler.ciHandler.FetchCiStatusForTriggerView(appId)
		}
		wg.Done()
	}()

	go func() {
		cdWorkflowStatus, err1 = handler.cdHandler.FetchAppWorkflowStatusForTriggerView(appId)
		wg.Done()
	}()
	wg.Wait()

	if err != nil {
		handler.Logger.Errorw("service err, FetchAppWorkflowStatusForTriggerView", "err", err, "appId", appId)
		if util.IsErrNoRows(err) {
			err = &util.ApiError{Code: "404", HttpStatusCode: 200, UserMessage: "no workflow found"}
			common.WriteJsonResp(w, err, nil, http.StatusOK)
		} else {
			common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		}
		return
	}

	if err1 != nil {
		handler.Logger.Errorw("service err, FetchAppWorkflowStatusForTriggerView", "err", err1, "appId", appId)
		if util.IsErrNoRows(err1) {
			err1 = &util.ApiError{Code: "404", HttpStatusCode: 200, UserMessage: "no status found"}
			common.WriteJsonResp(w, err1, nil, http.StatusOK)
		} else {
			common.WriteJsonResp(w, err1, nil, http.StatusInternalServerError)
		}
		return
	}

	triggerWorkflowStatus.CiWorkflowStatus = ciWorkflowStatus
	triggerWorkflowStatus.CdWorkflowStatus = cdWorkflowStatus
	common.WriteJsonResp(w, err, triggerWorkflowStatus, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) PipelineNameSuggestion(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("token")
	vars := mux.Vars(r)
	appId, err := strconv.Atoi(vars["appId"])
	if err != nil {
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	pType := vars["type"]
	handler.Logger.Infow("request payload, PipelineNameSuggestion", "err", err, "appId", appId)
	app, err := handler.pipelineBuilder.GetApp(appId)
	if err != nil {
		handler.Logger.Infow("service error, GetCIPipelineById", "err", err, "appId", appId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	suggestedName := beHelper.GetPipelineNameByPipelineType(pType, appId)
	resourceName := handler.enforcerUtil.GetAppRBACName(app.AppName)
	ok := handler.enforcerUtil.CheckAppRbacForAppOrJob(token, resourceName, casbin.ActionGet)
	if !ok {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusForbidden)
		return
	}
	common.WriteJsonResp(w, err, suggestedName, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) FetchAppWorkflowStatusForTriggerViewByEnvironment(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("token")
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	envId, err := strconv.Atoi(vars["envId"])
	if err != nil {
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	v := r.URL.Query()
	appIdsString := v.Get("appIds")
	var appIds []int
	if len(appIdsString) > 0 {
		appIdsSlices := strings.Split(appIdsString, ",")
		for _, appId := range appIdsSlices {
			id, err := strconv.Atoi(appId)
			if err != nil {
				common.WriteJsonResp(w, err, "please provide valid appIds", http.StatusBadRequest)
				return
			}
			appIds = append(appIds, id)
		}
	}
	var appGroupId int
	appGroupIdStr := v.Get("appGroupId")
	if len(appGroupIdStr) > 0 {
		appGroupId, err = strconv.Atoi(appGroupIdStr)
		if err != nil {
			common.WriteJsonResp(w, err, "please provide valid appGroupId", http.StatusBadRequest)
			return
		}
	}
	request := resourceGroup2.ResourceGroupingRequest{
		ParentResourceId:  envId,
		ResourceGroupId:   appGroupId,
		ResourceGroupType: resourceGroup2.APP_GROUP,
		ResourceIds:       appIds,
		CheckAuthBatch:    handler.checkAuthBatch,
		UserId:            userId,
		Ctx:               r.Context(),
	}
	triggerWorkflowStatus := pipelineConfig.TriggerWorkflowStatus{}
	_, span := otel.Tracer("orchestrator").Start(r.Context(), "ciHandler.FetchCiStatusForBuildAndDeployInResourceGrouping")
	ciWorkflowStatus, err := handler.ciHandler.FetchCiStatusForTriggerViewForEnvironment(request, token)
	span.End()
	if err != nil {
		handler.Logger.Errorw("service err", "err", err)
		if util.IsErrNoRows(err) {
			err = &util.ApiError{Code: "404", HttpStatusCode: 200, UserMessage: "no workflow found"}
			common.WriteJsonResp(w, err, nil, http.StatusOK)
		} else {
			common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		}
		return
	}

	_, span = otel.Tracer("orchestrator").Start(r.Context(), "ciHandler.FetchCdStatusForBuildAndDeployInResourceGrouping")
	cdWorkflowStatus, err := handler.cdHandler.FetchAppWorkflowStatusForTriggerViewForEnvironment(request, token)
	span.End()
	if err != nil {
		handler.Logger.Errorw("service err, FetchAppWorkflowStatusForTriggerView", "err", err)
		if util.IsErrNoRows(err) {
			err = &util.ApiError{Code: "404", HttpStatusCode: 200, UserMessage: "no status found"}
			common.WriteJsonResp(w, err, nil, http.StatusOK)
		} else {
			common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		}
		return
	}
	triggerWorkflowStatus.CiWorkflowStatus = ciWorkflowStatus
	triggerWorkflowStatus.CdWorkflowStatus = cdWorkflowStatus
	common.WriteJsonResp(w, err, triggerWorkflowStatus, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) GetEnvironmentListWithAppData(w http.ResponseWriter, r *http.Request) {
	v := r.URL.Query()
	token := r.Header.Get("token")
	envName := v.Get("envName")
	clusterIdString := v.Get("clusterIds")
	offset := 0
	offsetStr := v.Get("offset")
	if len(offsetStr) > 0 {
		offset, _ = strconv.Atoi(offsetStr)
	}
	size := 0
	sizeStr := v.Get("size")
	if len(sizeStr) > 0 {
		size, _ = strconv.Atoi(sizeStr)
	}
	var clusterIds []int
	if clusterIdString != "" {
		clusterIdSlices := strings.Split(clusterIdString, ",")
		for _, clusterId := range clusterIdSlices {
			id, err := strconv.Atoi(clusterId)
			if err != nil {
				common.WriteJsonResp(w, err, "please send valid cluster Ids", http.StatusBadRequest)
				return
			}
			clusterIds = append(clusterIds, id)
		}
	}
	_, span := otel.Tracer("orchestrator").Start(r.Context(), "pipelineBuilder.GetEnvironmentListWithAppData")
	result, err := handler.pipelineBuilder.GetEnvironmentListForAutocompleteFilter(envName, clusterIds, offset, size, token, handler.checkAuthBatch, r.Context())
	span.End()
	if err != nil {
		handler.Logger.Errorw("service err, get app", "err", err)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, result, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) GetApplicationsByEnvironment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := r.Header.Get("token")
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	envId, err := strconv.Atoi(vars["envId"])
	if err != nil {
		handler.Logger.Errorw("request err, get app", "err", err, "envId", envId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	v := r.URL.Query()
	appIdsString := v.Get("appIds")
	var appIds []int
	if len(appIdsString) > 0 {
		appIdsSlices := strings.Split(appIdsString, ",")
		for _, appId := range appIdsSlices {
			id, err := strconv.Atoi(appId)
			if err != nil {
				common.WriteJsonResp(w, err, "please provide valid appIds", http.StatusBadRequest)
				return
			}
			appIds = append(appIds, id)
		}
	}
	var appGroupId int
	appGroupIdStr := v.Get("appGroupId")
	if len(appGroupIdStr) > 0 {
		appGroupId, err = strconv.Atoi(appGroupIdStr)
		if err != nil {
			common.WriteJsonResp(w, err, "please provide valid appGroupId", http.StatusBadRequest)
			return
		}
	}
	request := resourceGroup2.ResourceGroupingRequest{
		ParentResourceId:  envId,
		ResourceGroupId:   appGroupId,
		ResourceGroupType: resourceGroup2.APP_GROUP,
		ResourceIds:       appIds,
		CheckAuthBatch:    handler.checkAuthBatch,
		UserId:            userId,
		Ctx:               r.Context(),
	}
	results, err := handler.pipelineBuilder.GetAppListForEnvironment(request, token)
	if err != nil {
		handler.Logger.Errorw("service err, get app", "err", err)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, results, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) FetchAppDeploymentStatusForEnvironments(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("token")
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	envId, err := strconv.Atoi(vars["envId"])
	if err != nil {
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	v := r.URL.Query()
	appIdsString := v.Get("appIds")
	var appIds []int
	if len(appIdsString) > 0 {
		appIdsSlices := strings.Split(appIdsString, ",")
		for _, appId := range appIdsSlices {
			id, err := strconv.Atoi(appId)
			if err != nil {
				common.WriteJsonResp(w, err, "please provide valid appIds", http.StatusBadRequest)
				return
			}
			appIds = append(appIds, id)
		}
	}
	var appGroupId int
	appGroupIdStr := v.Get("appGroupId")
	if len(appGroupIdStr) > 0 {
		appGroupId, err = strconv.Atoi(appGroupIdStr)
		if err != nil {
			common.WriteJsonResp(w, err, "please provide valid appGroupId", http.StatusBadRequest)
			return
		}
	}

	request := resourceGroup2.ResourceGroupingRequest{
		ParentResourceId:  envId,
		ResourceGroupId:   appGroupId,
		ResourceGroupType: resourceGroup2.APP_GROUP,
		ResourceIds:       appIds,
		CheckAuthBatch:    handler.checkAuthBatch,
		UserId:            userId,
		Ctx:               r.Context(),
	}
	_, span := otel.Tracer("orchestrator").Start(r.Context(), "pipelineBuilder.FetchAppDeploymentStatusForEnvironments")
	results, err := handler.cdHandler.FetchAppDeploymentStatusForEnvironments(request, token)
	span.End()
	if err != nil {
		handler.Logger.Errorw("service err, FetchAppWorkflowStatusForTriggerView", "err", err)
		if util.IsErrNoRows(err) {
			err = &util.ApiError{Code: "404", HttpStatusCode: 200, UserMessage: "no status found"}
			common.WriteJsonResp(w, err, nil, http.StatusOK)
		} else {
			common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		}
		return
	}
	common.WriteJsonResp(w, err, results, http.StatusOK)
}
