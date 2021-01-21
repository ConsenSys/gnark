// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package pb

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

// Groth16Client is the client API for Groth16 service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type Groth16Client interface {
	// Prove takes circuitID and witness as parameter
	// this is a synchronous call and bypasses the job queue
	// it is meant to be used for small circuits, for larger circuits (proving time) and witnesses,
	// use CreateProveJob instead
	Prove(ctx context.Context, in *ProveRequest, opts ...grpc.CallOption) (*ProveResult, error)
	// CreateProveJob enqueue a job into the job queue with WAITING_WITNESS status
	CreateProveJob(ctx context.Context, in *CreateProveJobRequest, opts ...grpc.CallOption) (*CreateProveJobResponse, error)
	// SubscribeToProveJob enables a client to get job status changes from the server
	// at connection start, server sends current job status
	// when job is done (ok or errored), server closes connection
	SubscribeToProveJob(ctx context.Context, in *SubscribeToProveJobRequest, opts ...grpc.CallOption) (Groth16_SubscribeToProveJobClient, error)
}

type groth16Client struct {
	cc grpc.ClientConnInterface
}

func NewGroth16Client(cc grpc.ClientConnInterface) Groth16Client {
	return &groth16Client{cc}
}

func (c *groth16Client) Prove(ctx context.Context, in *ProveRequest, opts ...grpc.CallOption) (*ProveResult, error) {
	out := new(ProveResult)
	err := c.cc.Invoke(ctx, "/gnarkd.Groth16/Prove", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *groth16Client) CreateProveJob(ctx context.Context, in *CreateProveJobRequest, opts ...grpc.CallOption) (*CreateProveJobResponse, error) {
	out := new(CreateProveJobResponse)
	err := c.cc.Invoke(ctx, "/gnarkd.Groth16/CreateProveJob", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *groth16Client) SubscribeToProveJob(ctx context.Context, in *SubscribeToProveJobRequest, opts ...grpc.CallOption) (Groth16_SubscribeToProveJobClient, error) {
	stream, err := c.cc.NewStream(ctx, &Groth16_ServiceDesc.Streams[0], "/gnarkd.Groth16/SubscribeToProveJob", opts...)
	if err != nil {
		return nil, err
	}
	x := &groth16SubscribeToProveJobClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Groth16_SubscribeToProveJobClient interface {
	Recv() (*ProveJobResult, error)
	grpc.ClientStream
}

type groth16SubscribeToProveJobClient struct {
	grpc.ClientStream
}

func (x *groth16SubscribeToProveJobClient) Recv() (*ProveJobResult, error) {
	m := new(ProveJobResult)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// Groth16Server is the server API for Groth16 service.
// All implementations must embed UnimplementedGroth16Server
// for forward compatibility
type Groth16Server interface {
	// Prove takes circuitID and witness as parameter
	// this is a synchronous call and bypasses the job queue
	// it is meant to be used for small circuits, for larger circuits (proving time) and witnesses,
	// use CreateProveJob instead
	Prove(context.Context, *ProveRequest) (*ProveResult, error)
	// CreateProveJob enqueue a job into the job queue with WAITING_WITNESS status
	CreateProveJob(context.Context, *CreateProveJobRequest) (*CreateProveJobResponse, error)
	// SubscribeToProveJob enables a client to get job status changes from the server
	// at connection start, server sends current job status
	// when job is done (ok or errored), server closes connection
	SubscribeToProveJob(*SubscribeToProveJobRequest, Groth16_SubscribeToProveJobServer) error
	mustEmbedUnimplementedGroth16Server()
}

// UnimplementedGroth16Server must be embedded to have forward compatible implementations.
type UnimplementedGroth16Server struct {
}

func (UnimplementedGroth16Server) Prove(context.Context, *ProveRequest) (*ProveResult, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Prove not implemented")
}
func (UnimplementedGroth16Server) CreateProveJob(context.Context, *CreateProveJobRequest) (*CreateProveJobResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateProveJob not implemented")
}
func (UnimplementedGroth16Server) SubscribeToProveJob(*SubscribeToProveJobRequest, Groth16_SubscribeToProveJobServer) error {
	return status.Errorf(codes.Unimplemented, "method SubscribeToProveJob not implemented")
}
func (UnimplementedGroth16Server) mustEmbedUnimplementedGroth16Server() {}

// UnsafeGroth16Server may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to Groth16Server will
// result in compilation errors.
type UnsafeGroth16Server interface {
	mustEmbedUnimplementedGroth16Server()
}

func RegisterGroth16Server(s grpc.ServiceRegistrar, srv Groth16Server) {
	s.RegisterService(&Groth16_ServiceDesc, srv)
}

func _Groth16_Prove_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ProveRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(Groth16Server).Prove(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/gnarkd.Groth16/Prove",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(Groth16Server).Prove(ctx, req.(*ProveRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Groth16_CreateProveJob_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateProveJobRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(Groth16Server).CreateProveJob(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/gnarkd.Groth16/CreateProveJob",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(Groth16Server).CreateProveJob(ctx, req.(*CreateProveJobRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Groth16_SubscribeToProveJob_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(SubscribeToProveJobRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(Groth16Server).SubscribeToProveJob(m, &groth16SubscribeToProveJobServer{stream})
}

type Groth16_SubscribeToProveJobServer interface {
	Send(*ProveJobResult) error
	grpc.ServerStream
}

type groth16SubscribeToProveJobServer struct {
	grpc.ServerStream
}

func (x *groth16SubscribeToProveJobServer) Send(m *ProveJobResult) error {
	return x.ServerStream.SendMsg(m)
}

// Groth16_ServiceDesc is the grpc.ServiceDesc for Groth16 service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Groth16_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "gnarkd.Groth16",
	HandlerType: (*Groth16Server)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Prove",
			Handler:    _Groth16_Prove_Handler,
		},
		{
			MethodName: "CreateProveJob",
			Handler:    _Groth16_CreateProveJob_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "SubscribeToProveJob",
			Handler:       _Groth16_SubscribeToProveJob_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "pb/gnarkd.proto",
}
