// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package k8shelper

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (k *k8sHelper) CreatePod() (*corev1.Pod, error) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k.name,
			Namespace: k.namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  k.name,
					Image: k.imageName,
				},
			},
			RestartPolicy: corev1.RestartPolicyAlways,
		},
	}

	if k.envVars != nil {
		var envVars []corev1.EnvVar
		for k, v := range k.envVars {
			envVar := corev1.EnvVar{
				Name:  k,
				Value: v,
			}
			envVars = append(envVars, envVar)
		}
		pod.Spec.Containers[0].Env = envVars
	}

	if k.command != nil {
		pod.Spec.Containers[0].Command = k.command
	}

	if k.args != nil {
		pod.Spec.Containers[0].Args = k.args
	}

	if k.containerPorts != nil {
		var ports []corev1.ContainerPort
		for _, port := range k.containerPorts {
			containerPort := corev1.ContainerPort{
				ContainerPort: port,
			}
			ports = append(ports, containerPort)
		}
		pod.Spec.Containers[0].Ports = ports
		pod.ObjectMeta.Labels = map[string]string{
			"app": k.name,
		}
	}

	// Create the pod
	fmt.Println("Creating pod...")

	return k.clientset.CoreV1().Pods(k.namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
}

func (k *k8sHelper) CleanupPod(ctx context.Context) error {
	return k.clientset.CoreV1().Pods(k.namespace).Delete(ctx, k.name, metav1.DeleteOptions{})
}

func (k *k8sHelper) WaitForPodRunning(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	watch, err := k.clientset.CoreV1().Pods(k.namespace).Watch(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", k.name),
	})
	if err != nil {
		return err
	}
	defer watch.Stop()

	fmt.Println("Waiting for pod to running...")

	for event := range watch.ResultChan() {
		pod, ok := event.Object.(*corev1.Pod)
		if !ok {
			continue
		}

		fmt.Printf("Pod %s status: %s\n", pod.Name, pod.Status.Phase)

		if pod.Status.Phase == corev1.PodRunning {
			return nil
		} else if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodSucceeded {
			return fmt.Errorf("pod ran to completion")
		}
	}

	return fmt.Errorf("watch closed before pod became running")
}
