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
	"errors"
	bean2 "github.com/devtron-labs/devtron/api/bean/gitOps"
	"github.com/devtron-labs/devtron/api/util"
	moduleBean "github.com/devtron-labs/devtron/pkg/module/bean"
	moduleRead "github.com/devtron-labs/devtron/pkg/module/read"
	moduleErr "github.com/devtron-labs/devtron/pkg/module/read/error"
	"net/http"
	"strconv"

	"github.com/devtron-labs/devtron/api/restHandler/common"
	"github.com/devtron-labs/devtron/pkg/auth/authorisation/casbin"
	"github.com/devtron-labs/devtron/pkg/auth/user"
	"github.com/devtron-labs/devtron/pkg/gitops"
	"github.com/devtron-labs/devtron/pkg/team"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"gopkg.in/go-playground/validator.v9"
)

type GitOpsConfigRestHandler interface {
	CreateGitOpsConfig(w http.ResponseWriter, r *http.Request)
	GetAllGitOpsConfig(w http.ResponseWriter, r *http.Request)
	GetGitOpsConfigById(w http.ResponseWriter, r *http.Request)
	UpdateGitOpsConfig(w http.ResponseWriter, r *http.Request)
	GetGitOpsConfigByProvider(w http.ResponseWriter, r *http.Request)
	GitOpsConfigured(w http.ResponseWriter, r *http.Request)
	GitOpsValidator(w http.ResponseWriter, r *http.Request)
}

type GitOpsConfigRestHandlerImpl struct {
	logger              *zap.SugaredLogger
	moduleReadService   moduleRead.ModuleReadService
	gitOpsConfigService gitops.GitOpsConfigService
	userAuthService     user.UserService
	validator           *validator.Validate
	enforcer            casbin.Enforcer
	teamService         team.TeamService
}

func NewGitOpsConfigRestHandlerImpl(
	logger *zap.SugaredLogger,
	moduleReadService moduleRead.ModuleReadService,
	gitOpsConfigService gitops.GitOpsConfigService, userAuthService user.UserService,
	validator *validator.Validate, enforcer casbin.Enforcer, teamService team.TeamService) *GitOpsConfigRestHandlerImpl {
	return &GitOpsConfigRestHandlerImpl{
		logger:              logger,
		moduleReadService:   moduleReadService,
		gitOpsConfigService: gitOpsConfigService,
		userAuthService:     userAuthService,
		validator:           validator,
		enforcer:            enforcer,
		teamService:         teamService,
	}
}

func (impl GitOpsConfigRestHandlerImpl) CreateGitOpsConfig(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	userId, err := impl.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	// RBAC enforcer applying
	token := r.Header.Get("token")
	if ok := impl.enforcer.Enforce(token, casbin.ResourceGlobal, casbin.ActionCreate, "*"); !ok {
		common.WriteJsonResp(w, errors.New("unauthorized"), nil, http.StatusForbidden)
		return
	}
	//RBAC enforcer Ends
	var bean bean2.GitOpsConfigDto
	err = decoder.Decode(&bean)
	if err != nil {
		impl.logger.Errorw("request err, createGitOpsConfig", "err", err, "payload", bean)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	bean.UserId = userId
	impl.logger.Infow("request payload, createGitOpsConfig", "err", err, "payload", bean)
	err = impl.validator.Struct(bean)
	if err != nil {
		impl.logger.Errorw("validation err, createGitOpsConfig", "err", err, "payload", bean)
		common.WriteJsonResp(w, util.CustomizeValidationError(err), nil, http.StatusBadRequest)
		return
	}
	detailedErrorGitOpsConfigResponse, err := impl.gitOpsConfigService.ValidateAndCreateGitOpsConfig(&bean)
	if err != nil {
		impl.logger.Errorw("service err, SaveGitRepoConfig", "err", err, "payload", bean)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, nil, detailedErrorGitOpsConfigResponse, http.StatusOK)
	return
}

func (impl GitOpsConfigRestHandlerImpl) UpdateGitOpsConfig(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	userId, err := impl.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	// RBAC enforcer applying
	token := r.Header.Get("token")
	if ok := impl.enforcer.Enforce(token, casbin.ResourceGlobal, casbin.ActionUpdate, "*"); !ok {
		common.WriteJsonResp(w, errors.New("unauthorized"), nil, http.StatusForbidden)
		return
	}
	//RBAC enforcer Ends
	var bean bean2.GitOpsConfigDto
	err = decoder.Decode(&bean)
	if bean.Token == "" {
		res, err := impl.gitOpsConfigService.GetGitOpsConfigByProvider(bean.Provider)
		if err != nil {
			impl.logger.Errorw("service err, GetGitOpsConfigByProvider", "err", err, "provider", bean.Provider, "response", res)
			common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
			return
		}
		bean.Token = res.Token
	}
	if err != nil {
		impl.logger.Errorw("request err, updateGitOpsConfig", "err", err, "payload", bean)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	bean.UserId = userId
	impl.logger.Infow("request payload, updateGitOpsConfig", "payload", bean)
	err = impl.validator.Struct(bean)
	if err != nil {
		impl.logger.Errorw("validation err, updateGitOpsConfig", "err", err, "payload", bean)
		common.WriteJsonResp(w, util.CustomizeValidationError(err), nil, http.StatusBadRequest)
		return
	}
	detailedErrorGitOpsConfigResponse, err := impl.gitOpsConfigService.ValidateAndUpdateGitOpsConfig(&bean)
	if err != nil {
		impl.logger.Errorw("service err, updateGitOpsConfig", "err", err, "payload", bean)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, nil, detailedErrorGitOpsConfigResponse, http.StatusOK)
	return
}

func (impl GitOpsConfigRestHandlerImpl) GetGitOpsConfigById(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userId, err := impl.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		impl.logger.Errorw("request err, GetGitOpsConfigById", "err", err, "chart repo id", id)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	res, err := impl.gitOpsConfigService.GetGitOpsConfigById(id)
	if err != nil {
		impl.logger.Errorw("service err, GetGitOpsConfigById", "err", err, "id", id)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}

	// RBAC enforcer applying
	token := r.Header.Get("token")
	if ok := impl.enforcer.Enforce(token, casbin.ResourceGlobal, casbin.ActionGet, "*"); !ok {
		common.WriteJsonResp(w, errors.New("unauthorized"), nil, http.StatusForbidden)
		return
	}
	// RBAC enforcer Ends

	common.WriteJsonResp(w, err, res, http.StatusOK)
}

func (impl GitOpsConfigRestHandlerImpl) GitOpsConfigured(w http.ResponseWriter, r *http.Request) {
	userId, err := impl.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	result, err := impl.gitOpsConfigService.GetAllGitOpsConfig()
	if err != nil {
		impl.logger.Errorw("service err, GetAllGitOpsConfig", "err", err)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	gitopsConfigured := false
	allowCustomRepository := false
	if len(result) > 0 {
		for _, gitopsConf := range result {
			if gitopsConf.Active {
				gitopsConfigured = true
				allowCustomRepository = gitopsConf.AllowCustomRepository
				break
			}
		}
	}
	res := make(map[string]bool)
	res["exists"] = gitopsConfigured
	res["allowCustomRepository"] = allowCustomRepository
	common.WriteJsonResp(w, err, res, http.StatusOK)
}

func (impl GitOpsConfigRestHandlerImpl) GetAllGitOpsConfig(w http.ResponseWriter, r *http.Request) {
	userId, err := impl.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	result, err := impl.gitOpsConfigService.GetAllGitOpsConfig()
	if err != nil {
		impl.logger.Errorw("service err, GetAllGitOpsConfig", "err", err)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}

	// RBAC enforcer applying
	token := r.Header.Get("token")
	if ok := impl.enforcer.Enforce(token, casbin.ResourceGlobal, casbin.ActionGet, "*"); !ok {
		common.WriteJsonResp(w, errors.New("unauthorized"), nil, http.StatusForbidden)
		return
	}
	//RBAC enforcer Ends

	common.WriteJsonResp(w, err, result, http.StatusOK)
}

func (impl GitOpsConfigRestHandlerImpl) GetGitOpsConfigByProvider(w http.ResponseWriter, r *http.Request) {
	userId, err := impl.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	provider := vars["provider"]
	res, err := impl.gitOpsConfigService.GetGitOpsConfigByProvider(provider)
	if err != nil {
		impl.logger.Errorw("service err, GetGitOpsConfigByProvider", "err", err, "provider", provider)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}

	// RBAC enforcer applying

	token := r.Header.Get("token")
	if ok := impl.enforcer.Enforce(token, casbin.ResourceGlobal, casbin.ActionGet, "*"); !ok {
		common.WriteJsonResp(w, errors.New("unauthorized"), nil, http.StatusForbidden)
		return
	}
	// RBAC enforcer Ends

	common.WriteJsonResp(w, err, res, http.StatusOK)
}
func (impl GitOpsConfigRestHandlerImpl) GitOpsValidator(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	userId, err := impl.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	// RBAC enforcer applying
	token := r.Header.Get("token")
	if ok := impl.enforcer.Enforce(token, casbin.ResourceGlobal, casbin.ActionCreate, "*"); !ok {
		common.WriteJsonResp(w, errors.New("unauthorized"), nil, http.StatusForbidden)
		return
	}
	//RBAC enforcer Ends
	var bean bean2.GitOpsConfigDto
	err = decoder.Decode(&bean)
	if bean.Token == "" {
		res, err := impl.gitOpsConfigService.GetGitOpsConfigByProvider(bean.Provider)
		if err != nil {
			impl.logger.Errorw("service err, GetGitOpsConfigByProvider", "err", err, "provider", bean.Provider, "response", res)
			common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
			return
		}
		bean.Token = res.Token
	}
	if err != nil {
		impl.logger.Errorw("request err, ValidateGitOpsConfig", "err", err, "payload", bean)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	bean.UserId = userId
	impl.logger.Infow("request payload, ValidateGitOpsConfig", "err", err, "payload", bean)
	err = impl.validator.Struct(bean)
	if err != nil {
		impl.logger.Errorw("validation err, ValidateGitOpsConfig", "err", err, "payload", bean)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	argoModule, err := impl.moduleReadService.GetModuleInfoByName(moduleBean.ModuleNameArgoCd)
	if err != nil && !errors.Is(err, moduleErr.ModuleNotFoundError) {
		impl.logger.Errorw("error in getting argo module", "error", err)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	detailedErrorGitOpsConfigResponse := impl.gitOpsConfigService.GitOpsValidateDryRun(argoModule.IsInstalled(), &bean)
	common.WriteJsonResp(w, nil, detailedErrorGitOpsConfigResponse, http.StatusOK)
}
