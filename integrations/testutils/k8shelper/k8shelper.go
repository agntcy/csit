// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package k8shelper

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type SecretVolume struct {
	SecretName string
	VolumeName string
	Path       string
}

type k8sHelper struct {
	clientset          kubernetes.Interface
	dynamicClient      dynamic.Interface
	name               string
	namespace          string
	imageName          string
	envVars            map[string]string
	command            []string
	args               []string
	containerPorts     []int32
	secretVolumes      []SecretVolume
	serviceAccountName string

	volumes       []corev1.Volume
	volumeMounts  []corev1.VolumeMount
	initContainer corev1.Container
}

func NewK8sHelper(name, namespace, imageName string, c kubernetes.Interface, dynamicClient dynamic.Interface) *k8sHelper {
	return &k8sHelper{
		clientset:     c,
		dynamicClient: dynamicClient,
		name:          name,
		namespace:     namespace,
		imageName:     imageName,
	}
}

func (k *k8sHelper) WithEnvVars(envVars map[string]string) *k8sHelper {
	k.envVars = envVars

	return k
}

func (k *k8sHelper) WithCommand(command []string) *k8sHelper {
	k.command = command

	return k
}

func (k *k8sHelper) WithServiceAccountName(name string) *k8sHelper {
	k.serviceAccountName = name

	return k
}

func (k *k8sHelper) WithArgs(args []string) *k8sHelper {
	k.args = args

	return k
}

func (k *k8sHelper) WithContainerPorts(ports []int32) *k8sHelper {
	k.containerPorts = ports

	return k
}

func (k *k8sHelper) WithSecretVolumes(secretVolumes []SecretVolume) *k8sHelper {
	k.secretVolumes = secretVolumes

	return k
}

func (k *k8sHelper) WithInitContainer(container corev1.Container) *k8sHelper {
	k.initContainer = container

	return k
}

func (k *k8sHelper) WithWithVolumeMounts(volumeMounts []corev1.VolumeMount) *k8sHelper {
	k.volumeMounts = volumeMounts

	return k
}

func (k *k8sHelper) WithVolumes(volumes []corev1.Volume) *k8sHelper {
	k.volumes = volumes

	return k
}

func CreateK8sClientSet() (*kubernetes.Clientset, error) {
	kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("unable to load kubeconfig %w", err)
	}
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "unable to load kubeconfig")

	return kubernetes.NewForConfig(config)
}

func CreateDynamicK8sClient() (*dynamic.DynamicClient, error) {
	kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("unable to load kubeconfig %w", err)
	}
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "unable to load kubeconfig")

	// Create dynamic client for custom resources
	return dynamic.NewForConfig(config)
}
