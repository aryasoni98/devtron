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

package repository

import (
	"github.com/devtron-labs/devtron/pkg/sql"
	"github.com/go-pg/pg"
	"go.uber.org/zap"
	"time"
)

type DeploymentTemplateHistoryRepository interface {
	CreateHistory(chart *DeploymentTemplateHistory) (*DeploymentTemplateHistory, error)
	CreateHistoryWithTxn(chart *DeploymentTemplateHistory, tx *pg.Tx) (*DeploymentTemplateHistory, error)
	GetHistoryForDeployedTemplateById(id, pipelineId int) (*DeploymentTemplateHistory, error)
	GetHistoryByPipelineIdAndWfrId(pipelineId, wfrId int) (*DeploymentTemplateHistory, error)
	GetDeployedHistoryForPipelineIdOnTime(pipelineId int, deployedOn time.Time) (*DeploymentTemplateHistory, error)
	GetDeployedHistoryList(pipelineId, baseConfigId int) ([]*DeploymentTemplateHistory, error)
	GetDeployedOnByDeploymentTemplateAndPipelineId(id, pipelineId int) (time.Time, error)
}

type DeploymentTemplateHistoryRepositoryImpl struct {
	dbConnection *pg.DB
	logger       *zap.SugaredLogger
}

func NewDeploymentTemplateHistoryRepositoryImpl(logger *zap.SugaredLogger, dbConnection *pg.DB) *DeploymentTemplateHistoryRepositoryImpl {
	return &DeploymentTemplateHistoryRepositoryImpl{dbConnection: dbConnection, logger: logger}
}

type DeploymentTemplateHistory struct {
	tableName               struct{}  `sql:"deployment_template_history" pg:",discard_unknown_columns"`
	Id                      int       `sql:"id,pk"`
	PipelineId              int       `sql:"pipeline_id"`
	AppId                   int       `sql:"app_id"`
	ImageDescriptorTemplate string    `sql:"image_descriptor_template"`
	Template                string    `sql:"template"`
	TargetEnvironment       int       `sql:"target_environment"`
	TemplateName            string    `sql:"template_name"`
	TemplateVersion         string    `sql:"template_version"`
	IsAppMetricsEnabled     bool      `sql:"is_app_metrics_enabled,notnull"`
	Deployed                bool      `sql:"deployed"`
	DeployedOn              time.Time `sql:"deployed_on"`
	DeployedBy              int32     `sql:"deployed_by"`
	MergeStrategy           string    `sql:"merge_strategy"`
	PipelineIds             []int     `sql:"pipeline_ids,array"`
	sql.AuditLog
	//getting below data from cd_workflow_runner and users join
	DeploymentStatus  string `sql:"-"`
	DeployedByEmailId string `sql:"-"`
}

func (impl DeploymentTemplateHistoryRepositoryImpl) CreateHistory(chart *DeploymentTemplateHistory) (*DeploymentTemplateHistory, error) {
	err := impl.dbConnection.Insert(chart)
	if err != nil {
		impl.logger.Errorw("err in creating deployment template history entry", "err", err, "history", chart)
		return chart, err
	}
	return chart, nil
}

func (impl DeploymentTemplateHistoryRepositoryImpl) CreateHistoryWithTxn(chart *DeploymentTemplateHistory, tx *pg.Tx) (*DeploymentTemplateHistory, error) {
	err := tx.Insert(chart)
	if err != nil {
		impl.logger.Errorw("err in creating deployment template history entry", "err", err, "history", chart)
		return chart, err
	}
	return chart, nil
}

func (impl DeploymentTemplateHistoryRepositoryImpl) GetHistoryForDeployedTemplateById(id, pipelineId int) (*DeploymentTemplateHistory, error) {
	var history DeploymentTemplateHistory
	err := impl.dbConnection.Model(&history).Where("id = ?", id).
		Where("pipeline_id = ?", pipelineId).
		Where("deployed = ?", true).Select()
	if err != nil {
		impl.logger.Errorw("error in getting deployment template history", "err", err)
		return &history, err
	}
	return &history, nil
}

func (impl DeploymentTemplateHistoryRepositoryImpl) GetHistoryByPipelineIdAndWfrId(pipelineId, wfrId int) (*DeploymentTemplateHistory, error) {
	var history DeploymentTemplateHistory
	err := impl.dbConnection.Model(&history).Join("INNER JOIN cd_workflow_runner cwr ON cwr.started_on = deployment_template_history.deployed_on").
		Where("deployment_template_history.pipeline_id = ?", pipelineId).
		Where("deployment_template_history.deployed = ?", true).
		Where("cwr.id = ?", wfrId).
		Select()
	if err != nil {
		impl.logger.Errorw("error in getting deployment template history by pipelineId & wfrId", "err", err, "pipelineId", pipelineId, "wfrId", wfrId)
		return &history, err
	}
	return &history, nil
}

func (impl DeploymentTemplateHistoryRepositoryImpl) GetDeployedHistoryList(pipelineId, baseConfigId int) ([]*DeploymentTemplateHistory, error) {
	var histories []*DeploymentTemplateHistory
	query := "SELECT dth.id, dth.deployed_on, dth.deployed_by, cwr.status as deployment_status, users.email_id as deployed_by_email_id" +
		" FROM deployment_template_history dth" +
		" INNER JOIN cd_workflow_runner cwr ON cwr.started_on = dth.deployed_on" +
		" INNER JOIN users ON users.id = dth.deployed_by" +
		" WHERE dth.pipeline_id = ? AND dth.deployed = true AND dth.id <= ?" +
		" ORDER BY dth.id DESC;"
	_, err := impl.dbConnection.Query(&histories, query, pipelineId, baseConfigId)
	if err != nil {
		impl.logger.Errorw("error in getting deployment template history list by pipelineId", "err", err, "pipelineId", pipelineId)
		return histories, err
	}
	return histories, nil
}

func (impl DeploymentTemplateHistoryRepositoryImpl) GetDeployedHistoryForPipelineIdOnTime(pipelineId int, deployedOn time.Time) (*DeploymentTemplateHistory, error) {
	var history DeploymentTemplateHistory
	err := impl.dbConnection.Model(&history).
		Where("deployment_template_history.deployed_on = ?", deployedOn).
		Where("deployment_template_history.pipeline_id = ?", pipelineId).
		Where("deployment_template_history.deployed = ?", true).
		Select()
	return &history, err
}

func (impl DeploymentTemplateHistoryRepositoryImpl) GetDeployedOnByDeploymentTemplateAndPipelineId(id, pipelineId int) (time.Time, error) {
	var deployedOn time.Time
	err := impl.dbConnection.Model((*DeploymentTemplateHistory)(nil)).Column("deployed_on").Where("id = ?", id).
		Where("pipeline_id = ?", pipelineId).
		Where("deployed = ?", true).Select(&deployedOn)
	if err != nil {
		impl.logger.Errorw("error in getting deployed on by deploymentTemplateHistoryId and pipelineId", "err", err)
		return time.Time{}, err
	}
	return deployedOn, nil
}
