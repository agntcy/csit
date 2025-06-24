// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package k8shelper

import (
	corev1 "k8s.io/api/core/v1"
)

func (k *k8sHelper) WithCertSecret() *k8sHelper {
	return k.WithVolumes(
		[]corev1.Volume{
			{
				Name: "certs",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: "mtls-client-tls",
					},
				},
			},
		},
	).WithWithVolumeMounts([]corev1.VolumeMount{
		{
			Name:      "certs",
			MountPath: "/etc/certs",
		},
	})
}
