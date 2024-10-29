
FROM quay.io/zncdatadev/go-devel:1.23.2-kubedoop0.0.0-dev AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download

COPY cmd/main.go cmd/main.go
COPY internal/controller/ internal/controller/

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager cmd/main.go

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
