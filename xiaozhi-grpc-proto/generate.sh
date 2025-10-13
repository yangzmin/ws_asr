#!/bin/bash

cd proto
protoc --go_out=../generated/go/ai_service --go_opt=paths=source_relative \
       --go-grpc_out=../generated/go/ai_service --go-grpc_opt=paths=source_relative \
       ai_service.proto

echo "gRPC代码生成完成"

