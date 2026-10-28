# API en Go: un solo build y dos imágenes finales.
#   docker build --target local  -t api-go .   -> servidor HTTP (docker compose)
#   docker build --target lambda -t api-go .   -> AWS Lambda (runtime provided.al2023)

FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go vet ./... && go test ./...
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -tags lambda.norpc -o /out/api ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot AS local
COPY --from=build /out/api /api
EXPOSE 8080
ENTRYPOINT ["/api"]

FROM public.ecr.aws/lambda/provided:al2023 AS lambda
COPY --from=build /out/api ./api
ENTRYPOINT ["./api"]
