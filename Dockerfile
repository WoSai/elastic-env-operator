# Build the manager binary
FROM golang:1.13 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
COPY domain/ domain/
COPY webhook/ webhook/

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
ENV  GOPROXY="https://goproxy.cn,direct" GO111MODULE=on CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN  go mod download && go build -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM debian:stretch
WORKDIR /
COPY --from=builder /workspace/manager .

ENTRYPOINT ["/manager"]
