# Сначала собираем исполняемый файл
FROM golang:latest AS compiling_stage
WORKDIR /go/src/task_26A.3.1
COPY main.go .
COPY go.mod .
COPY go.sum .
RUN go mod download
RUN go build -o /go/bin/pipeline

# Конечный образ
FROM alpine:latest
LABEL version="1.0.0"
LABEL maintainer="Denis Maksimov"
COPY --from=compiling_stage /go/bin/pipeline /usr/local/bin/pipeline
CMD ["pipeline"]