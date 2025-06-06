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

package user

import (
	"fmt"
	jwt2 "github.com/devtron-labs/authenticator/jwt"
	"github.com/devtron-labs/devtron/pkg/auth/user/adapter"
	"github.com/devtron-labs/devtron/pkg/auth/user/bean"
	"github.com/devtron-labs/devtron/pkg/auth/user/repository"
	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
)

type UserSelfRegistrationService interface {
	CheckSelfRegistrationRoles() (CheckResponse, error)
	SelfRegister(emailId string) (*bean.UserInfo, error)
	CheckAndCreateUserIfConfigured(claims jwt.MapClaims) bool
}

type UserSelfRegistrationServiceImpl struct {
	logger                          *zap.SugaredLogger
	selfRegistrationRolesRepository repository.SelfRegistrationRolesRepository
	userService                     UserService
}

func NewUserSelfRegistrationServiceImpl(logger *zap.SugaredLogger,
	selfRegistrationRolesRepository repository.SelfRegistrationRolesRepository, userService UserService) *UserSelfRegistrationServiceImpl {
	return &UserSelfRegistrationServiceImpl{
		logger:                          logger,
		selfRegistrationRolesRepository: selfRegistrationRolesRepository,
		userService:                     userService,
	}
}

func (impl *UserSelfRegistrationServiceImpl) GetAllSelfRegistrationRoles() ([]string, error) {
	roleEntries, err := impl.selfRegistrationRolesRepository.GetAll()
	if err != nil {
		impl.logger.Errorf("error fetching all role %+v", err)
		return nil, err
	}
	var roles []string
	for _, role := range roleEntries {
		if role.Role != "" {
			roles = append(roles, role.Role)
		}
	}
	return roles, nil
}

type CheckResponse struct {
	Enabled bool     `json:"enabled"`
	Roles   []string `json:"roles"`
}

func (impl *UserSelfRegistrationServiceImpl) CheckSelfRegistrationRoles() (CheckResponse, error) {
	roleEntries, err := impl.selfRegistrationRolesRepository.GetAll()
	var checkResponse CheckResponse
	if err != nil {
		impl.logger.Errorf("error fetching all role %+v", err)
		checkResponse.Enabled = false
		return checkResponse, err
	}
	//var roles []string
	if roleEntries != nil {
		for _, role := range roleEntries {
			if role.Role != "" {
				checkResponse.Roles = append(checkResponse.Roles, role.Role)
				checkResponse.Enabled = true
				//return checkResponse, err
			}
		}
		if checkResponse.Enabled == true {
			return checkResponse, err
		}
		checkResponse.Enabled = false
		return checkResponse, err
	}
	checkResponse.Enabled = false
	return checkResponse, nil
}

func (impl *UserSelfRegistrationServiceImpl) SelfRegister(emailId string) (*bean.UserInfo, error) {
	roles, err := impl.CheckSelfRegistrationRoles()
	if err != nil || roles.Enabled == false {
		return nil, err
	}
	impl.logger.Infow("self register start")
	userInfo := &bean.UserInfo{
		EmailId:    emailId,
		Roles:      roles.Roles,
		SuperAdmin: false,
	}

	userInfos, err := impl.userService.SelfRegisterUserIfNotExists(adapter.BuildSelfRegisterDto(userInfo))
	if err != nil {
		impl.logger.Errorw("error while register user", "error", err)
		return nil, err
	}
	impl.logger.Errorw("registerd user", "user", userInfos)
	if len(userInfos) > 0 {
		return userInfos[0], nil
	} else {
		return nil, fmt.Errorf("user not created")
	}
}

func (impl *UserSelfRegistrationServiceImpl) CheckAndCreateUserIfConfigured(claims jwt.MapClaims) bool {
	emailId := jwt2.GetField(claims, "email")
	sub := jwt2.GetField(claims, "sub")
	if emailId == "" && sub == "admin" {
		emailId = sub
	}
	exists := impl.userService.UserExists(emailId)
	if !exists {
		impl.logger.Infow("self registering user,  ", "email", emailId)
		user, err := impl.SelfRegister(emailId)
		if err != nil {
			impl.logger.Errorw("error while register user", "error", err)
		} else if user != nil && user.Id > 0 {
			exists = true
		}
	}
	impl.logger.Infow("user status", "email", emailId, "status", exists)
	return exists
}
