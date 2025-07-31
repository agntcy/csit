# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

spire:
  enabled: {{ .Spire.Enabled }}

serviceAccount:
    name:

slim:
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
      slim/{{.ServiceName }}:
        pubsub:
          servers:
          - endpoint: "0.0.0.0:{{ .SlimPort }}"
            tls:
    {{- if .Spire.Enabled }}
              cert_file: "/svids/tls.crt"
              key_file: "/svids/tls.key"
              ca_file: "/svids/svid_bundle.pem"
    {{- else }}
              insecure: true
        {{- end }}
        controller:
          clients:
            - endpoint: "{{ .SlimControllerEndpoint }}"
              tls:
                insecure: true
