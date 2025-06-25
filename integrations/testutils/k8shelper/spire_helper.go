// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package k8shelper

import (
	corev1 "k8s.io/api/core/v1"
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
			// Use inline func to get address of the constant
			RestartPolicy: func() *corev1.ContainerRestartPolicy {
				v := corev1.ContainerRestartPolicyAlways
				return &v
			}(),
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
		})

}
