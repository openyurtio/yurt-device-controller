FROM --platform=${BUILDPLATFORM} golang:1.18 as builder
ADD . /build
ARG TARGETOS TARGETARCH
WORKDIR /build/
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} make build

FROM --platform=${TARGETPLATFORM} alpine:3.17
ARG TARGETOS TARGETARCH
WORKDIR /
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
    apk add ca-certificates bash libc6-compat iptables ip6tables && update-ca-certificates && rm /var/cache/apk/*
COPY --from=builder /build/_output/bin/${TARGETOS}/${TARGETARCH}/yurt-device-controller /yurt-device-controller
ENTRYPOINT ["/yurt-device-controller"]