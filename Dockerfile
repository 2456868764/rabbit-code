# syntax=docker/dockerfile:1
# Build context: rabbit-code/ (module root, same directory as go.mod).
FROM golang:1.22-bookworm AS build
WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .
ARG VERSION=0.0.0-docker
ARG COMMIT=unknown
RUN CGO_ENABLED=0 go build -trimpath \
	-ldflags "-s -w -X github.com/2456868764/rabbit-code/internal/version.Version=${VERSION} -X github.com/2456868764/rabbit-code/internal/version.Commit=${COMMIT}" \
	-o /out/rabbit-code ./cmd/rabbit-code

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/rabbit-code /rabbit-code
ENTRYPOINT ["/rabbit-code"]
