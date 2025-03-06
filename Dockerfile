# use official docker golang image
FROM golang:1.21 AS builder

# set up workdir
WORKDIR /app

# copy go modules and app sources
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# compile app inside the container
RUN CGO_ENABLED=0 GOOS=linux go build -o app

# create minimal image
FROM alpine:latest

# install required packages
RUN apk --no-cache add ca-certificates redis

# copy compiled app
COPY --from=builder /app/app /app/app

# set up workdir
WORKDIR /app

# expose port, default = 8200
EXPOSE 8200

# run app
CMD ["./app"]
