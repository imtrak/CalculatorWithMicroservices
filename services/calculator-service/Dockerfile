FROM golang:1.27 AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/calculator-service ./cmd/calculator-service

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/calculator-service /calculator-service
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/calculator-service"]
