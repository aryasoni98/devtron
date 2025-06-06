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

package executors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	argoWfApiV1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	argoWfClientV1 "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
	apiBean "github.com/devtron-labs/devtron/api/bean"
	"github.com/devtron-labs/devtron/pkg/pipeline/bean"
	"github.com/devtron-labs/devtron/pkg/pipeline/executors/adapter"
	"github.com/devtron-labs/devtron/pkg/pipeline/types"
	"go.uber.org/zap"
	k8sApiV1 "k8s.io/api/core/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
	"net/url"
)

type WorkflowExecutor interface {
	ExecuteWorkflow(workflowTemplate bean.WorkflowTemplate) (*unstructured.UnstructuredList, error)
	TerminateWorkflow(workflowName string, namespace string, clusterConfig *rest.Config) error
	GetWorkflow(workflowName string, namespace string, clusterConfig *rest.Config) (*unstructured.UnstructuredList, error)
	GetWorkflowStatus(workflowName string, namespace string, clusterConfig *rest.Config) (*types.WorkflowStatus, error)
	TerminateDanglingWorkflow(workflowGenerateName string, namespace string, clusterConfig *rest.Config) error
}

type ArgoWorkflowExecutor interface {
	WorkflowExecutor
}

type ArgoWorkflowExecutorImpl struct {
	logger *zap.SugaredLogger
}

func NewArgoWorkflowExecutorImpl(logger *zap.SugaredLogger) *ArgoWorkflowExecutorImpl {
	return &ArgoWorkflowExecutorImpl{logger: logger}
}

func (impl *ArgoWorkflowExecutorImpl) TerminateWorkflow(workflowName string, namespace string, clusterConfig *rest.Config) error {
	impl.logger.Debugw("terminating wf", "name", workflowName)
	wfClient, err := impl.getClientInstance(namespace, clusterConfig)
	if err != nil {
		impl.logger.Errorw("cannot build wf client", "wfName", workflowName, "err", err)
		return err
	}
	_, err = wfClient.Get(context.Background(), workflowName, k8sMetaV1.GetOptions{})
	if err != nil {
		impl.logger.Errorw("cannot find workflow", "name", workflowName, "err", err)
		return errors.New("cannot find workflow " + workflowName)
	}
	err = util.TerminateWorkflow(context.Background(), wfClient, workflowName)
	return err
}

func (impl *ArgoWorkflowExecutorImpl) TerminateDanglingWorkflow(workflowGenerateName string, namespace string, clusterConfig *rest.Config) error {
	impl.logger.Debugw("terminating dangling wf", "workflowGenerateName", workflowGenerateName)
	wfClient, err := impl.getClientInstance(namespace, clusterConfig)
	if err != nil {
		impl.logger.Errorw("cannot build wf client", "workflowGenerateName", workflowGenerateName, "err", err)
		return err
	}
	jobSelectorLabel := fmt.Sprintf("%s=%s", bean.WorkflowGenerateNamePrefix, workflowGenerateName)
	wfList, err := wfClient.List(context.Background(), k8sMetaV1.ListOptions{LabelSelector: jobSelectorLabel})
	if err != nil {
		impl.logger.Errorw("error in fetching list of workflows", "namespace", namespace, "err", err)
		return err
	}
	for _, wf := range wfList.Items {
		err = util.TerminateWorkflow(context.Background(), wfClient, wf.Name)
		if err != nil {
			impl.logger.Errorw("error in terminating argo executor workflow", "name", wf.Name, "err", err)
			return err
		}
	}
	return nil
}

func (impl *ArgoWorkflowExecutorImpl) ExecuteWorkflow(workflowTemplate bean.WorkflowTemplate) (*unstructured.UnstructuredList, error) {

	entryPoint := workflowTemplate.WorkflowType
	// get cm and cs argo step templates
	templates, err := impl.getArgoTemplates(workflowTemplate.ConfigMaps, workflowTemplate.Secrets, workflowTemplate.WorkflowType == bean.CI_WORKFLOW_NAME)
	if err != nil {
		impl.logger.Errorw("error occurred while fetching argo templates and steps", "err", err)
		return nil, err
	}
	if len(templates) > 0 {
		entryPoint = workflowTemplate.GetEntrypoint()
	}

	wfContainer := workflowTemplate.Containers[0]
	ciCdTemplate := argoWfApiV1.Template{
		Name:      workflowTemplate.WorkflowType,
		Container: &wfContainer,
		ActiveDeadlineSeconds: &intstr.IntOrString{
			IntVal: int32(*workflowTemplate.ActiveDeadlineSeconds),
		},
	}
	impl.updateBlobStorageConfig(workflowTemplate, &ciCdTemplate)
	templates = append(templates, ciCdTemplate)

	objectMeta := workflowTemplate.CreateObjectMetadata()

	var (
		ciCdWorkflow = argoWfApiV1.Workflow{
			ObjectMeta: *objectMeta,
			Spec: argoWfApiV1.WorkflowSpec{
				ServiceAccountName: workflowTemplate.ServiceAccountName,
				NodeSelector:       workflowTemplate.NodeSelector,
				Tolerations:        workflowTemplate.Tolerations,
				Entrypoint:         entryPoint,
				TTLStrategy: &argoWfApiV1.TTLStrategy{
					SecondsAfterCompletion: workflowTemplate.TTLValue,
				},
				Templates: templates,
				Volumes:   workflowTemplate.Volumes,
				PodGC: &argoWfApiV1.PodGC{
					Strategy: argoWfApiV1.PodGCOnWorkflowCompletion,
				},
			},
		}
	)

	wfTemplate, err := json.Marshal(ciCdWorkflow)
	if err != nil {
		impl.logger.Errorw("error occurred while marshalling json", "err", err)
		return nil, err
	}
	impl.logger.Debugw("workflow request to submit", "wf", string(wfTemplate))

	wfClient, err := impl.getClientInstance(workflowTemplate.Namespace, workflowTemplate.ClusterConfig)
	if err != nil {
		impl.logger.Errorw("cannot build wf client", "err", err)
		return nil, err
	}

	createdWf, err := wfClient.Create(context.Background(), &ciCdWorkflow, k8sMetaV1.CreateOptions{})
	if err != nil {
		impl.logger.Errorw("error in wf trigger", "err", err)
		return nil, err
	}
	impl.logger.Debugw("workflow submitted: ", "name", createdWf.Name)
	return impl.convertToUnstructured(createdWf), nil
}

func (impl *ArgoWorkflowExecutorImpl) GetWorkflow(workflowName string, namespace string, clusterConfig *rest.Config) (*unstructured.UnstructuredList, error) {

	wf, err := impl.getWorkflow(workflowName, namespace, clusterConfig)
	if err != nil {
		return nil, err
	}
	return impl.convertToUnstructured(wf), err
}

func (impl *ArgoWorkflowExecutorImpl) GetWorkflowStatus(workflowName string, namespace string, clusterConfig *rest.Config) (*types.WorkflowStatus, error) {
	wf, err := impl.getWorkflow(workflowName, namespace, clusterConfig)
	if err != nil {
		return nil, err
	}
	wfStatus := &types.WorkflowStatus{
		Status:  string(wf.Status.Phase),
		Message: wf.Status.Message,
	}
	return wfStatus, err
}

func (impl *ArgoWorkflowExecutorImpl) getWorkflow(workflowName string, namespace string, clusterConfig *rest.Config) (*argoWfApiV1.Workflow, error) {
	wfClient, err := impl.getClientInstance(namespace, clusterConfig)
	if err != nil {
		impl.logger.Errorw("cannot build wf client", "wfName", workflowName, "err", err)
		return nil, err
	}
	wf, err := wfClient.Get(context.Background(), workflowName, k8sMetaV1.GetOptions{})
	if err != nil {
		impl.logger.Errorw("cannot find workflow", "name", workflowName, "err", err)
		return nil, fmt.Errorf("cannot find workflow %s", workflowName)
	}
	return wf, nil
}

func (impl *ArgoWorkflowExecutorImpl) convertToUnstructured(cdWorkflow interface{}) *unstructured.UnstructuredList {
	unstructedObjMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&cdWorkflow)
	if err != nil {
		return nil
	}
	unstructuredObj := unstructured.Unstructured{Object: unstructedObjMap}
	unstructuredList := &unstructured.UnstructuredList{Items: []unstructured.Unstructured{unstructuredObj}}
	return unstructuredList
}

func (impl *ArgoWorkflowExecutorImpl) updateBlobStorageConfig(workflowTemplate bean.WorkflowTemplate, cdTemplate *argoWfApiV1.Template) {
	cdTemplate.ArchiveLocation = &argoWfApiV1.ArtifactLocation{
		ArchiveLogs: &workflowTemplate.ArchiveLogs,
	}
	if workflowTemplate.BlobStorageConfigured {
		var s3Artifact *argoWfApiV1.S3Artifact
		var gcsArtifact *argoWfApiV1.GCSArtifact
		blobStorageS3Config := workflowTemplate.BlobStorageS3Config
		gcpBlobConfig := workflowTemplate.GcpBlobConfig
		cloudStorageKey := workflowTemplate.CloudStorageKey
		if blobStorageS3Config != nil {
			s3CompatibleEndpointUrl := blobStorageS3Config.EndpointUrl
			if s3CompatibleEndpointUrl == "" {
				s3CompatibleEndpointUrl = "s3.amazonaws.com"
			} else {
				parsedUrl, err := url.Parse(s3CompatibleEndpointUrl)
				if err != nil {
					impl.logger.Errorw("error occurred while parsing s3CompatibleEndpointUrl, ", "s3CompatibleEndpointUrl", s3CompatibleEndpointUrl, "err", err)
				} else {
					s3CompatibleEndpointUrl = parsedUrl.Host
				}
			}
			isInsecure := blobStorageS3Config.IsInSecure
			var accessKeySelector *k8sApiV1.SecretKeySelector
			var secretKeySelector *k8sApiV1.SecretKeySelector
			if blobStorageS3Config.AccessKey != "" {
				accessKeySelector = AccessKeySelector
				secretKeySelector = SecretKeySelector
			}
			s3Artifact = &argoWfApiV1.S3Artifact{
				Key: cloudStorageKey,
				S3Bucket: argoWfApiV1.S3Bucket{
					Endpoint:        s3CompatibleEndpointUrl,
					AccessKeySecret: accessKeySelector,
					SecretKeySecret: secretKeySelector,
					Bucket:          blobStorageS3Config.CiLogBucketName,
					Insecure:        &isInsecure,
				},
			}
			if blobStorageS3Config.CiLogRegion != "" {
				//TODO checking for Azure
				s3Artifact.Region = blobStorageS3Config.CiLogRegion
			}
		} else if gcpBlobConfig != nil {
			gcsArtifact = &argoWfApiV1.GCSArtifact{
				Key: cloudStorageKey,
				GCSBucket: argoWfApiV1.GCSBucket{
					Bucket:                  gcpBlobConfig.LogBucketName,
					ServiceAccountKeySecret: SecretKeySelector,
				},
			}
		}

		// set in ArchiveLocation
		cdTemplate.ArchiveLocation.S3 = s3Artifact
		cdTemplate.ArchiveLocation.GCS = gcsArtifact
	}
}

func (impl *ArgoWorkflowExecutorImpl) getArgoTemplates(configMaps []apiBean.ConfigSecretMap, secrets []apiBean.ConfigSecretMap, isCi bool) ([]argoWfApiV1.Template, error) {
	var templates []argoWfApiV1.Template
	var steps []argoWfApiV1.ParallelSteps
	cmIndex := 0
	csIndex := 0
	for _, configMap := range configMaps {
		if configMap.External {
			continue
		}
		parallelStep, argoTemplate, err := impl.appendCMCSToStepAndTemplate(false, configMap, cmIndex)
		if err != nil {
			return templates, err
		}
		steps = append(steps, parallelStep)
		templates = append(templates, argoTemplate)
		cmIndex++
	}
	for _, secret := range secrets {
		if secret.External {
			continue
		}
		parallelStep, argoTemplate, err := impl.appendCMCSToStepAndTemplate(true, secret, csIndex)
		if err != nil {
			return templates, err
		}
		steps = append(steps, parallelStep)
		templates = append(templates, argoTemplate)
		csIndex++
	}
	if len(templates) <= 0 {
		return templates, nil
	}
	stepName := bean.CD_WORKFLOW_NAME
	templateName := bean.CD_WORKFLOW_WITH_STAGES
	if isCi {
		stepName = bean.CI_WORKFLOW_NAME
		templateName = bean.CI_WORKFLOW_WITH_STAGES
	}

	steps = append(steps, argoWfApiV1.ParallelSteps{
		Steps: []argoWfApiV1.WorkflowStep{
			{
				Name:     "run-wf",
				Template: stepName,
			},
		},
	})

	templates = append(templates, argoWfApiV1.Template{
		Name:  templateName,
		Steps: steps,
	})

	return templates, nil
}

func (impl *ArgoWorkflowExecutorImpl) getConfigMapOrSecretJson(configMapSecretDto types.ConfigMapSecretDto, isSecret bool) (cmSecretJson string, err error) {
	if isSecret {
		cmSecretJson, err = adapter.GetSecretJson(configMapSecretDto)
		if err != nil {
			impl.logger.Errorw("error occurred while extracting cm/secret json", "secretName", configMapSecretDto.Name, "err", err)
			return cmSecretJson, err
		}
	} else {
		cmSecretJson, err = adapter.GetConfigMapJson(configMapSecretDto)
		if err != nil {
			impl.logger.Errorw("error occurred while extracting cm/secret json", "configMapName", configMapSecretDto.Name, "err", err)
			return cmSecretJson, err
		}
	}
	return cmSecretJson, nil
}

func (impl *ArgoWorkflowExecutorImpl) appendCMCSToStepAndTemplate(isSecret bool, configSecretMap apiBean.ConfigSecretMap, cmSecretIndex int) (parallelStep argoWfApiV1.ParallelSteps, argoTemplate argoWfApiV1.Template, err error) {
	configMapSecretDto, err := adapter.GetConfigMapSecretDto(configSecretMap, ArgoWorkflowOwnerRef, isSecret)
	if err != nil {
		impl.logger.Errorw("error occurred while extracting config map secret dto", "configSecretName", configSecretMap.Name, "err", err)
		return parallelStep, argoTemplate, err
	}
	cmSecretJson, err := impl.getConfigMapOrSecretJson(configMapSecretDto, isSecret)
	if err != nil {
		impl.logger.Errorw("error occurred while extracting cm/secret json", "configSecretName", configSecretMap.Name, "err", err)
		return parallelStep, argoTemplate, err
	}
	parallelStep, argoTemplate = impl.createStepAndTemplate(isSecret, cmSecretIndex, cmSecretJson)
	return parallelStep, argoTemplate, nil
}

func (impl *ArgoWorkflowExecutorImpl) createStepAndTemplate(isSecret bool, cmSecretIndex int, cmSecretJson string) (argoWfApiV1.ParallelSteps, argoWfApiV1.Template) {
	stepName := fmt.Sprintf(STEP_NAME_REGEX, "cm", cmSecretIndex)
	templateName := fmt.Sprintf(TEMPLATE_NAME_REGEX, "cm", cmSecretIndex)
	if isSecret {
		stepName = fmt.Sprintf(STEP_NAME_REGEX, "secret", cmSecretIndex)
		templateName = fmt.Sprintf(TEMPLATE_NAME_REGEX, "secret", cmSecretIndex)
	}
	parallelStep := argoWfApiV1.ParallelSteps{
		Steps: []argoWfApiV1.WorkflowStep{
			{
				Name:     stepName,
				Template: templateName,
			},
		},
	}
	argoTemplate := argoWfApiV1.Template{
		Name: templateName,
		Resource: &argoWfApiV1.ResourceTemplate{
			Action:            RESOURCE_CREATE_ACTION,
			SetOwnerReference: true,
			Manifest:          string(cmSecretJson),
		},
	}
	return parallelStep, argoTemplate
}

func (impl *ArgoWorkflowExecutorImpl) getClientInstance(namespace string, clusterConfig *rest.Config) (argoWfClientV1.WorkflowInterface, error) {
	clientSet, err := versioned.NewForConfig(clusterConfig)
	if err != nil {
		impl.logger.Errorw("error occurred while creating client from config", "err", err)
		return nil, err
	}
	wfClient := clientSet.ArgoprojV1alpha1().Workflows(namespace) // create the workflow client
	return wfClient, nil
}
