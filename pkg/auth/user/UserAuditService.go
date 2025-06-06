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

package user

import (
	"time"

	repository2 "github.com/devtron-labs/devtron/pkg/auth/user/repository"
	"github.com/go-pg/pg"
	"go.uber.org/zap"
)

type UserAudit struct {
	UserId    int32
	ClientIp  string
	CreatedOn time.Time
	UpdatedOn time.Time
}

type UserAuditService interface {
	Save(userAudit *UserAudit) error
	GetLatestByUserId(userId int32) (*UserAudit, error)
	GetLatestUser() (*UserAudit, error)
	Update(userAudit *UserAudit) error
	GetActiveUsersCountInLast30Days() (int, error)
}

type UserAuditServiceImpl struct {
	logger              *zap.SugaredLogger
	userAuditRepository repository2.UserAuditRepository
}

func NewUserAuditServiceImpl(logger *zap.SugaredLogger, userAuditRepository repository2.UserAuditRepository) *UserAuditServiceImpl {
	return &UserAuditServiceImpl{
		logger:              logger,
		userAuditRepository: userAuditRepository,
	}
}

func (impl UserAuditServiceImpl) Update(userAudit *UserAudit) error {
	userId := userAudit.UserId
	impl.logger.Infow("Saving user audit", "userId", userId)
	userAuditDb := &repository2.UserAudit{
		UserId:   userId,
		ClientIp: userAudit.ClientIp,
	}
	err := impl.userAuditRepository.Update(userAuditDb)
	if err != nil {
		impl.logger.Errorw("error while updating user audit log", "userId", userId, "error", err)
		return err
	}
	return nil
}

func (impl UserAuditServiceImpl) Save(userAudit *UserAudit) error {
	userId := userAudit.UserId
	impl.logger.Infow("Saving user audit", "userId", userId)
	userAuditDb := &repository2.UserAudit{
		UserId:    userId,
		ClientIp:  userAudit.ClientIp,
		CreatedOn: userAudit.CreatedOn,
		UpdatedOn: userAudit.UpdatedOn,
	}
	err := impl.userAuditRepository.Save(userAuditDb)
	if err != nil {
		impl.logger.Errorw("error while saving user audit log", "userId", userId, "error", err)
		return err
	}
	return nil
}

func (impl UserAuditServiceImpl) GetLatestByUserId(userId int32) (*UserAudit, error) {
	impl.logger.Infow("Getting latest user audit", "userId", userId)
	userAuditDb, err := impl.userAuditRepository.GetLatestByUserId(userId)

	if err != nil {
		if err == pg.ErrNoRows {
			return nil, nil
		} else {
			impl.logger.Errorw("error while getting latest audit log", "userId", userId, "error", err)
			return nil, err
		}
	}

	return &UserAudit{
		UserId:    userId,
		ClientIp:  userAuditDb.ClientIp,
		CreatedOn: userAuditDb.CreatedOn,
		UpdatedOn: userAuditDb.UpdatedOn,
	}, nil
}

func (impl UserAuditServiceImpl) GetLatestUser() (*UserAudit, error) {
	impl.logger.Info("Getting latest user audit")
	userAuditDb, err := impl.userAuditRepository.GetLatestUser()

	if err != nil {
		if err == pg.ErrNoRows {
			impl.logger.Errorw("no user audits found", "err", err)
		} else {
			impl.logger.Errorw("error while getting latest user audit log", "err", err)
		}
		return nil, err
	}

	return &UserAudit{
		UserId:    userAuditDb.UserId,
		ClientIp:  userAuditDb.ClientIp,
		CreatedOn: userAuditDb.CreatedOn,
		UpdatedOn: userAuditDb.UpdatedOn,
	}, nil
}

func (impl UserAuditServiceImpl) GetActiveUsersCountInLast30Days() (int, error) {
	impl.logger.Info("Getting active users count in last 30 days")
	count, err := impl.userAuditRepository.GetActiveUsersCountInLast30Days()
	if err != nil {
		impl.logger.Errorw("error while getting active users count in last 30 days", "err", err)
		return 0, err
	}
	return count, nil
}
