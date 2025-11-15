FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY src/ ./src/
RUN go build -o main ./src/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
COPY --from=builder /app/src/static ./src/static
COPY --from=builder /app/src/docs ./src/docs

EXPOSE 3001
ENV DATABASE_URL="postgres://postgres:Hyg57aff@host.docker.internal:5432/owesome?sslmode=disable"
ENV JWT_SECRET="dkfslæfksdlæfkdlæsdkfcm,vxc.xcmv,.sdfmsd,.fmsd,.fm,sd.dfmsd.,aæda'ødæasøødæas'ødasæddaksdkaslædask"

CMD ["./main"]