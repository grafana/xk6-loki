PWD := $(shell pwd)
GO_FILES := $(shell find . -type f -name "*.go" -print)

.PHONY: run

k6: $(GO_FILES)
	xk6 build \
		--replace "github.com/mingrammer/flog=github.com/chaudum/flog@v0.4.4-0.20220419113107-eb2f67f18b99" \
		--replace "google.golang.org/grpc=google.golang.org/grpc@v1.45.0" \
	  --with "github.com/grafana/xk6-loki=$(PWD)"

go.sum: $(GO_FILES) go.mod
	go mod tidy

run:
	$(PWD)/k6 run examples/read-write-scenario.js
