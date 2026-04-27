// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package k8shelper

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (k *k8sHelper) WithSpire() *k8sHelper {
	hostPathType := corev1.HostPathDirectory
	k.volumes = append(k.volumes, corev1.Volume{
		Name: "spire-agent-socket",
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: "/run/spire/agent-sockets",
				Type: &hostPathType,
			},
		},
	})
	k.volumeMounts = append(k.volumeMounts, corev1.VolumeMount{
		Name:      "spire-agent-socket",
		MountPath: "/tmp/spire-agent/public",
		ReadOnly:  false,
	})
	k.serviceAccountName = k.name
	return k
}

func (k *k8sHelper) CreateServiceAccount() error {
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k.name,
			Namespace: k.namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "slim-client",
				"app.kubernetes.io/component": k.name,
			},
		},
	}

	_, err := k.clientset.CoreV1().ServiceAccounts(k.namespace).Create(
		context.TODO(),
		serviceAccount,
		metav1.CreateOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to create service account %s: %v", k.name, err)
	}

	return nil
}

func (k *k8sHelper) CleanupServiceAccount() error {
	err := k.clientset.CoreV1().ServiceAccounts(k.namespace).Delete(
		context.TODO(),
		k.name,
		metav1.DeleteOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to delete service account %s: %v", k.name, err)
	}

	return nil
}

func (k *k8sHelper) CreateClusterSPIFFEID() error {
	// Create ClusterSPIFFEID custom resource
	clusterSPIFFEID := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "spire.spiffe.io/v1alpha1",
			"kind":       "ClusterSPIFFEID",
			"metadata": map[string]interface{}{
				"name": k.name,
				"labels": map[string]string{
					"app.kubernetes.io/name":      "slim-client",
					"app.kubernetes.io/component": k.name,
				},
			},
			"spec": map[string]interface{}{
				"className": "spire-spire",
				"podSelector": map[string]interface{}{
					"matchExpressions": []interface{}{
						map[string]interface{}{
							"key":      "app.kubernetes.io/component",
							"operator": "In",
							"values":   []interface{}{k.name},
						},
					},
				},
				"workloadSelectorTemplates": []interface{}{
					fmt.Sprintf("k8s:sa:%s", k.name),
				},
				"spiffeIDTemplate": fmt.Sprintf("spiffe://{{ .TrustDomain }}/ns/%s/sa/%s", k.namespace, k.name),
			},
		},
	}

	// Define the GVR for ClusterSPIFFEID
	clusterSPIFFEIDGVR := schema.GroupVersionResource{
		Group:    "spire.spiffe.io",
		Version:  "v1alpha1",
		Resource: "clusterspiffeids",
	}

	_, err := k.dynamicClient.Resource(clusterSPIFFEIDGVR).Create(
		context.TODO(),
		clusterSPIFFEID,
		metav1.CreateOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to create ClusterSPIFFEID %s: %v", k.name, err)
	}

	return nil
}

func (k *k8sHelper) CleanupClusterSPIFFEID() error {

	// Define the GVR for ClusterSPIFFEID
	clusterSPIFFEIDGVR := schema.GroupVersionResource{
		Group:    "spire.spiffe.io",
		Version:  "v1alpha1",
		Resource: "clusterspiffeids",
	}

	err := k.dynamicClient.Resource(clusterSPIFFEIDGVR).Delete(
		context.TODO(),
		k.name,
		metav1.DeleteOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to delete ClusterSPIFFEID %s: %v", k.name, err)
	}

	return nil
}
