module github.com/grafana/xk6-loki

go 1.16

require (
	github.com/brianvoe/gofakeit/v6 v6.9.0
	github.com/dop251/goja v0.0.0-20220405120441-9037c2b61cbf
	github.com/gogo/protobuf v1.3.1
	github.com/golang/snappy v0.0.3
	github.com/grafana/loki v1.6.1
	github.com/json-iterator/go v1.1.10
	github.com/mingrammer/flog v0.4.3
	github.com/pkg/errors v0.9.1
	github.com/prometheus/common v0.10.0
	github.com/sirupsen/logrus v1.8.1
	go.k6.io/k6 v0.38.0
)

replace github.com/mingrammer/flog => github.com/chaudum/flog v0.4.4-0.20211115125504-92153be038e6

replace google.golang.org/grpc => google.golang.org/grpc v1.40.0
