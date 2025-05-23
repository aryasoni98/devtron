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

package chartConfig

import (
	"github.com/devtron-labs/devtron/pkg/chartRepo/repository"
	"github.com/devtron-labs/devtron/pkg/sql"
	"github.com/go-pg/pg"
	"github.com/juju/errors"
)

type PipelineStrategy struct {
	tableName  struct{}                               `sql:"pipeline_strategy" pg:",discard_unknown_columns"`
	Id         int                                    `sql:"id,pk"`
	PipelineId int                                    `sql:"pipeline_id"`
	Strategy   chartRepoRepository.DeploymentStrategy `sql:"strategy,notnull"`
	Config     string                                 `sql:"config"`
	Default    bool                                   `sql:"default,notnull"`
	Deleted    bool                                   `sql:"deleted,notnull"`
	sql.AuditLog
}

type PipelineConfigRepository interface {
	Save(pipelineStrategy *PipelineStrategy, tx *pg.Tx) error
	Update(pipelineStrategy *PipelineStrategy, tx *pg.Tx) error
	FindById(id int) (chart *PipelineStrategy, err error)
	FindByStrategy(strategy chartRepoRepository.DeploymentStrategy) (pipelineStrategy *PipelineStrategy, err error)
	FindByStrategyAndPipelineId(strategy chartRepoRepository.DeploymentStrategy, pipelineId int) (pipelineStrategy *PipelineStrategy, err error)
	GetAllStrategyByPipelineId(pipelineId int) ([]*PipelineStrategy, error)
	GetDefaultStrategyByPipelineId(pipelineId int) (pipelineStrategy *PipelineStrategy, err error)
	MarkAsDeleted(pipelineStrategy *PipelineStrategy, userId int32, tx *pg.Tx) error
	GetAllStrategyByPipelineIds(pipelineIds []int) ([]*PipelineStrategy, error)
}

type PipelineConfigRepositoryImpl struct {
	dbConnection *pg.DB
}

func NewPipelineConfigRepository(dbConnection *pg.DB) *PipelineConfigRepositoryImpl {
	return &PipelineConfigRepositoryImpl{dbConnection: dbConnection}
}

func (impl PipelineConfigRepositoryImpl) Save(pipelineStrategy *PipelineStrategy, tx *pg.Tx) error {
	return tx.Insert(pipelineStrategy)
}

func (impl PipelineConfigRepositoryImpl) Update(pipelineStrategy *PipelineStrategy, tx *pg.Tx) error {
	_, err := tx.Model(pipelineStrategy).WherePK().UpdateNotNull()
	return err
}

func (impl PipelineConfigRepositoryImpl) FindById(id int) (pipelineStrategy *PipelineStrategy, err error) {
	pipelineStrategy = &PipelineStrategy{}
	err = impl.dbConnection.Model(pipelineStrategy).
		Where("id = ?", id).
		Where("deleted = ?", false).
		Select()
	return pipelineStrategy, err
}

func (impl PipelineConfigRepositoryImpl) FindByStrategy(strategy chartRepoRepository.DeploymentStrategy) (pipelineStrategy *PipelineStrategy, err error) {
	pipelineStrategy = &PipelineStrategy{}
	err = impl.dbConnection.Model(pipelineStrategy).
		Where("strategy = ?", strategy).
		Where("deleted = ?", false).
		Select()
	return pipelineStrategy, err
}

func (impl PipelineConfigRepositoryImpl) FindByStrategyAndPipelineId(strategy chartRepoRepository.DeploymentStrategy, pipelineId int) (pipelineStrategy *PipelineStrategy, err error) {
	pipelineStrategy = &PipelineStrategy{}
	err = impl.dbConnection.Model(pipelineStrategy).
		Where("strategy = ?", strategy).
		Where("pipeline_id = ?", pipelineId).
		Where("deleted = ?", false).
		Select()
	return pipelineStrategy, err
}

// GetAllStrategyByPipelineId -
// it will return for multiple pipeline strategies for a pipeline
// per pipeline single pipeline strategy (BLUE_GREEN, CANARY, ROLLING, RECREATE) can be there
func (impl PipelineConfigRepositoryImpl) GetAllStrategyByPipelineId(pipelineId int) ([]*PipelineStrategy, error) {
	var pipelineStrategies []*PipelineStrategy
	err := impl.dbConnection.
		Model(&pipelineStrategies).
		Where("pipeline_id = ?", pipelineId).
		Where("deleted = ?", false).
		Select()
	if pg.ErrNoRows == err {
		return nil, errors.NotFoundf(err.Error())
	}
	return pipelineStrategies, err
}

// GetDefaultStrategyByPipelineId -
// it will return single latest pipeline strategy for the requested pipeline
func (impl PipelineConfigRepositoryImpl) GetDefaultStrategyByPipelineId(pipelineId int) (pipelineStrategy *PipelineStrategy, err error) {
	pipelineStrategy = &PipelineStrategy{}
	err = impl.dbConnection.
		Model(pipelineStrategy).
		Where("pipeline_strategy.pipeline_id = ?", pipelineId).
		Where("pipeline_strategy.default = ?", true).
		Where("pipeline_strategy.deleted = ?", false).
		Select()
	if pg.ErrNoRows == err {
		return nil, errors.NotFoundf(err.Error())
	}
	return pipelineStrategy, err
}

// MarkAsDeleted -
// it will soft-delete the pipeline strategy from the database
func (impl PipelineConfigRepositoryImpl) MarkAsDeleted(pipelineStrategy *PipelineStrategy, userId int32, tx *pg.Tx) error {
	pipelineStrategy.Deleted = true
	pipelineStrategy.UpdateAuditLog(userId)
	return impl.Update(pipelineStrategy, tx)
}

func (impl PipelineConfigRepositoryImpl) GetAllStrategyByPipelineIds(pipelineIds []int) ([]*PipelineStrategy, error) {
	if len(pipelineIds) == 0 {
		return nil, errors.NotFoundf("pipeline strategies")
	}
	var pipelineStrategies []*PipelineStrategy
	err := impl.dbConnection.
		Model(&pipelineStrategies).
		Where("pipeline_id in (?)", pg.In(pipelineIds)).
		Where("deleted = ?", false).
		Select()
	if pg.ErrNoRows == err {
		return nil, errors.NotFoundf(err.Error())
	}
	return pipelineStrategies, err
}
