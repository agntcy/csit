#!/bin/sh -l

# Get input parameters
GATEWAY_IMAGE_TAG=$1
GATEWAY_CHART_TAG=$2
KIND_CLUSTER_NAME=$3
KIND_CLUSTER_NAMESPACE=$4

echo "Deploying with gateway image tag: $GATEWAY_IMAGE_TAG"
echo "Using chart version: $GATEWAY_CHART_TAG"
echo "To cluster: $KIND_CLUSTER_NAME"
echo "In namespace: $KIND_CLUSTER_NAMESPACE"

task test-env:deploy