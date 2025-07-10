# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

spire:
  enabled: {{ .Spire.Enabled }}

slim:
  config:
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
        pubsub:
          servers:
          - endpoint: "{{ .SlimEndpoint }}"
            tls:
              cert_file: "/svids/tls.crt"
              key_file: "/svids/tls.key"
              ca_file: "/svids/svid_bundle.pem"
            controller:
        controller:
          server:
            endpoint: "{{ .SlimControllerEndpoint }}"
            tls:
              insecure: true

