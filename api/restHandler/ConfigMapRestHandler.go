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

package restHandler

import (
	"encoding/json"
	"fmt"
	"github.com/devtron-labs/devtron/pkg/auth/user/util"
	"github.com/devtron-labs/devtron/pkg/pipeline/draftAwareConfigService"
	"net/http"
	"strconv"

	"github.com/devtron-labs/devtron/api/restHandler/common"
	"github.com/devtron-labs/devtron/internal/sql/repository/pipelineConfig"
	"github.com/devtron-labs/devtron/pkg/auth/authorisation/casbin"
	"github.com/devtron-labs/devtron/pkg/auth/user"
	"github.com/devtron-labs/devtron/pkg/chart"
	"github.com/devtron-labs/devtron/pkg/pipeline"
	"github.com/devtron-labs/devtron/pkg/pipeline/bean"
	"github.com/devtron-labs/devtron/pkg/team"
	"github.com/devtron-labs/devtron/util/rbac"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type ConfigMapRestHandler interface {
	CMGlobalAddUpdate(w http.ResponseWriter, r *http.Request)
	CMEnvironmentAddUpdate(w http.ResponseWriter, r *http.Request)
	CMGlobalFetch(w http.ResponseWriter, r *http.Request)
	CMEnvironmentFetch(w http.ResponseWriter, r *http.Request)
	CMGlobalFetchForEdit(w http.ResponseWriter, r *http.Request)
	CMEnvironmentFetchForEdit(w http.ResponseWriter, r *http.Request)

	CSGlobalAddUpdate(w http.ResponseWriter, r *http.Request)
	CSEnvironmentAddUpdate(w http.ResponseWriter, r *http.Request)
	CSGlobalFetch(w http.ResponseWriter, r *http.Request)
	CSEnvironmentFetch(w http.ResponseWriter, r *http.Request)

	CMGlobalDelete(w http.ResponseWriter, r *http.Request)
	CMEnvironmentDelete(w http.ResponseWriter, r *http.Request)
	CSGlobalDelete(w http.ResponseWriter, r *http.Request)
	CSEnvironmentDelete(w http.ResponseWriter, r *http.Request)

	CSGlobalFetchForEdit(w http.ResponseWriter, r *http.Request)
	CSEnvironmentFetchForEdit(w http.ResponseWriter, r *http.Request)
	ConfigSecretBulkPatch(w http.ResponseWriter, r *http.Request)

	AddEnvironmentToJob(w http.ResponseWriter, r *http.Request)
	RemoveEnvironmentFromJob(w http.ResponseWriter, r *http.Request)
	GetEnvironmentsForJob(w http.ResponseWriter, r *http.Request)
}

type ConfigMapRestHandlerImpl struct {
	pipelineBuilder           pipeline.PipelineBuilder
	Logger                    *zap.SugaredLogger
	chartService              chart.ChartService
	userAuthService           user.UserService
	teamService               team.TeamService
	enforcer                  casbin.Enforcer
	pipelineRepository        pipelineConfig.PipelineRepository
	enforcerUtil              rbac.EnforcerUtil
	configMapService          pipeline.ConfigMapService
	draftAwareResourceService draftAwareConfigService.DraftAwareConfigService
}

func NewConfigMapRestHandlerImpl(pipelineBuilder pipeline.PipelineBuilder, Logger *zap.SugaredLogger,
	chartService chart.ChartService, userAuthService user.UserService, teamService team.TeamService,
	enforcer casbin.Enforcer, pipelineRepository pipelineConfig.PipelineRepository,
	enforcerUtil rbac.EnforcerUtil, configMapService pipeline.ConfigMapService,
	draftAwareResourceService draftAwareConfigService.DraftAwareConfigService,
) *ConfigMapRestHandlerImpl {
	return &ConfigMapRestHandlerImpl{
		pipelineBuilder:           pipelineBuilder,
		Logger:                    Logger,
		chartService:              chartService,
		userAuthService:           userAuthService,
		teamService:               teamService,
		enforcer:                  enforcer,
		pipelineRepository:        pipelineRepository,
		enforcerUtil:              enforcerUtil,
		configMapService:          configMapService,
		draftAwareResourceService: draftAwareResourceService,
	}
}

func (handler ConfigMapRestHandlerImpl) CMGlobalAddUpdate(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	var configMapRequest bean.ConfigDataRequest

	err = decoder.Decode(&configMapRequest)
	if err != nil {
		handler.Logger.Errorw("request err, CMGlobalAddUpdate", "err", err, "payload", configMapRequest)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	configMapRequest.UserId = userId
	handler.Logger.Errorw("request payload, CMGlobalAddUpdate", "payload", configMapRequest)

	//RBAC START
	token := r.Header.Get("token")
	object := handler.enforcerUtil.GetAppRBACNameByAppId(configMapRequest.AppId)
	ok := handler.enforcerUtil.CheckAppRbacForAppOrJob(token, object, casbin.ActionCreate)
	if !ok {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusForbidden)
		return
	}
	//RBAC END
	ctx := r.Context()
	isSuperAdmin := handler.enforcer.Enforce(token, casbin.ResourceGlobal, casbin.ActionCreate, "*")
	userMetadata := util.GetUserMetadata(r.Context(), userId, isSuperAdmin)
	res, err := handler.draftAwareResourceService.CMGlobalAddUpdate(ctx, &configMapRequest, userMetadata)
	if err != nil {
		handler.Logger.Errorw("service err, CMGlobalAddUpdate", "err", err, "payload", configMapRequest)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, res, http.StatusOK)
}

func (handler ConfigMapRestHandlerImpl) CMEnvironmentAddUpdate(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	var configMapRequest bean.ConfigDataRequest
	err = decoder.Decode(&configMapRequest)
	if err != nil {
		handler.Logger.Errorw("request err, CMEnvironmentAddUpdate", "err", err, "payload", configMapRequest)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	configMapRequest.UserId = userId
	handler.Logger.Errorw("request payload, CMEnvironmentAddUpdate", "payload", configMapRequest)

	//RBAC START
	token := r.Header.Get("token")
	object := handler.enforcerUtil.GetAppRBACNameByAppId(configMapRequest.AppId)
	ok := handler.enforcerUtil.CheckAppRbacForAppOrJob(token, object, casbin.ActionCreate)
	if !ok {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusForbidden)
		return
	}
	object = handler.enforcerUtil.GetEnvRBACNameByAppId(configMapRequest.AppId, configMapRequest.EnvironmentId)
	object2 := handler.enforcerUtil.GetTeamEnvRBACNameByAppId(configMapRequest.AppId, configMapRequest.EnvironmentId)
	if ok := handler.enforcer.Enforce(token, casbin.ResourceEnvironment, casbin.ActionCreate, object); !ok {
		if ok2 := handler.enforcer.Enforce(token, casbin.ResourceJobsEnv, casbin.ActionCreate, object2); !ok2 {
			common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusForbidden)
			return
		}
	}
	//RBAC END
	ctx := r.Context()
	isSuperAdmin := handler.enforcer.Enforce(token, casbin.ResourceGlobal, casbin.ActionCreate, "*")
	userMetadata := util.GetUserMetadata(r.Context(), userId, isSuperAdmin)
	res, err := handler.draftAwareResourceService.CMEnvironmentAddUpdate(ctx, &configMapRequest, userMetadata)
	if err != nil {
		handler.Logger.Errorw("service err, CMEnvironmentAddUpdate", "err", err, "payload", configMapRequest)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, res, http.StatusOK)
}

func (handler ConfigMapRestHandlerImpl) CMGlobalFetch(w http.ResponseWriter, r *http.Request) {
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	appId, err := strconv.Atoi(vars["appId"])
	if err != nil {
		handler.Logger.Errorw("request err, CMGlobalFetch", "err", err, "appId", appId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}

	//RBAC START
	token := r.Header.Get("token")
	object := handler.enforcerUtil.GetAppRBACNameByAppId(appId)
	ok := handler.enforcerUtil.CheckAppRbacForAppOrJob(token, object, casbin.ActionGet)
	if !ok {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusForbidden)
		return
	}
	//RBAC END

	res, err := handler.configMapService.CMGlobalFetch(appId)
	if err != nil {
		handler.Logger.Errorw("service err, CMGlobalFetch", "err", err, "appId", appId)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, res, http.StatusOK)
}

func (handler ConfigMapRestHandlerImpl) CMGlobalFetchForEdit(w http.ResponseWriter, r *http.Request) {
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	appId, err := strconv.Atoi(vars["appId"])
	if err != nil {
		handler.Logger.Errorw("request err, CMGlobalFetch", "err", err, "appId", appId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	cmId, err := strconv.Atoi(vars["id"])
	if err != nil {
		handler.Logger.Errorw("request err, CMGlobalFetch", "err", err, "cmId", cmId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	token := r.Header.Get("token")
	object := handler.enforcerUtil.GetAppRBACNameByAppId(appId)
	ok := handler.enforcerUtil.CheckAppRbacForAppOrJob(token, object, casbin.ActionGet)
	if !ok {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusForbidden)
		return
	}

	name := vars["name"]
	response, err := handler.configMapService.CMGlobalFetchForEdit(name, cmId)
	if err != nil {
		handler.Logger.Errorw("service err, CMGlobalFetchForEdit", "err", err, "appId", appId)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, response, http.StatusOK)
}

func (handler ConfigMapRestHandlerImpl) CMEnvironmentFetchForEdit(w http.ResponseWriter, r *http.Request) {
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	appId, err := strconv.Atoi(vars["appId"])
	if err != nil {
		handler.Logger.Errorw("request err, CMGlobalFetch", "err", err, "appId", appId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	envId, err := strconv.Atoi(vars["envId"])
	if err != nil {
		handler.Logger.Errorw("request err, CMGlobalFetch", "err", err, "envId", envId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	cmId, err := strconv.Atoi(vars["id"])
	if err != nil {
		handler.Logger.Errorw("request err, CMGlobalFetch", "err", err, "cmId", cmId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	token := r.Header.Get("token")
	object := handler.enforcerUtil.GetAppRBACNameByAppId(appId)
	ok := handler.enforcerUtil.CheckAppRbacForAppOrJob(token, object, casbin.ActionGet)
	if !ok {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusForbidden)
		return
	}
	object = handler.enforcerUtil.GetEnvRBACNameByAppId(appId, envId)
	object2 := handler.enforcerUtil.GetTeamEnvRBACNameByAppId(appId, envId)
	if ok := handler.enforcer.Enforce(token, casbin.ResourceEnvironment, casbin.ActionGet, object); !ok {
		if ok2 := handler.enforcer.Enforce(token, casbin.ResourceJobsEnv, casbin.ActionGet, object2); !ok2 {
			common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusForbidden)
			return
		}
	}

	name := vars["name"]
	response, err := handler.configMapService.CMEnvironmentFetchForEdit(name, cmId, appId, envId)
	if err != nil {
		handler.Logger.Errorw("service err, CMEnvironmentFetchForEdit", "err", err, "appId", appId, "envId", envId)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, response, http.StatusOK)
}

func (handler ConfigMapRestHandlerImpl) CMEnvironmentFetch(w http.ResponseWriter, r *http.Request) {
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	appId, err := strconv.Atoi(vars["appId"])
	if err != nil {
		handler.Logger.Errorw("request err, CMEnvironmentFetch", "err", err, "appId", appId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	envId, err := strconv.Atoi(vars["envId"])
	if err != nil {
		handler.Logger.Errorw("request err, CMEnvironmentFetch", "err", err, "appId", appId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}

	//RBAC START
	token := r.Header.Get("token")
	object := handler.enforcerUtil.GetAppRBACNameByAppId(appId)
	ok := handler.enforcerUtil.CheckAppRbacForAppOrJob(token, object, casbin.ActionGet)
	if !ok {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusForbidden)
		return
	}
	//RBAC END

	res, err := handler.configMapService.CMEnvironmentFetch(appId, envId)
	if err != nil {
		handler.Logger.Errorw("service err, CMEnvironmentFetch", "err", err, "appId", appId)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, res, http.StatusOK)
}

func (handler ConfigMapRestHandlerImpl) CSGlobalAddUpdate(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	var configMapRequest bean.ConfigDataRequest

	err = decoder.Decode(&configMapRequest)
	if err != nil {
		handler.Logger.Errorw("request err, CSGlobalAddUpdate", "err", err, "payload", configMapRequest)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	configMapRequest.UserId = userId
	handler.Logger.Errorw("request payload, CSGlobalAddUpdate", "payload", configMapRequest)

	//RBAC START
	token := r.Header.Get("token")
	object := handler.enforcerUtil.GetAppRBACNameByAppId(configMapRequest.AppId)
	ok := handler.enforcerUtil.CheckAppRbacForAppOrJob(token, object, casbin.ActionCreate)
	if !ok {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusForbidden)
		return
	}
	//RBAC END
	ctx := r.Context()
	isSuperAdmin := handler.enforcer.Enforce(token, casbin.ResourceGlobal, casbin.ActionCreate, "*")
	userMetadata := util.GetUserMetadata(r.Context(), userId, isSuperAdmin)
	res, err := handler.draftAwareResourceService.CSGlobalAddUpdate(ctx, &configMapRequest, userMetadata)
	if err != nil {
		handler.Logger.Errorw("service err, CSGlobalAddUpdate", "err", err, "payload", configMapRequest)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, res, http.StatusOK)
}

func (handler ConfigMapRestHandlerImpl) CSEnvironmentAddUpdate(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	var configMapRequest bean.ConfigDataRequest

	err = decoder.Decode(&configMapRequest)
	if err != nil {
		handler.Logger.Errorw("request err, CSEnvironmentAddUpdate", "err", err, "payload", configMapRequest)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	configMapRequest.UserId = userId
	handler.Logger.Errorw("request payload, CSEnvironmentAddUpdate", "payload", configMapRequest)

	//RBAC START
	token := r.Header.Get("token")
	object := handler.enforcerUtil.GetAppRBACNameByAppId(configMapRequest.AppId)
	ok := handler.enforcerUtil.CheckAppRbacForAppOrJob(token, object, casbin.ActionCreate)
	if !ok {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusForbidden)
		return
	}
	object = handler.enforcerUtil.GetEnvRBACNameByAppId(configMapRequest.AppId, configMapRequest.EnvironmentId)
	object2 := handler.enforcerUtil.GetTeamEnvAppRbacObjectByAppIdEnvIdOrName(configMapRequest.AppId, configMapRequest.EnvironmentId, "")
	if ok := handler.enforcer.Enforce(token, casbin.ResourceEnvironment, casbin.ActionCreate, object); !ok {
		if ok2 := handler.enforcer.Enforce(token, casbin.ResourceJobsEnv, casbin.ActionCreate, object2); !ok2 {
			common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusForbidden)
			return
		}
	}
	//RBAC END
	ctx := r.Context()
	isSuperAdmin := handler.enforcer.Enforce(token, casbin.ResourceGlobal, casbin.ActionCreate, "*")
	userMetadata := util.GetUserMetadata(r.Context(), userId, isSuperAdmin)
	res, err := handler.draftAwareResourceService.CSEnvironmentAddUpdate(ctx, &configMapRequest, userMetadata)
	if err != nil {
		handler.Logger.Errorw("service err, CSEnvironmentAddUpdate", "err", err, "payload", configMapRequest)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, res, http.StatusOK)
}

func (handler ConfigMapRestHandlerImpl) CSGlobalFetch(w http.ResponseWriter, r *http.Request) {
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	appId, err := strconv.Atoi(vars["appId"])
	if err != nil {
		handler.Logger.Errorw("request err, CSGlobalFetch", "err", err, "appId", appId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}

	//RBAC START
	token := r.Header.Get("token")
	object := handler.enforcerUtil.GetAppRBACNameByAppId(appId)
	ok := handler.enforcerUtil.CheckAppRbacForAppOrJob(token, object, casbin.ActionGet)
	if !ok {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), nil, http.StatusForbidden)
		return
	}
	//RBAC END

	res, err := handler.configMapService.CSGlobalFetch(appId)
	if err != nil {
		handler.Logger.Errorw("service err, CSGlobalFetch", "err", err, "appId", appId)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, res, http.StatusOK)
}

func (handler ConfigMapRestHandlerImpl) CSEnvironmentFetch(w http.ResponseWriter, r *http.Request) {
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	appId, err := strconv.Atoi(vars["appId"])
	if err != nil {
		handler.Logger.Errorw("request err, CSEnvironmentFetch", "err", err, "appId", appId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	envId, err := strconv.Atoi(vars["envId"])
	if err != nil {
		handler.Logger.Errorw("bad request", "err", err)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}

	//RBAC START
	token := r.Header.Get("token")
	object := handler.enforcerUtil.GetAppRBACNameByAppId(appId)
	ok := handler.enforcerUtil.CheckAppRbacForAppOrJob(token, object, casbin.ActionGet)
	if !ok {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), nil, http.StatusForbidden)
		return
	}
	//RBAC END

	res, err := handler.configMapService.CSEnvironmentFetch(appId, envId)
	if err != nil {
		handler.Logger.Errorw("service err, CSEnvironmentFetch", "err", err, "appId", appId)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, res, http.StatusOK)
}

func (handler ConfigMapRestHandlerImpl) CMGlobalDelete(w http.ResponseWriter, r *http.Request) {
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	appId, err := strconv.Atoi(vars["appId"])
	if err != nil {
		handler.Logger.Errorw("request err, CMGlobalDelete", "err", err, "appId", appId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		handler.Logger.Errorw("request err, CMGlobalDelete", "err", err, "id", id)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	name := vars["name"]
	handler.Logger.Errorw("request payload, CMGlobalDelete", "appId", appId, "id", id, "name", name)

	//RBAC START
	token := r.Header.Get("token")
	object := handler.enforcerUtil.GetAppRBACNameByAppId(appId)
	ok := handler.enforcerUtil.CheckAppRbacForAppOrJob(token, object, casbin.ActionDelete)
	if !ok {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), nil, http.StatusForbidden)
		return
	}
	//RBAC END
	ctx := r.Context()
	isSuperAdmin := handler.enforcer.Enforce(token, casbin.ResourceGlobal, casbin.ActionCreate, "*")
	userMetadata := util.GetUserMetadata(r.Context(), userId, isSuperAdmin)
	deleteReq := &bean.ConfigDataRequest{
		Id:     id,
		AppId:  appId,
		UserId: userId,
	}
	res, err := handler.draftAwareResourceService.CMGlobalDelete(ctx, name, deleteReq, userMetadata)
	if err != nil {
		handler.Logger.Errorw("service err, CMGlobalDelete", "err", err, "appId", appId, "id", id, "name", name)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, res, http.StatusOK)
}

func (handler ConfigMapRestHandlerImpl) CMEnvironmentDelete(w http.ResponseWriter, r *http.Request) {
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	appId, err := strconv.Atoi(vars["appId"])
	if err != nil {
		handler.Logger.Errorw("request err, CMEnvironmentDelete", "err", err, "appId", appId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	envId, err := strconv.Atoi(vars["envId"])
	if err != nil {
		handler.Logger.Errorw("request err, CMEnvironmentDelete", "err", err, "envId", envId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		handler.Logger.Errorw("request err, CMEnvironmentDelete", "err", err, "id", id)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	name := vars["name"]
	handler.Logger.Errorw("request payload, CMEnvironmentDelete", "appId", appId, "envId", envId, "id", id, "name", name)

	//RBAC START
	token := r.Header.Get("token")
	object := handler.enforcerUtil.GetAppRBACNameByAppId(appId)
	ok := handler.enforcerUtil.CheckAppRbacForAppOrJob(token, object, casbin.ActionDelete)
	if !ok {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), nil, http.StatusForbidden)
		return
	}
	object = handler.enforcerUtil.GetEnvRBACNameByAppId(appId, envId)
	object2 := handler.enforcerUtil.GetTeamEnvAppRbacObjectByAppIdEnvIdOrName(appId, envId, "")
	if ok := handler.enforcer.Enforce(token, casbin.ResourceEnvironment, casbin.ActionDelete, object); !ok {
		if ok2 := handler.enforcer.Enforce(token, casbin.ResourceJobsEnv, casbin.ActionDelete, object2); !ok2 {
			common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), nil, http.StatusForbidden)
			return
		}
	}
	//RBAC END
	ctx := r.Context()
	isSuperAdmin := handler.enforcer.Enforce(token, casbin.ResourceGlobal, casbin.ActionCreate, "*")
	userMetadata := util.GetUserMetadata(r.Context(), userId, isSuperAdmin)
	deleteReq := &bean.ConfigDataRequest{
		Id:     id,
		AppId:  appId,
		UserId: userId,
	}
	res, err := handler.draftAwareResourceService.CMEnvironmentDelete(ctx, name, deleteReq, userMetadata)
	if err != nil {
		handler.Logger.Errorw("service err, CMEnvironmentDelete", "err", err, "appId", appId, "envId", envId, "id", id)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, res, http.StatusOK)
}

func (handler ConfigMapRestHandlerImpl) CSGlobalDelete(w http.ResponseWriter, r *http.Request) {
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	appId, err := strconv.Atoi(vars["appId"])
	if err != nil {
		handler.Logger.Errorw("request err, CSGlobalDelete", "err", err, "appId", appId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		handler.Logger.Errorw("request err, CSGlobalDelete", "err", err, "id", id)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	name := vars["name"]
	handler.Logger.Errorw("request payload, CSGlobalDelete", "appId", appId, "id", id, "name", name)

	//RBAC START
	token := r.Header.Get("token")
	object := handler.enforcerUtil.GetAppRBACNameByAppId(appId)
	ok := handler.enforcerUtil.CheckAppRbacForAppOrJob(token, object, casbin.ActionDelete)
	if !ok {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), nil, http.StatusForbidden)
		return
	}
	//RBAC END
	ctx := r.Context()
	isSuperAdmin := handler.enforcer.Enforce(token, casbin.ResourceGlobal, casbin.ActionCreate, "*")
	userMetadata := util.GetUserMetadata(r.Context(), userId, isSuperAdmin)
	deleteReq := &bean.ConfigDataRequest{
		Id:     id,
		AppId:  appId,
		UserId: userId,
	}
	res, err := handler.draftAwareResourceService.CSGlobalDelete(ctx, name, deleteReq, userMetadata)
	if err != nil {
		handler.Logger.Errorw("service err, CSGlobalDelete", "err", err, "appId", appId, "id", id, "name", name)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, res, http.StatusOK)
}

func (handler ConfigMapRestHandlerImpl) CSEnvironmentDelete(w http.ResponseWriter, r *http.Request) {
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	appId, err := strconv.Atoi(vars["appId"])
	if err != nil {
		handler.Logger.Errorw("request err, CSEnvironmentDelete", "err", err, "appId", appId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	envId, err := strconv.Atoi(vars["envId"])
	if err != nil {
		handler.Logger.Errorw("request err, CSEnvironmentDelete", "err", err, "envId", envId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		handler.Logger.Errorw("request err, CSEnvironmentDelete", "err", err, "id", id)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	name := vars["name"]
	handler.Logger.Errorw("request payload, CSEnvironmentDelete", "appId", appId, "envId", envId, "id", id, "name", name)

	//RBAC START
	token := r.Header.Get("token")
	object := handler.enforcerUtil.GetAppRBACNameByAppId(appId)
	ok := handler.enforcerUtil.CheckAppRbacForAppOrJob(token, object, casbin.ActionDelete)
	if !ok {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), nil, http.StatusForbidden)
		return
	}
	object = handler.enforcerUtil.GetEnvRBACNameByAppId(appId, envId)
	object2 := handler.enforcerUtil.GetTeamEnvAppRbacObjectByAppIdEnvIdOrName(appId, envId, "")
	if ok := handler.enforcer.Enforce(token, casbin.ResourceEnvironment, casbin.ActionDelete, object); !ok {
		if ok2 := handler.enforcer.Enforce(token, casbin.ResourceJobsEnv, casbin.ActionDelete, object2); !ok2 {
			common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), nil, http.StatusForbidden)
			return
		}
	}
	//RBAC END
	ctx := r.Context()
	isSuperAdmin := handler.enforcer.Enforce(token, casbin.ResourceGlobal, casbin.ActionCreate, "*")
	userMetadata := util.GetUserMetadata(r.Context(), userId, isSuperAdmin)
	deleteReq := &bean.ConfigDataRequest{
		Id:     id,
		AppId:  appId,
		UserId: userId,
	}
	res, err := handler.draftAwareResourceService.CSEnvironmentDelete(ctx, name, deleteReq, userMetadata)
	if err != nil {
		handler.Logger.Errorw("service err, CSEnvironmentDelete", "err", err, "appId", appId, "envId", envId, "id", id)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, res, http.StatusOK)
}

func (handler ConfigMapRestHandlerImpl) CSGlobalFetchForEdit(w http.ResponseWriter, r *http.Request) {
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	appId, err := strconv.Atoi(vars["appId"])
	if err != nil {
		handler.Logger.Errorw("request err, CSGlobalFetchForEdit", "err", err, "appId", appId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		handler.Logger.Errorw("request err, CSGlobalFetchForEdit", "err", err, "id", id)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	name := vars["name"]
	handler.Logger.Errorw("request payload, CSGlobalFetchForEdit", "appId", appId, "id", id, "name", name)

	//RBAC START
	token := r.Header.Get("token")
	object := handler.enforcerUtil.GetAppRBACNameByAppId(appId)
	ok := handler.enforcerUtil.CheckAppRbacForAppOrJob(token, object, casbin.ActionUpdate)
	if !ok {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), nil, http.StatusForbidden)
		return
	}
	//RBAC END

	res, err := handler.configMapService.CSGlobalFetchForEdit(name, id)
	if err != nil {
		handler.Logger.Errorw("service err, CSGlobalFetchForEdit", "err", err, "appId", appId, "id", id, "name", name)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, res, http.StatusOK)
}

func (handler ConfigMapRestHandlerImpl) CSEnvironmentFetchForEdit(w http.ResponseWriter, r *http.Request) {
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	appId, err := strconv.Atoi(vars["appId"])
	if err != nil {
		handler.Logger.Errorw("request err, CSEnvironmentFetchForEdit", "err", err, "appId", appId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	envId, err := strconv.Atoi(vars["envId"])
	if err != nil {
		handler.Logger.Errorw("request err, CSEnvironmentFetchForEdit", "err", err, "envId", envId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		handler.Logger.Errorw("request err, CSEnvironmentFetchForEdit", "err", err, "id", id)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	name := vars["name"]
	handler.Logger.Errorw("request payload, CSEnvironmentFetchForEdit", "appId", appId, "envId", envId, "id", id, "name", name)

	//RBAC START
	token := r.Header.Get("token")
	object := handler.enforcerUtil.GetAppRBACNameByAppId(appId)
	ok := handler.enforcerUtil.CheckAppRbacForAppOrJob(token, object, casbin.ActionUpdate)
	if !ok {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), nil, http.StatusForbidden)
		return
	}
	object = handler.enforcerUtil.GetEnvRBACNameByAppId(appId, envId)
	object2 := handler.enforcerUtil.GetTeamEnvAppRbacObjectByAppIdEnvIdOrName(appId, envId, "")
	if ok := handler.enforcer.Enforce(token, casbin.ResourceEnvironment, casbin.ActionUpdate, object); !ok {
		if ok2 := handler.enforcer.Enforce(token, casbin.ResourceJobsEnv, casbin.ActionUpdate, object2); !ok2 {
			common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), nil, http.StatusForbidden)
			return
		}
	}
	//RBAC END

	res, err := handler.configMapService.CSEnvironmentFetchForEdit(name, id, appId, envId)
	if err != nil {
		handler.Logger.Errorw("service err, CSEnvironmentFetchForEdit", "err", err, "appId", appId, "envId", envId, "id", id)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, res, http.StatusOK)
}

func (handler ConfigMapRestHandlerImpl) ConfigSecretBulkPatch(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}

	//AUTH - check from casbin db
	token := r.Header.Get("token")
	if isSuperAdmin := handler.enforcer.Enforce(token, casbin.ResourceGlobal, casbin.ActionUpdate, "*"); !isSuperAdmin {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusForbidden)
		return
	}
	//AUTH

	var bulkPatchRequest bean.BulkPatchRequest
	err = decoder.Decode(&bulkPatchRequest)
	if err != nil {
		handler.Logger.Errorw("request err, ConfigSecretBulkPatch", "err", err, "payload", bulkPatchRequest)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.Logger.Infow("request payload, ConfigSecretBulkPatch", "payload", bulkPatchRequest)
	bulkPatchRequest.UserId = userId
	if bulkPatchRequest.Global {
		_, err := handler.configMapService.ConfigSecretGlobalBulkPatch(&bulkPatchRequest)
		if err != nil {
			handler.Logger.Errorw("service err, ConfigSecretBulkPatch", "err", err, "payload", bulkPatchRequest)
			common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
			return
		}
	} else {
		_, err := handler.configMapService.ConfigSecretEnvironmentBulkPatch(&bulkPatchRequest)
		if err != nil {
			handler.Logger.Errorw("service err, ConfigSecretBulkPatch", "err", err, "payload", bulkPatchRequest)
			common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
			return
		}
	}
	common.WriteJsonResp(w, err, true, http.StatusOK)
}

func (handler ConfigMapRestHandlerImpl) AddEnvironmentToJob(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	token := r.Header.Get("token")
	var envOverrideRequest bean.CreateJobEnvOverridePayload
	err = decoder.Decode(&envOverrideRequest)
	if err != nil {
		handler.Logger.Errorw("request err, AddEvironmentToJob", "err", err, "payload", envOverrideRequest)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}

	// RBAC ENFORCEMENT
	resourceObject := handler.enforcerUtil.GetTeamEnvRBACNameByAppId(envOverrideRequest.AppId, envOverrideRequest.EnvId)
	if ok := handler.enforcer.Enforce(token, casbin.ResourceJobsEnv, casbin.ActionCreate, resourceObject); !ok {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusForbidden)
		return
	}
	// RBAC ENFORCEMENT ENDS

	envOverrideRequest.UserId = userId
	handler.Logger.Infow("request payload, AddEvironmentToJob", "payload", envOverrideRequest)
	resp, err := handler.configMapService.ConfigSecretEnvironmentCreate(&envOverrideRequest)
	if err != nil {
		handler.Logger.Errorw("service err, AddEvironmentToJob", "err", err, "payload", envOverrideRequest)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}

	common.WriteJsonResp(w, err, resp, http.StatusOK)
}

func (handler ConfigMapRestHandlerImpl) RemoveEnvironmentFromJob(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}

	token := r.Header.Get("token")
	var envOverrideRequest bean.CreateJobEnvOverridePayload
	err = decoder.Decode(&envOverrideRequest)
	if err != nil {
		handler.Logger.Errorw("request err, RemoveEnvironmentFromJob", "err", err, "payload", envOverrideRequest)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}

	// RBAC ENFORCEMENT
	resourceObject := handler.enforcerUtil.GetTeamEnvRBACNameByAppId(envOverrideRequest.AppId, envOverrideRequest.EnvId)
	if ok := handler.enforcer.Enforce(token, casbin.ResourceJobsEnv, casbin.ActionDelete, resourceObject); !ok {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusForbidden)
		return
	}
	// RBAC ENFORCEMENT ENDS

	envOverrideRequest.UserId = userId
	handler.Logger.Infow("request payload, RemoveEnvironmentFromJob", "payload", envOverrideRequest)
	resp, err := handler.configMapService.ConfigSecretEnvironmentDelete(&envOverrideRequest)
	if err != nil {
		handler.Logger.Errorw("service err, RemoveEnvironmentFromJob", "err", err, "payload", envOverrideRequest)
		common.WriteJsonResp(w, err, resp, http.StatusInternalServerError)
		return
	}

	common.WriteJsonResp(w, err, true, http.StatusOK)
}
func (handler ConfigMapRestHandlerImpl) GetEnvironmentsForJob(w http.ResponseWriter, r *http.Request) {
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	appId, err := strconv.Atoi(vars["appId"])
	if err != nil {
		handler.Logger.Errorw("request err, GetEnvironmentsForJob", "err", err, "appId", appId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	//AUTH - check from casbin db
	token := r.Header.Get("token")
	object := handler.enforcerUtil.GetAppRBACNameByAppId(appId)
	if ok := handler.enforcer.Enforce(token, casbin.ResourceJobs, casbin.ActionGet, object); !ok {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusForbidden)
		return
	}
	//AUTH
	if err != nil {
		handler.Logger.Errorw("request err, GetEnvironmentsForJob", "err", err, "appId", appId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.Logger.Infow("request payload, GetEnvironmentsForJob", "appId", appId)
	resp, err := handler.configMapService.ConfigSecretEnvironmentGet(appId)
	if err != nil {
		handler.Logger.Errorw("service err, GetEnvironmentsForJob", "err", err, "appId", appId)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}

	common.WriteJsonResp(w, err, resp, http.StatusOK)
}
