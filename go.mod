module github.com/grafana/xk6-loki

go 1.26.4

require (
	github.com/brianvoe/gofakeit/v6 v6.9.0
	github.com/gogo/protobuf v1.3.2
	github.com/golang/snappy v1.0.0
	github.com/grafana/loki/pkg/push v0.0.0-20260611205623-ac76b402773c
	github.com/grafana/loki/v3 v3.0.0-20260611205623-ac76b402773c
	github.com/grafana/sobek v0.0.0-20260908083152-4698bc773ae7
	github.com/mailru/easyjson v0.7.7
	github.com/prometheus/common v0.70.1
	github.com/sirupsen/logrus v1.9.4
	go.k6.io/k6/v2 v2.3.0
)

require (
	github.com/Azure/go-ntlmssp v0.1.1 // indirect
	github.com/Soontao/goHttpDigestClient v0.0.0-20170320082612-6d28bb1415c5 // indirect
	github.com/andybalholm/brotli v1.2.3 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dlclark/regexp2/v2 v2.7.2 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/evanw/esbuild v0.28.2 // indirect
	github.com/fatih/color v1.19.0 // indirect
	github.com/go-kit/log v0.2.1 // indirect
	github.com/go-logfmt/logfmt v0.6.1 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-sourcemap/sourcemap v2.1.4+incompatible // indirect
	github.com/google/pprof v0.0.0-20260302011040-a15ffb7f9dcc // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grafana/sobek-webapi-encoding v0.1.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.30.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/klauspost/compress v1.20.0 // indirect
	github.com/mattn/go-colorable v0.1.15 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/mstoykov/atlas v0.0.0-20220811071828-388f114305dd // indirect
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/serenize/snaker v0.0.0-20201027110005-a7ad2135616e // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/stretchr/testify v1.12.1 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.46.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.46.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.46.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.46.0 // indirect
	go.opentelemetry.io/otel/metric v1.46.0 // indirect
	go.opentelemetry.io/otel/sdk v1.46.0 // indirect
	go.opentelemetry.io/otel/trace v1.46.0 // indirect
	go.opentelemetry.io/proto/otlp v1.11.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/crypto v0.56.0 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260819154853-08b0e4226688 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260819154853-08b0e4226688 // indirect
	google.golang.org/grpc v1.84.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
	gopkg.in/guregu/null.v3 v3.5.0 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
)

// Use fork of gocql that has gokit logs and Prometheus metrics.
replace github.com/gocql/gocql => github.com/grafana/gocql v0.0.0-20200605141915-ba5dc39ece85

replace github.com/Azure/go-ntlmssp => github.com/Azure/go-ntlmssp v0.1.1

exclude k8s.io/client-go v8.0.0+incompatible
