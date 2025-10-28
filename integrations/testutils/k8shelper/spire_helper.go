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

func (k *k8sHelper) WithSpireHelper() *k8sHelper {
	return k.WithVolumes(
		[]corev1.Volume{
			{
				Name: "spire-agent-socket",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/run/spire/agent-sockets",
						Type: func() *corev1.HostPathType {
							t := corev1.HostPathDirectory
							return &t
						}(),
					},
				},
			},
			{
				Name: "svids-volume",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			{
				Name: "config-volume",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: k.name,
						},
					},
				},
			},
		},
	).WithWithVolumeMounts([]corev1.VolumeMount{
		{
			Name:      "svids-volume",
			MountPath: "/svids",
		},
	}).WithInitContainer(
		corev1.Container{
			Name:  "spiffe-helper",
			Image: "ghcr.io/spiffe/spiffe-helper:0.10.0",
			Args:  []string{"-config", "config/helper.conf"},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "config-volume",
					MountPath: "/config/helper.conf",
					SubPath:   "helper.conf",
				},
				{
					Name:      "spire-agent-socket",
					MountPath: "/run/spire/agent-sockets",
					ReadOnly:  false,
				},
				{
					Name:      "svids-volume",
					MountPath: "/svids",
					ReadOnly:  false,
				},
			},
		}).WithServiceAccountName(k.name)

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
