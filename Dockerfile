# Build UI
FROM node:20.10.0-alpine3.18 as UI-BUILDER
WORKDIR /ui
COPY /ui/package.json /ui/package-lock.json ./
RUN npm ci
COPY /ui .
RUN npm run build

# Build go
FROM golang:1.21.5-alpine3.19 as BUILDER
RUN apk add --no-cache git
WORKDIR /workspace
COPY . .
RUN rm -rf /ui/dist/edge-ui
COPY --from=UI-BUILDER /ui/dist/edge-ui ./ui/dist/edge-ui
RUN go mod tidy
ARG CGO_ENABLED=0
ARG GOOS=linux
ARG GOARCH=amd64
RUN CGO_ENABLED=${CGO_ENABLED} GOOS=${GOOS} GOARCH=${GOARCH} \
  go build -ldflags='-w -s -extldflags "-static"' -a -o edge .

# Copy binary into final (alpine) image
FROM alpine:3.19
RUN adduser -D -h /workspace 1000
USER 1000
WORKDIR /workspace
COPY --from=BUILDER /workspace/edge .
EXPOSE 8080
ENTRYPOINT ["/workspace/edge"]
CMD ["server"]
