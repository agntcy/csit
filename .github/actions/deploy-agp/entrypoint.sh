#!/bin/sh -l

# Get input parameters
GATEWAY_IMAGE_TAG=$1
GATEWAY_CHART_TAG=$2
KIND_CLUSTER_NAME=$3
KIND_CLUSTER_NAMESPACE=$4

# Define the path to the kubeconfig in the mounted workspace
KUBECONFIG="/github/workspace/.kube/config"
kubectl get pods -A

echo "Deploying with gateway image tag: $GATEWAY_IMAGE_TAG"
echo "Using chart version: $GATEWAY_CHART_TAG"
echo "To cluster: $KIND_CLUSTER_NAME"
echo "In namespace: $KIND_CLUSTER_NAMESPACE"

ls
which task

task test-env:deploy