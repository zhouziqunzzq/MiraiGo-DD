// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package predictor

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// ChatPredictorClient is the client API for ChatPredictor service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ChatPredictorClient interface {
	// get predictions for one chat msg
	PredictOne(ctx context.Context, in *PredictRequest, opts ...grpc.CallOption) (*PredictReply, error)
}

type chatPredictorClient struct {
	cc grpc.ClientConnInterface
}

func NewChatPredictorClient(cc grpc.ClientConnInterface) ChatPredictorClient {
	return &chatPredictorClient{cc}
}

func (c *chatPredictorClient) PredictOne(ctx context.Context, in *PredictRequest, opts ...grpc.CallOption) (*PredictReply, error) {
	out := new(PredictReply)
	err := c.cc.Invoke(ctx, "/ChatPredictor/PredictOne", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ChatPredictorServer is the server API for ChatPredictor service.
// All implementations must embed UnimplementedChatPredictorServer
// for forward compatibility
type ChatPredictorServer interface {
	// get predictions for one chat msg
	PredictOne(context.Context, *PredictRequest) (*PredictReply, error)
	mustEmbedUnimplementedChatPredictorServer()
}

// UnimplementedChatPredictorServer must be embedded to have forward compatible implementations.
type UnimplementedChatPredictorServer struct {
}

func (UnimplementedChatPredictorServer) PredictOne(context.Context, *PredictRequest) (*PredictReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PredictOne not implemented")
}
func (UnimplementedChatPredictorServer) mustEmbedUnimplementedChatPredictorServer() {}

// UnsafeChatPredictorServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ChatPredictorServer will
// result in compilation errors.
type UnsafeChatPredictorServer interface {
	mustEmbedUnimplementedChatPredictorServer()
}

func RegisterChatPredictorServer(s grpc.ServiceRegistrar, srv ChatPredictorServer) {
	s.RegisterService(&ChatPredictor_ServiceDesc, srv)
}

func _ChatPredictor_PredictOne_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PredictRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ChatPredictorServer).PredictOne(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/ChatPredictor/PredictOne",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ChatPredictorServer).PredictOne(ctx, req.(*PredictRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// ChatPredictor_ServiceDesc is the grpc.ServiceDesc for ChatPredictor service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ChatPredictor_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "ChatPredictor",
	HandlerType: (*ChatPredictorServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "PredictOne",
			Handler:    _ChatPredictor_PredictOne_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "protos/predictor.proto",
}