FROM docker.io/library/golang:1.25-bookworm AS go-build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
RUN CGO_ENABLED=0 go build -o /markovd ./cmd/markovd

FROM docker.io/library/node:22-bookworm-slim AS ui-build
WORKDIR /ui
COPY ui/package.json ui/package-lock.json ./
RUN npm ci
COPY ui/ .
RUN npm run build

FROM docker.io/library/debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=go-build /markovd /usr/local/bin/markovd
COPY bin/markov /usr/local/bin/markov
COPY --from=ui-build /ui/dist /srv/ui
ENV MARKOVD_MARKOV_BIN=/usr/local/bin/markov
EXPOSE 8080
ENTRYPOINT ["markovd"]
