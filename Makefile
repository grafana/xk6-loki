PWD := $(shell pwd)
GO_FILES := $(shell find . -type f -name "*.go" -print)

k6: $(GO_FILES)
	xk6 build \
		--replace "github.com/mingrammer/flog=github.com/chaudum/flog@v0.4.4-0.20211115125504-92153be038e6" \
	  --with "github.com/grafana/xk6-loki=$(PWD)"

go.sum: $(GO_FILES) go.mod
	go mod tidy

run:
	$(PWD)/k6 run test.js -v
