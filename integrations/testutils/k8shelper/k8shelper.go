// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package k8shelper

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type k8sHelper struct {
	clientset      kubernetes.Interface
	name           string
	namespace      string
	imageName      string
	envVars        map[string]string
	command        []string
	args           []string
	containerPorts []int32
}

func NewK8sHelper(name, namespace, imageName string, c kubernetes.Interface) *k8sHelper {
	return &k8sHelper{
		clientset: c,
		name:      name,
		namespace: namespace,
		imageName: imageName,
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

func (k *k8sHelper) WithArgs(args []string) *k8sHelper {
	k.args = args

	return k
}

func (k *k8sHelper) WithContainerPorts(ports []int32) *k8sHelper {
	k.containerPorts = ports

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
