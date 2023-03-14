PWD := $(shell pwd)
GO_FILES := $(shell find . -type f -name "*.go" -print)

.PHONY: run

k6: $(GO_FILES)
	xk6 build master \
		--replace "google.golang.org/grpc=google.golang.org/grpc@v1.45.0" \
		--replace "github.com/hashicorp/consul=github.com/hashicorp/consul@v1.5.1" \
		--replace "github.com/gocql/gocql=github.com/grafana/gocql@v0.0.0-20200605141915-ba5dc39ece85" \
		--replace "github.com/prometheus/prometheus=github.com/prometheus/prometheus@v0.42.0" \
		--replace "github.com/grafana/loki=github.com/grafana/loki-hackathon-2023-03-project-lili@401b0faefc0a9437abb1e1964afa5a071207e76a" \
		--replace "github.com/grafana/loki/pkg/push=github.com/grafana/loki-hackathon-2023-03-project-lili/pkg/push@401b0faefc0a9437abb1e1964afa5a071207e76a" \
    --with "github.com/grafana/xk6-loki=$(PWD)"

go.sum: $(GO_FILES) go.mod
	go mod tidy

run:
	$(PWD)/k6 run examples/read-write-scenario.js
