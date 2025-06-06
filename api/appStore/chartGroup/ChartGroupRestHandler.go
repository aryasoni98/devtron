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

package chartGroup

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/devtron-labs/devtron/api/restHandler/common"
	"github.com/devtron-labs/devtron/pkg/appStore/chartGroup"
	"github.com/devtron-labs/devtron/pkg/auth/authorisation/casbin"
	"github.com/devtron-labs/devtron/pkg/auth/user"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"gopkg.in/go-playground/validator.v9"
)

const CHART_GROUP_DELETE_SUCCESS_RESP = "Chart group deleted successfully."

type ChartGroupRestHandlerImpl struct {
	ChartGroupService chartGroup.ChartGroupService
	Logger            *zap.SugaredLogger
	userAuthService   user.UserService
	enforcer          casbin.Enforcer
	validator         *validator.Validate
}

func NewChartGroupRestHandlerImpl(ChartGroupService chartGroup.ChartGroupService,
	Logger *zap.SugaredLogger, userAuthService user.UserService,
	enforcer casbin.Enforcer, validator *validator.Validate) *ChartGroupRestHandlerImpl {
	return &ChartGroupRestHandlerImpl{
		ChartGroupService: ChartGroupService,
		Logger:            Logger,
		userAuthService:   userAuthService,
		validator:         validator,
		enforcer:          enforcer,
	}
}

type ChartGroupRestHandler interface {
	CreateChartGroup(w http.ResponseWriter, r *http.Request)
	UpdateChartGroup(w http.ResponseWriter, r *http.Request)
	SaveChartGroupEntries(w http.ResponseWriter, r *http.Request)
	GetChartGroupWithChartMetaData(w http.ResponseWriter, r *http.Request)
	GetChartGroupList(w http.ResponseWriter, r *http.Request)
	GetChartGroupInstallationDetail(w http.ResponseWriter, r *http.Request)
	GetChartGroupListMin(w http.ResponseWriter, r *http.Request)
	DeleteChartGroup(w http.ResponseWriter, r *http.Request)
}

func (impl *ChartGroupRestHandlerImpl) CreateChartGroup(w http.ResponseWriter, r *http.Request) {
	userId, err := impl.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	decoder := json.NewDecoder(r.Body)
	var request chartGroup.ChartGroupBean
	err = decoder.Decode(&request)
	if err != nil {
		impl.Logger.Errorw("request err, CreateChartGroup", "err", err, "payload", request)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	err = impl.validator.Struct(request)
	if err != nil {
		impl.Logger.Errorw("validate err, CreateChartGroup", "err", err, "payload", request)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	request.UserId = userId
	impl.Logger.Infow("request payload, CreateChartGroup", "payload", request)

	//RBAC block starts from here
	token := r.Header.Get("token")
	rbacObject := request.Name
	if ok := impl.enforcer.Enforce(token, casbin.ResourceChartGroup, casbin.ActionCreate, rbacObject); !ok {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusForbidden)
		return
	}
	//RBAC block ends here

	res, err := impl.ChartGroupService.CreateChartGroup(&request)
	if err != nil {
		impl.Logger.Errorw("service err, CreateChartGroup", "err", err, "payload", request)
		statusCode := http.StatusInternalServerError
		if chartGroup.AppNameAlreadyExistsError == err.Error() {
			statusCode = http.StatusBadRequest
		}
		common.WriteJsonResp(w, err, nil, statusCode)
		return
	}
	common.WriteJsonResp(w, err, res, http.StatusOK)
}

func (impl *ChartGroupRestHandlerImpl) UpdateChartGroup(w http.ResponseWriter, r *http.Request) {
	userId, err := impl.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	decoder := json.NewDecoder(r.Body)
	var request chartGroup.ChartGroupBean
	err = decoder.Decode(&request)
	if err != nil {
		impl.Logger.Errorw("request err, UpdateChartGroup", "err", err, "payload", request)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	err = impl.validator.Struct(request)
	if err != nil {
		impl.Logger.Errorw("validate err, UpdateChartGroup", "err", err, "payload", request)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	request.UserId = userId
	impl.Logger.Infow("request payload, UpdateChartGroup", "payload", request)

	//RBAC block starts from here
	token := r.Header.Get("token")
	rbacObject := request.Name
	if ok := impl.enforcer.Enforce(token, casbin.ResourceChartGroup, casbin.ActionUpdate, rbacObject); !ok {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusForbidden)
		return
	}
	//RBAC block ends here

	res, err := impl.ChartGroupService.UpdateChartGroup(&request)
	if err != nil {
		impl.Logger.Errorw("service err, UpdateChartGroup", "err", err, "payload", request)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, res, http.StatusOK)
}

func (impl *ChartGroupRestHandlerImpl) SaveChartGroupEntries(w http.ResponseWriter, r *http.Request) {
	userId, err := impl.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	decoder := json.NewDecoder(r.Body)
	var request chartGroup.ChartGroupBean
	err = decoder.Decode(&request)
	if err != nil {
		impl.Logger.Errorw("request err, SaveChartGroupEntries", "err", err, "payload", request)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	request.UserId = userId
	//RBAC block starts from here
	token := r.Header.Get("token")
	rbacObject := request.Name
	if ok := impl.enforcer.Enforce(token, casbin.ResourceChartGroup, casbin.ActionCreate, rbacObject); !ok {
		if ok1 := impl.enforcer.Enforce(token, casbin.ResourceChartGroup, casbin.ActionUpdate, rbacObject); !ok1 {
			common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusForbidden)
			return
		}
	}
	//RBAC block ends here
	res, err := impl.ChartGroupService.SaveChartGroupEntries(&request)
	if err != nil {
		impl.Logger.Errorw("service err, SaveChartGroupEntries", "err", err, "payload", request)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, res, http.StatusOK)
}

func (impl *ChartGroupRestHandlerImpl) GetChartGroupWithChartMetaData(w http.ResponseWriter, r *http.Request) {
	userId, err := impl.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	chartGroupId, err := strconv.Atoi(vars["chartGroupId"])
	if err != nil {
		impl.Logger.Errorw("request err, GetChartGroupWithChartMetaData", "err", err, "chartGroupId", chartGroupId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}

	//RBAC block starts from here
	token := r.Header.Get("token")
	rbacObject := ""
	if ok := impl.enforcer.Enforce(token, casbin.ResourceChartGroup, casbin.ActionGet, rbacObject); !ok {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusForbidden)
		return
	}
	//RBAC block ends here

	res, err := impl.ChartGroupService.GetChartGroupWithChartMetaData(chartGroupId)
	if err != nil {
		impl.Logger.Errorw("service err, GetChartGroupWithChartMetaData", "err", err, "chartGroupId", chartGroupId)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, res, http.StatusOK)
}

func (impl *ChartGroupRestHandlerImpl) GetChartGroupInstallationDetail(w http.ResponseWriter, r *http.Request) {
	userId, err := impl.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	chartGroupId, err := strconv.Atoi(vars["chartGroupId"])
	if err != nil {
		impl.Logger.Errorw("request err, GetChartGroupInstallationDetail", "err", err, "chartGroupId", chartGroupId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}

	//RBAC block starts from here
	token := r.Header.Get("token")
	rbacObject := ""
	if ok := impl.enforcer.Enforce(token, casbin.ResourceChartGroup, casbin.ActionGet, rbacObject); !ok {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusForbidden)
		return
	}
	//RBAC block ends here

	res, err := impl.ChartGroupService.GetChartGroupWithInstallationDetail(chartGroupId)
	if err != nil {
		impl.Logger.Errorw("service err, GetChartGroupInstallationDetail", "err", err, "chartGroupId", chartGroupId)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, res, http.StatusOK)
}

func (impl *ChartGroupRestHandlerImpl) GetChartGroupList(w http.ResponseWriter, r *http.Request) {
	userId, err := impl.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}

	//RBAC block starts from here
	token := r.Header.Get("token")
	rbacObject := ""
	if ok := impl.enforcer.Enforce(token, casbin.ResourceChartGroup, casbin.ActionGet, rbacObject); !ok {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusForbidden)
		return
	}
	//RBAC block ends here

	max := r.FormValue("max")
	maxCount := 0
	if len(max) > 0 {
		maxCount, err = strconv.Atoi(max)
		if err != nil {
			impl.Logger.Errorw("request err, GetChartGroupList", "err", err, "max", max)
			common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
			return
		}
	}
	res, err := impl.ChartGroupService.GetChartGroupList(maxCount)
	if err != nil {
		impl.Logger.Errorw("service err, GetChartGroupList", "err", err, "max", max)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, res, http.StatusOK)
}

func (impl *ChartGroupRestHandlerImpl) GetChartGroupListMin(w http.ResponseWriter, r *http.Request) {
	userId, err := impl.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}

	//RBAC block starts from here
	token := r.Header.Get("token")
	rbacObject := ""
	if ok := impl.enforcer.Enforce(token, casbin.ResourceChartGroup, casbin.ActionGet, rbacObject); !ok {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusForbidden)
		return
	}
	//RBAC block ends here

	max := r.FormValue("max")
	maxCount := 0
	if len(max) > 0 {
		maxCount, err = strconv.Atoi(max)
		if err != nil {
			impl.Logger.Errorw("request err, GetChartGroupListMin", "err", err, "max", max)
			common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
			return
		}
	}
	res, err := impl.ChartGroupService.ChartGroupListMin(maxCount)
	if err != nil {
		impl.Logger.Errorw("service err, GetChartGroupListMin", "err", err, "max", max)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, res, http.StatusOK)
}

func (impl *ChartGroupRestHandlerImpl) DeleteChartGroup(w http.ResponseWriter, r *http.Request) {
	userId, err := impl.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	decoder := json.NewDecoder(r.Body)
	var request chartGroup.ChartGroupBean
	err = decoder.Decode(&request)
	if err != nil {
		impl.Logger.Errorw("request err, DeleteChartGroup", "err", err, "payload", request)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	err = impl.validator.Struct(request)
	if err != nil {
		impl.Logger.Errorw("validate err, DeleteChartGroup", "err", err, "payload", request)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	request.UserId = userId
	impl.Logger.Infow("request payload, DeleteChartGroup", "payload", request)

	//RBAC block starts from here
	token := r.Header.Get("token")
	rbacObject := request.Name
	if ok := impl.enforcer.Enforce(token, casbin.ResourceChartGroup, casbin.ActionCreate, rbacObject); !ok {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusForbidden)
		return
	}
	//RBAC block ends here
	err = impl.ChartGroupService.DeleteChartGroup(&request)
	if err != nil {
		impl.Logger.Errorw("service err, DeleteChartGroup", "err", err, "payload", request)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, CHART_GROUP_DELETE_SUCCESS_RESP, http.StatusOK)
}
