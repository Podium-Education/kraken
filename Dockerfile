FROM golang as builder

WORKDIR /src

COPY . .

RUN go vet ./... && \
    go test ./... && \
    make

FROM gcr.io/distroless/static AS final

USER nonroot:nonroot

WORKDIR /app

COPY --from=builder --chown=nonroot:nonroot /src/build/kraken .

ENTRYPOINT ["/app/kraken"]