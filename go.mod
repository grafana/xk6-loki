module github.com/grafana/xk6-loki

go 1.16

require (
	github.com/brianvoe/gofakeit/v6 v6.9.0
	github.com/dop251/goja v0.0.0-20220815083517-0c74f9139fd6
	github.com/gogo/protobuf v1.3.2
	github.com/golang/snappy v0.0.4
	github.com/grafana/loki v1.6.2-0.20221006221238-7979cfbe076d
	github.com/mailru/easyjson v0.7.7
	github.com/mingrammer/flog v0.4.3
	github.com/pkg/errors v0.9.1
	github.com/prometheus/common v0.37.0
	github.com/sirupsen/logrus v1.8.1
	go.k6.io/k6 v0.40.0
)

replace github.com/mingrammer/flog => github.com/chaudum/flog v0.4.4-0.20220419113107-eb2f67f18b99

replace google.golang.org/grpc => google.golang.org/grpc v1.45.0

replace github.com/hashicorp/consul => github.com/hashicorp/consul v1.5.1

// Fork containing a line-buffered logger which should improve logging performance.
// TODO: submit PR to upstream and remove this
replace github.com/go-kit/log => github.com/dannykopping/go-kit-log v0.2.2-0.20221002180827-5591c1641b6b
