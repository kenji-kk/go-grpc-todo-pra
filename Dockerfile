FROM golang:latest
 
WORKDIR /app/api
 
ADD ./api /app/api
 
RUN go get -u google.golang.org/grpc \
    && go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.26\
    && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1\
    && go get go.mongodb.org/mongo-driver/mongo\
    && go mod tidy
 