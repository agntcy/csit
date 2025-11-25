// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package k8shelper

import (
	"context"
	"fmt"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (k *k8sHelper) CreateConfigMap(fileName, content string) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k.name,
			Namespace: k.namespace,
		},
		Data: map[string]string{
			fileName: content,
		},
	}

	return k.clientset.CoreV1().ConfigMaps(k.namespace).Create(context.TODO(), configMap, metav1.CreateOptions{})
}

func (k *k8sHelper) CreateConfigMapFromFile(fileName, filePath string) (*corev1.ConfigMap, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return k.CreateConfigMap(fileName, string(content))
}

func (k *k8sHelper) CleanupConfigMap(ctx context.Context) error {
	return k.clientset.CoreV1().ConfigMaps(k.namespace).Delete(ctx, k.name, metav1.DeleteOptions{})
}

func (k *k8sHelper) CreatePod() (*corev1.Pod, error) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k.name,
			Namespace: k.namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "slim-client",
				"app.kubernetes.io/component": k.name,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  k.name,
					Image: k.imageName,
				},
			},
			RestartPolicy: corev1.RestartPolicyOnFailure,
		},
	}

	if k.serviceAccountName != "" {
		pod.Spec.ServiceAccountName = k.serviceAccountName
	}

	if k.secretVolumes != nil {
		var volumes []corev1.Volume
		var volumeMounts []corev1.VolumeMount
		for _, secretVolume := range k.secretVolumes {
			volume := corev1.Volume{
				Name: secretVolume.VolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: secretVolume.SecretName,
					},
				},
			}
			volumes = append(volumes, volume)

			volumeMount := corev1.VolumeMount{
				Name:      secretVolume.VolumeName,
				MountPath: secretVolume.Path,
			}
			volumeMounts = append(volumeMounts, volumeMount)
		}
		pod.Spec.Volumes = volumes
		pod.Spec.Containers[0].VolumeMounts = volumeMounts
	}

	if k.volumeMounts != nil {
		pod.Spec.Containers[0].VolumeMounts = append(pod.Spec.Containers[0].VolumeMounts, k.volumeMounts...)
	}

	if k.initContainer.Name != "" {
		pod.Spec.InitContainers = append(pod.Spec.InitContainers, k.initContainer)
	}

	if k.volumes != nil {
		pod.Spec.Volumes = append(pod.Spec.Volumes, k.volumes...)
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

	fmt.Println("Waiting for pod to be running and all containers ready...")

	for event := range watch.ResultChan() {
		pod, ok := event.Object.(*corev1.Pod)
		if !ok {
			continue
		}

		fmt.Printf("Pod %s status: %s\n", pod.Name, pod.Status.Phase)

		if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodSucceeded {
			return fmt.Errorf("pod ran to completion with phase: %s", pod.Status.Phase)
		}

		if pod.Status.Phase == corev1.PodRunning {
			// Check if all containers are ready
			allReady := true
			for _, containerStatus := range pod.Status.ContainerStatuses {
				fmt.Printf("Container %s ready: %t\n", containerStatus.Name, containerStatus.Ready)
				if !containerStatus.Ready {
					allReady = false
					break
				}
			}

			// Also check init containers if they exist
			for _, initContainerStatus := range pod.Status.InitContainerStatuses {
				fmt.Printf("Init container %s ready: %t\n", initContainerStatus.Name, initContainerStatus.Ready)
				if !initContainerStatus.Ready {
					allReady = false
					break
				}
			}

			if allReady {
				fmt.Printf("Pod %s is running and all containers are ready\n", pod.Name)
				return nil
			}
		}
	}

	return fmt.Errorf("watch closed before pod became running with all containers ready")
}
