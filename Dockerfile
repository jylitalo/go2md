FROM golang:1.20 AS build-stage
WORKDIR /app
COPY . ./
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /go2md go2md.go

FROM gcr.io/distroless/base-debian11 AS build-release-stage
WORKDIR /
COPY --from=build-stage /go2md /go2md
ENTRYPOINT ["/go2md", "--recursive", "--ignore-main", "--output=README.md"]

