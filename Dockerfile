# Build the manager binary
FROM golang:1.17 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
COPY utils/ utils/
COPY templates/ templates/
COPY readinessProbe/ readinessProbe/
COPY cmd/awsDataGather/ awsDataGather/
COPY pkg/ pkg/

RUN mkdir bin

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o manager main.go

# Build readiness probe binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o readinessServer readinessProbe/main.go

# Build aws data gathering binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o bin/awsDataGather awsDataGather/main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
COPY --from=builder /workspace/readinessServer .
COPY --from=builder /workspace/bin/ /bin/
COPY --from=builder /workspace/templates/customernotification.html /templates/
USER nonroot:nonroot

ENTRYPOINT ["/manager"]
