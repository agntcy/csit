{
  "endpoint": "http://{{ .SlimHost }}:{{ .SlimPort }}",
  "auth": {
    "basic": {
      "username": "testuser",
      "password": "secret123"
    }
  },
  "buffer_size": 1024,
  "compression": "Gzip",
  "connect_timeout": "10s",
  "headers": {
    "x-custom-header": "value"
  },
  "keepalive": {
    "http2_keepalive": "2h",
    "keep_alive_while_idle": false,
    "tcp_keepalive": "20s",
    "timeout": "20s"
  },
  "origin": "https://client.example.com",
  "rate_limit": "20/60",                    
  {{- if .Spire.Enabled }}
  "tls": {
    "cert_file": "/svids/tls.crt",
    "key_file": "/svids/tls.key",
    "ca_file": "/svids/svid_bundle.pem"
  },                    
  {{- end }}                    
  "request_timeout": "30s"
}