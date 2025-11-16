package kim

//go:generate protoc --go_out=./idl/session --go-grpc_out=./idl/session ./idl/session/session.proto
//go:generate protoc --go_out=./idl/gateway --go-grpc_out=./idl/gateway ./idl/gateway/gateway.proto
//go:generate protoc --go_out=./idl/push --go-grpc_out=./idl/push ./idl/push/push.proto
//go:generate protoc --go_out=./idl/message --go-grpc_out=./idl/message ./idl/message/message.proto
