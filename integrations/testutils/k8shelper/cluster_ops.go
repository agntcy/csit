// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package k8shelper

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// ListPodsByLabel returns the pods in a namespace matching a label selector.
func ListPodsByLabel(ctx context.Context, clientset kubernetes.Interface, namespace, labelSelector string) ([]corev1.Pod, error) {
	list, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods in %s with selector %q: %w", namespace, labelSelector, err)
	}
	return list.Items, nil
}

// DeletePodByName deletes a pod by name. A Deployment/DaemonSet owner will
// recreate it. node_id in the control plane equals the pod name, so this can be
// used to restart a specific SLIM node.
func DeletePodByName(ctx context.Context, clientset kubernetes.Interface, namespace, name string) error {
	if err := clientset.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("failed to delete pod %s/%s: %w", namespace, name, err)
	}
	return nil
}

// RestartDeployment triggers a rollout restart of a Deployment (equivalent to
// `kubectl rollout restart deployment/<name>`).
func RestartDeployment(ctx context.Context, clientset kubernetes.Interface, namespace, name string) error {
	patch := fmt.Sprintf(
		`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":%q}}}}}`,
		time.Now().Format(time.RFC3339Nano),
	)
	_, err := clientset.AppsV1().Deployments(namespace).Patch(
		ctx, name, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to restart deployment %s/%s: %w", namespace, name, err)
	}
	return nil
}

// WaitForDeploymentAvailable waits until a Deployment has finished rolling out
// and all replicas are available.
func WaitForDeploymentAvailable(ctx context.Context, clientset kubernetes.Interface, namespace, name string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		d, err := clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			lastErr = err
			time.Sleep(3 * time.Second)
			continue
		}
		desired := int32(1)
		if d.Spec.Replicas != nil {
			desired = *d.Spec.Replicas
		}
		if d.Status.ObservedGeneration >= d.Generation &&
			d.Status.UpdatedReplicas == desired &&
			d.Status.AvailableReplicas == desired &&
			d.Status.UnavailableReplicas == 0 {
			return nil
		}
		lastErr = fmt.Errorf("deployment %s/%s not ready: updated=%d available=%d unavailable=%d desired=%d",
			namespace, name, d.Status.UpdatedReplicas, d.Status.AvailableReplicas, d.Status.UnavailableReplicas, desired)
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("timed out waiting for deployment %s/%s: %w", namespace, name, lastErr)
}
