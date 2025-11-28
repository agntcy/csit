# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

spire:
  enabled: {{ .Spire.Enabled }}

slim:
  daemonset: {{ .DeployAsDaemonSet }}
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
        node_id: ${env:SLIM_SVC_ID}
        group_name: "{{ .ClusterName }}"      
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
