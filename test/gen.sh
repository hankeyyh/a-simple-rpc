#!/bin/sh
# check if argument is provided
if [ "$#" -ne 1 ]; then
  echo "Usage: $0 <proto_filename>"
  exit 1
fi

# get proto file
PROTO_FILE=$1

go build -o protoc-gen-simple-rpc ../gen/main.go

protoc -I proto \
--go_out=pb --go_opt=paths=source_relative \
--simple-rpc_out=pb --simple-rpc_opt=paths=source_relative \
--plugin=./protoc-gen-simple-rpc \
"$PROTO_FILE"

rm ./protoc-gen-simple-rpc