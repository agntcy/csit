// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package k8shelper

import (
	"context"
	"fmt"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (k *k8sHelper) CreateJob() (*batchv1.Job, error) {
	var backOffLimit int32 = 2
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k.name,
			Namespace: k.namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  k.name,
							Image: k.imageName,
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
			BackoffLimit: &backOffLimit,
		},
	}

	var envVars []corev1.EnvVar
	if k.envVars != nil {
		for k, v := range k.envVars {
			envVar := corev1.EnvVar{
				Name:  k,
				Value: v,
			}
			envVars = append(envVars, envVar)
		}
		job.Spec.Template.Spec.Containers[0].Env = envVars
	}

	if k.command != nil {
		job.Spec.Template.Spec.Containers[0].Command = k.command
	}

	if k.args != nil {
		job.Spec.Template.Spec.Containers[0].Args = k.args
	}

	// Create the job
	fmt.Println("Creating job...")

	return k.clientset.BatchV1().Jobs(k.namespace).Create(context.TODO(), job, metav1.CreateOptions{})
}

func (k *k8sHelper) CleanupJob(ctx context.Context) error {
	deletePolicy := metav1.DeletePropagationBackground

	return k.clientset.BatchV1().Jobs(k.namespace).Delete(ctx, k.name, metav1.DeleteOptions{PropagationPolicy: &deletePolicy})
}

func (k *k8sHelper) WaitForJobCompletion(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	watch, err := k.clientset.BatchV1().Jobs(k.namespace).Watch(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", k.name),
	})
	if err != nil {
		return err
	}
	defer watch.Stop()

	fmt.Println("Waiting for job to complete...")

	for event := range watch.ResultChan() {
		job, ok := event.Object.(*batchv1.Job)
		if !ok {
			continue
		}

		fmt.Printf("Job %s status: Active=%d, Succeeded=%d, Failed=%d\n",
			job.Name, job.Status.Active, job.Status.Succeeded, job.Status.Failed)

		// Check job conditions for completion or failure
		for _, condition := range job.Status.Conditions {
			if condition.Type == batchv1.JobComplete && condition.Status == corev1.ConditionTrue {
				fmt.Printf("Job %s completed successfully\n", job.Name)
				return nil
			} else if condition.Type == batchv1.JobFailed && condition.Status == corev1.ConditionTrue {
				return fmt.Errorf("job failed: %s", condition.Reason)
			}
		}
	}

	return fmt.Errorf("watch closed before job completed")
}
