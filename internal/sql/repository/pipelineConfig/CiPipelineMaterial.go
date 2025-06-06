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

package pipelineConfig

import (
	"github.com/devtron-labs/devtron/internal/sql/constants"
	"github.com/devtron-labs/devtron/pkg/build/git/gitMaterial/repository"
	"github.com/devtron-labs/devtron/pkg/sql"
	"github.com/go-pg/pg"
	"go.uber.org/zap"
)

type CiPipelineMaterial struct {
	tableName     struct{} `sql:"ci_pipeline_material" pg:",discard_unknown_columns"`
	Id            int      `sql:"id"`
	GitMaterialId int      `sql:"git_material_id"` //id stored in db GitMaterial( foreign key)
	CiPipelineId  int      `sql:"ci_pipeline_id"`
	Path          string   `sql:"path"` // defaults to root of git repo
	//depricated was used in gocd remove this
	CheckoutPath string               `sql:"checkout_path"` //path where code will be checked out for single source `./` default for multiSource configured by user
	Type         constants.SourceType `sql:"type"`
	Value        string               `sql:"value"`
	ScmId        string               `sql:"scm_id"`      //id of gocd object
	ScmName      string               `sql:"scm_name"`    //gocd scm name
	ScmVersion   string               `sql:"scm_version"` //gocd scm version
	Active       bool                 `sql:"active,notnull"`
	Regex        string               `json:"regex"`
	GitTag       string               `sql:"-"`
	CiPipeline   *CiPipeline
	GitMaterial  *repository.GitMaterial
	sql.AuditLog
}

type CiPipelineMaterialRepository interface {
	Save(tx *pg.Tx, pipeline ...*CiPipelineMaterial) error
	Update(tx *pg.Tx, material ...*CiPipelineMaterial) error
	UpdateNotNull(tx *pg.Tx, material ...*CiPipelineMaterial) error
	FindByCiPipelineIdsIn(ids []int) ([]*CiPipelineMaterial, error)
	GetById(id int) (*CiPipelineMaterial, error)
	GetByIdsIncludeDeleted(ids []int) ([]*CiPipelineMaterial, error)
	GetByPipelineId(id int) ([]*CiPipelineMaterial, error)
	GetRegexByPipelineId(id int) ([]*CiPipelineMaterial, error)
	CheckRegexExistsForMaterial(id int) bool
	GetByPipelineIdForRegexAndFixed(id int) ([]*CiPipelineMaterial, error)
	GetCheckoutPath(gitMaterialId int) (string, error)
	GetByPipelineIdAndGitMaterialId(id int, gitMaterialId int) ([]*CiPipelineMaterial, error)
}

type CiPipelineMaterialRepositoryImpl struct {
	dbConnection *pg.DB
	logger       *zap.SugaredLogger
}

func NewCiPipelineMaterialRepositoryImpl(dbConnection *pg.DB, logger *zap.SugaredLogger) *CiPipelineMaterialRepositoryImpl {
	return &CiPipelineMaterialRepositoryImpl{
		dbConnection: dbConnection,
		logger:       logger,
	}
}

func (impl CiPipelineMaterialRepositoryImpl) GetById(id int) (*CiPipelineMaterial, error) {
	ciPipelineMaterial := &CiPipelineMaterial{}
	err := impl.dbConnection.Model(ciPipelineMaterial).
		Column("ci_pipeline_material.*", "CiPipeline", "GitMaterial").
		Where("ci_pipeline_material.id = ?", id).
		Where("ci_pipeline_material.active = ?", true).
		Select()
	return ciPipelineMaterial, err
}

func (impl CiPipelineMaterialRepositoryImpl) GetByIdsIncludeDeleted(ids []int) ([]*CiPipelineMaterial, error) {
	var ciPipelineMaterials []*CiPipelineMaterial
	if len(ids) == 0 {
		return ciPipelineMaterials, nil
	}
	err := impl.dbConnection.Model(&ciPipelineMaterials).
		Column("ci_pipeline_material.*", "CiPipeline", "CiPipeline.CiTemplate", "CiPipeline.CiTemplate.GitMaterial", "CiPipeline.App", "CiPipeline.CiTemplate.DockerRegistry", "CiPipeline.CiTemplate.CiBuildConfig", "GitMaterial", "GitMaterial.GitProvider").
		Where("ci_pipeline_material.id in (?)", pg.In(ids)).
		Select()
	return ciPipelineMaterials, err
}

func (impl CiPipelineMaterialRepositoryImpl) GetByPipelineId(id int) ([]*CiPipelineMaterial, error) {
	var ciPipelineMaterials []*CiPipelineMaterial
	err := impl.dbConnection.Model(&ciPipelineMaterials).
		Column("ci_pipeline_material.*", "CiPipeline", "CiPipeline.CiTemplate", "CiPipeline.CiTemplate.GitMaterial", "CiPipeline.App", "CiPipeline.CiTemplate.DockerRegistry", "CiPipeline.CiTemplate.CiBuildConfig", "GitMaterial", "GitMaterial.GitProvider").
		Where("ci_pipeline_material.ci_pipeline_id = ?", id).
		Where("ci_pipeline_material.active = ?", true).
		Where("ci_pipeline_material.type != ?", constants.SOURCE_TYPE_BRANCH_REGEX).
		Select()
	return ciPipelineMaterials, err
}

func (impl CiPipelineMaterialRepositoryImpl) GetByPipelineIdAndGitMaterialId(id int, gitMaterialId int) ([]*CiPipelineMaterial, error) {
	var ciPipelineMaterials []*CiPipelineMaterial
	err := impl.dbConnection.Model(&ciPipelineMaterials).
		Column("ci_pipeline_material.*", "CiPipeline", "CiPipeline.CiTemplate", "CiPipeline.CiTemplate.GitMaterial", "CiPipeline.App", "CiPipeline.CiTemplate.DockerRegistry", "CiPipeline.CiTemplate.CiBuildConfig", "GitMaterial", "GitMaterial.GitProvider").
		Where("ci_pipeline_material.ci_pipeline_id = ?", id).
		Where("ci_pipeline_material.active = ?", true).
		Where("ci_pipeline_material.type != ?", constants.SOURCE_TYPE_BRANCH_REGEX).
		Where("ci_pipeline_material.git_material_id =?", gitMaterialId).
		Select()
	return ciPipelineMaterials, err
}

func (impl CiPipelineMaterialRepositoryImpl) GetByPipelineIdForRegexAndFixed(id int) ([]*CiPipelineMaterial, error) {
	var ciPipelineMaterials []*CiPipelineMaterial
	err := impl.dbConnection.Model(&ciPipelineMaterials).
		Column("ci_pipeline_material.*", "CiPipeline", "CiPipeline.CiTemplate", "CiPipeline.CiTemplate.GitMaterial", "CiPipeline.App", "CiPipeline.CiTemplate.DockerRegistry", "CiPipeline.CiTemplate.CiBuildConfig", "GitMaterial", "GitMaterial.GitProvider").
		Where("ci_pipeline_material.ci_pipeline_id = ?", id).
		Where("ci_pipeline_material.active = ?", true).
		Select()
	return ciPipelineMaterials, err
}

func (impl CiPipelineMaterialRepositoryImpl) FindByCiPipelineIdsIn(ids []int) ([]*CiPipelineMaterial, error) {
	var ciPipelineMaterials []*CiPipelineMaterial
	err := impl.dbConnection.Model(&ciPipelineMaterials).
		Column("ci_pipeline_material.*", "CiPipeline", "CiPipeline.CiTemplate", "CiPipeline.CiTemplate.DockerRegistry", "GitMaterial", "GitMaterial.GitProvider").
		Where("ci_pipeline_material.active = ?", true).
		Where("ci_pipeline_material.ci_pipeline_id in (?)", pg.In(ids)).
		Select()
	return ciPipelineMaterials, err
}

func (impl CiPipelineMaterialRepositoryImpl) Save(tx *pg.Tx, material ...*CiPipelineMaterial) error {
	_, err := tx.Model(&material).Insert()
	return err
}

func (impl CiPipelineMaterialRepositoryImpl) Update(tx *pg.Tx, materials ...*CiPipelineMaterial) error {
	_, err := tx.Model(&materials).Update()
	if err != nil {
		return err
	}
	return nil
}
func (impl CiPipelineMaterialRepositoryImpl) UpdateNotNull(tx *pg.Tx, materials ...*CiPipelineMaterial) error {
	/*err := tx.RunInTransaction(func(tx *pg.Tx) error {
		for _, material := range materials {
			r, err := tx.Model(material).WherePK().UpdateNotNull()
			if err != nil {
				return err
			}
			impl.logger.Infof("total rows saved %d", r.RowsAffected())
		}
		return nil
	})*/
	for _, material := range materials {
		_, err := tx.Model(material).WherePK().UpdateNotNull()
		if err != nil {
			impl.logger.Errorw("error in deleting ci pipeline material", "err", err)
			return err
		}
	}

	return nil
}
func (impl CiPipelineMaterialRepositoryImpl) GetRegexByPipelineId(id int) ([]*CiPipelineMaterial, error) {
	var ciPipelineMaterials []*CiPipelineMaterial
	err := impl.dbConnection.Model(&ciPipelineMaterials).
		Column("ci_pipeline_material.*", "CiPipeline", "CiPipeline.CiTemplate", "CiPipeline.CiTemplate.GitMaterial", "CiPipeline.App", "CiPipeline.CiTemplate.DockerRegistry", "CiPipeline.CiTemplate.CiBuildConfig", "GitMaterial", "GitMaterial.GitProvider").
		Where("ci_pipeline_material.ci_pipeline_id = ?", id).
		Where("ci_pipeline_material.active = ?", true).
		Where("ci_pipeline_material.type = ?", constants.SOURCE_TYPE_BRANCH_REGEX).
		Select()
	return ciPipelineMaterials, err
}

func (impl CiPipelineMaterialRepositoryImpl) CheckRegexExistsForMaterial(id int) bool {
	var ciPipelineMaterials []*CiPipelineMaterial
	exists, err := impl.dbConnection.Model(&ciPipelineMaterials).
		Column("ci_pipeline_material.*", "CiPipeline", "CiPipeline.CiTemplate", "CiPipeline.CiTemplate.GitMaterial", "CiPipeline.App", "CiPipeline.CiTemplate.DockerRegistry", "CiPipeline.CiTemplate.CiBuildConfig", "GitMaterial", "GitMaterial.GitProvider").
		Where("ci_pipeline_material.id = ?", id).
		Where("ci_pipeline_material.regex != ?", "").
		Exists()
	if err != nil {
		return false
	}
	return exists
}

func (impl CiPipelineMaterialRepositoryImpl) GetCheckoutPath(gitMaterialId int) (string, error) {
	var checkoutPath string
	err := impl.dbConnection.Model((*repository.GitMaterial)(nil)).
		Column("git_material.checkout_path").
		Where("id=?", gitMaterialId).
		Select(&checkoutPath)
	return checkoutPath, err
}
