// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package k8shelper

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (k *k8sHelper) CreateService(name string) (*corev1.Service, error) {
	if k.containerPorts == nil {
		return nil, errors.New("unabele to create service because containerPort is not defined")
	}
	var ports []corev1.ServicePort
	for _, port := range k.containerPorts {
		servicePort := corev1.ServicePort{
			Protocol:   corev1.ProtocolTCP,
			Port:       port,
			TargetPort: intstr.FromInt(int(port)),
		}
		ports = append(ports, servicePort)
	}
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: k.namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": k.name,
			},
			Ports: ports,
			Type:  corev1.ServiceTypeClusterIP,
		},
	}

	// Create the secice
	fmt.Println("Creating service...")

	return k.clientset.CoreV1().Services(k.namespace).Create(context.TODO(), service, metav1.CreateOptions{})
}
