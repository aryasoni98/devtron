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

package pipeline

import (
	"context"
	blob_storage "github.com/devtron-labs/common-lib/blob-storage"
	"github.com/devtron-labs/common-lib/utils/k8s"
	"github.com/devtron-labs/devtron/pkg/build/pipeline/bean"
	"github.com/devtron-labs/devtron/pkg/pipeline/types"
	"go.uber.org/zap"
	"io"
	"k8s.io/client-go/kubernetes"
	"os"
	"path/filepath"
)

type CiLogService interface {
	FetchRunningWorkflowLogs(ciLogRequest types.BuildLogRequest, clusterConfig *k8s.ClusterConfig, isExt bool, followLogs bool) (io.ReadCloser, func() error, error)
	FetchLogs(baseLogLocationPathConfig string, ciLogRequest types.BuildLogRequest) (*os.File, func() error, error)
}

type CiLogServiceImpl struct {
	logger     *zap.SugaredLogger
	kubeClient *kubernetes.Clientset
	k8sUtil    *k8s.K8sServiceImpl
}

func NewCiLogServiceImpl(logger *zap.SugaredLogger, k8sUtil *k8s.K8sServiceImpl) (*CiLogServiceImpl, error) {
	_, _, clientSet, err := k8sUtil.GetK8sInClusterConfigAndClients()
	if err != nil {
		logger.Errorw("error in getting k8s in cluster client set", "err", err)
		return nil, err
	}
	return &CiLogServiceImpl{
		logger:     logger,
		kubeClient: clientSet,
		k8sUtil:    k8sUtil,
	}, nil
}

func (impl *CiLogServiceImpl) FetchRunningWorkflowLogs(ciLogRequest types.BuildLogRequest, clusterConfig *k8s.ClusterConfig, isExt bool, followLogs bool) (io.ReadCloser, func() error, error) {
	var kubeClient *kubernetes.Clientset
	kubeClient = impl.kubeClient
	var err error
	if isExt {
		_, _, kubeClient, err = impl.k8sUtil.GetK8sConfigAndClients(clusterConfig)
		if err != nil {
			impl.logger.Errorw("error in getting kubeClient by cluster config", "err", err, "workFlowId", ciLogRequest.WorkflowId)
			return nil, nil, err
		}
	}
	req := impl.k8sUtil.GetLogsForAPod(kubeClient, ciLogRequest.Namespace, ciLogRequest.PodName, bean.Main, followLogs)
	podLogs, err := req.Stream(context.Background())
	if err != nil {
		impl.logger.Errorw("error in opening stream", "name", ciLogRequest.PodName, "err", err)
		return nil, nil, err
	} else if podLogs == nil {
		impl.logger.Warnw("no stream reader found", "name", ciLogRequest.PodName)
		return nil, func() error { return nil }, err
	}
	cleanUpFunc := func() error {
		impl.logger.Info("closing running pod log stream")
		err = podLogs.Close()
		if err != nil {
			impl.logger.Errorw("err", "err", err)
		}
		return err
	}
	return podLogs, cleanUpFunc, nil
}

func (impl *CiLogServiceImpl) FetchLogs(baseLogLocationPathConfig string, logRequest types.BuildLogRequest) (*os.File, func() error, error) {
	tempFile := baseLogLocationPathConfig
	tempFile = filepath.Clean(filepath.Join(tempFile, logRequest.PodName+".log"))

	blobStorageService := blob_storage.NewBlobStorageServiceImpl(impl.logger)
	request := &blob_storage.BlobStorageRequest{
		StorageType:         logRequest.CloudProvider,
		SourceKey:           logRequest.LogsFilePath,
		DestinationKey:      tempFile,
		AzureBlobBaseConfig: logRequest.AzureBlobConfig,
		AwsS3BaseConfig:     logRequest.AwsS3BaseConfig,
		GcpBlobBaseConfig:   logRequest.GcpBlobBaseConfig,
	}

	_, _, err := blobStorageService.Get(request)
	if err != nil {
		impl.logger.Errorw("err occurred while downloading logs file", "request", request, "err", err)
		return nil, nil, err
	}

	file, err := os.Open(tempFile)
	if err != nil {
		impl.logger.Errorw("err", "err", err)
		return nil, nil, err
	}

	cleanUpFunc := func() error {
		impl.logger.Info("cleaning up log files")
		fErr := file.Close()
		if fErr != nil {
			impl.logger.Errorw("err", "err", fErr)
			return fErr
		}
		fErr = os.Remove(tempFile)
		if fErr != nil {
			impl.logger.Errorw("err", "err", fErr)
			return fErr
		}
		return fErr
	}

	if err != nil {
		impl.logger.Errorw("err", "err", err)
		_ = cleanUpFunc()
		return nil, nil, err
	}
	return file, cleanUpFunc, nil
}
