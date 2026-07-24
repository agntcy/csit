# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

spire:
  enabled: {{ .Spire.Enabled }}

slim:
  deploymentMode: {{ if .DeployAsDaemonSet }}"DaemonSet"{{ else }}"Deployment"{{ end }}
  replicaCount: {{ .ReplicaCount }}
  overrideConfig:
    tracing:
      log_level: debug
      display_thread_names: true
      display_thread_ids: true

    runtime:
      n_cores: 0
      thread_name: "slim-data-plane"
      drain_timeout: 10s

    services:
      slim/0:
        node_id: ${env:SLIM_NODE_ID}
        group_name: "{{ .ClusterName }}"      
        # Intra-cluster peer discovery: every replica in this cluster watches the
        # cluster Service's EndpointSlices and forms a full mesh of peer links, so
        # subscriptions propagate across all nodes (1 hop). This makes a client
        # reachable regardless of which node it connects to via the Service and
        # regardless of which node holds the control-plane inter-cluster link.
        # NOTE: the port must be set explicitly here because the chart only
        # auto-injects it on the non-overrideConfig path.
        peers:
          deployment_name: "{{ .ClusterName }}"
          topology: full_mesh
          discovery:
            type: kubernetes
            namespace: "{{ .ClusterName }}"
            service_name: "agntcy-{{ .ClusterName }}-slim"
            port: {{ .SlimPort }}
        dataplane:
          servers:
          - endpoint: "0.0.0.0:{{ .SlimPort }}"
            metadata:
              local_endpoint: ${env:MY_POD_IP}
              external_endpoint: "{{ .ServiceName }}:{{ .SlimPort }}"    
              trust_domain: "example.org" 
            tls:
    {{- if .Spire.Enabled }}
              source:
                type: spire
                socket_path: unix:/tmp/spire-agent/public/api.sock
                target_spiffe_id: spiffe://example.local/ns/slim/sa/slim                  
              ca_source:
                type: spire
                socket_path: unix:/tmp/spire-agent/public/api.sock
    {{- else }}
              insecure: true
    {{- end }}
        controller:
          clients:
            - endpoint: "{{ .SlimControllerEndpoint }}"
              tls:
    {{- if .Spire.Enabled }}
                source:
                  type: spire
                  socket_path: unix:/tmp/spire-agent/public/api.sock               
                ca_source:
                  type: spire
                  socket_path: unix:/tmp/spire-agent/public/api.sock
                  trust_domains:
                    - example.org
    {{- else }}
                insecure: true
    {{- end }}
