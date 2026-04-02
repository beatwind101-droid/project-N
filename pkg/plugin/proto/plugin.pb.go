// Package proto provides gRPC service definitions for plugin communication.
// This is a hand-written implementation based on plugin.proto.
// Generated code would be preferred for production use.
// To generate code: protoc --go_out=. plugin.proto
package proto

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Request/Response types

type MetadataRequest struct{}
type MetadataResponse struct {
	Name        string
	Version     string
	Description string
	Author      string
	Tags        []string
	Category    string
}

func (m *MetadataResponse) GetName() string        { return m.Name }
func (m *MetadataResponse) GetVersion() string     { return m.Version }
func (m *MetadataResponse) GetDescription() string { return m.Description }
func (m *MetadataResponse) GetAuthor() string      { return m.Author }
func (m *MetadataResponse) GetTags() []string      { return m.Tags }
func (m *MetadataResponse) GetCategory() string    { return m.Category }

type InitRequest struct {
	Config map[string]string
}

func (r *InitRequest) GetConfig() map[string]string { return r.Config }

type InitResponse struct {
	Success bool
	Error   string
}

func (r *InitResponse) GetSuccess() bool { return r.Success }
func (r *InitResponse) GetError() string { return r.Error }

type ExecuteRequest struct {
	Params map[string]string
}

func (r *ExecuteRequest) GetParams() map[string]string { return r.Params }

type ExecuteResponse struct {
	Success bool
	Data    string
	Error   string
}

func (r *ExecuteResponse) GetSuccess() bool { return r.Success }
func (r *ExecuteResponse) GetData() string  { return r.Data }
func (r *ExecuteResponse) GetError() string { return r.Error }

type ValidateRequest struct {
	Params map[string]string
}

func (r *ValidateRequest) GetParams() map[string]string { return r.Params }

type ValidateResponse struct {
	Valid bool
	Error string
}

func (r *ValidateResponse) GetValid() bool   { return r.Valid }
func (r *ValidateResponse) GetError() string { return r.Error }

type ShutdownRequest struct{}
type ShutdownResponse struct {
	Success bool
	Error   string
}

func (r *ShutdownResponse) GetSuccess() bool { return r.Success }
func (r *ShutdownResponse) GetError() string { return r.Error }

// ToolServiceServer is the server API for ToolService.
type ToolServiceServer interface {
	GetMetadata(context.Context, *MetadataRequest) (*MetadataResponse, error)
	Initialize(context.Context, *InitRequest) (*InitResponse, error)
	Execute(context.Context, *ExecuteRequest) (*ExecuteResponse, error)
	Validate(context.Context, *ValidateRequest) (*ValidateResponse, error)
	Shutdown(context.Context, *ShutdownRequest) (*ShutdownResponse, error)
}

// UnimplementedToolServiceServer can be embedded to have forward compatible implementations.
type UnimplementedToolServiceServer struct{}

func (UnimplementedToolServiceServer) GetMetadata(context.Context, *MetadataRequest) (*MetadataResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetMetadata not implemented")
}
func (UnimplementedToolServiceServer) Initialize(context.Context, *InitRequest) (*InitResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Initialize not implemented")
}
func (UnimplementedToolServiceServer) Execute(context.Context, *ExecuteRequest) (*ExecuteResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Execute not implemented")
}
func (UnimplementedToolServiceServer) Validate(context.Context, *ValidateRequest) (*ValidateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Validate not implemented")
}
func (UnimplementedToolServiceServer) Shutdown(context.Context, *ShutdownRequest) (*ShutdownResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Shutdown not implemented")
}

// RegisterToolServiceServer registers the ToolServiceServer to the grpc.Server.
func RegisterToolServiceServer(s *grpc.Server, srv ToolServiceServer) {
	s.RegisterService(&_ToolService_serviceDesc, srv)
}

func _ToolService_GetMetadata_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MetadataRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ToolServiceServer).GetMetadata(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/toolkit.plugin.ToolService/GetMetadata"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ToolServiceServer).GetMetadata(ctx, req.(*MetadataRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ToolService_Initialize_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(InitRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ToolServiceServer).Initialize(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/toolkit.plugin.ToolService/Initialize"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ToolServiceServer).Initialize(ctx, req.(*InitRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ToolService_Execute_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ExecuteRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ToolServiceServer).Execute(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/toolkit.plugin.ToolService/Execute"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ToolServiceServer).Execute(ctx, req.(*ExecuteRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ToolService_Validate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ValidateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ToolServiceServer).Validate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/toolkit.plugin.ToolService/Validate"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ToolServiceServer).Validate(ctx, req.(*ValidateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ToolService_Shutdown_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ShutdownRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ToolServiceServer).Shutdown(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/toolkit.plugin.ToolService/Shutdown"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ToolServiceServer).Shutdown(ctx, req.(*ShutdownRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _ToolService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "toolkit.plugin.ToolService",
	HandlerType: (*ToolServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "GetMetadata", Handler: _ToolService_GetMetadata_Handler},
		{MethodName: "Initialize", Handler: _ToolService_Initialize_Handler},
		{MethodName: "Execute", Handler: _ToolService_Execute_Handler},
		{MethodName: "Validate", Handler: _ToolService_Validate_Handler},
		{MethodName: "Shutdown", Handler: _ToolService_Shutdown_Handler},
	},
	Streams: []grpc.StreamDesc{},
}

// ToolServiceClient is the client API for ToolService.
type ToolServiceClient interface {
	GetMetadata(ctx context.Context, in *MetadataRequest, opts ...grpc.CallOption) (*MetadataResponse, error)
	Initialize(ctx context.Context, in *InitRequest, opts ...grpc.CallOption) (*InitResponse, error)
	Execute(ctx context.Context, in *ExecuteRequest, opts ...grpc.CallOption) (*ExecuteResponse, error)
	Validate(ctx context.Context, in *ValidateRequest, opts ...grpc.CallOption) (*ValidateResponse, error)
	Shutdown(ctx context.Context, in *ShutdownRequest, opts ...grpc.CallOption) (*ShutdownResponse, error)
}

type toolServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewToolServiceClient(cc grpc.ClientConnInterface) ToolServiceClient {
	return &toolServiceClient{cc}
}

func (c *toolServiceClient) GetMetadata(ctx context.Context, in *MetadataRequest, opts ...grpc.CallOption) (*MetadataResponse, error) {
	out := new(MetadataResponse)
	err := c.cc.Invoke(ctx, "/toolkit.plugin.ToolService/GetMetadata", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *toolServiceClient) Initialize(ctx context.Context, in *InitRequest, opts ...grpc.CallOption) (*InitResponse, error) {
	out := new(InitResponse)
	err := c.cc.Invoke(ctx, "/toolkit.plugin.ToolService/Initialize", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *toolServiceClient) Execute(ctx context.Context, in *ExecuteRequest, opts ...grpc.CallOption) (*ExecuteResponse, error) {
	out := new(ExecuteResponse)
	err := c.cc.Invoke(ctx, "/toolkit.plugin.ToolService/Execute", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *toolServiceClient) Validate(ctx context.Context, in *ValidateRequest, opts ...grpc.CallOption) (*ValidateResponse, error) {
	out := new(ValidateResponse)
	err := c.cc.Invoke(ctx, "/toolkit.plugin.ToolService/Validate", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *toolServiceClient) Shutdown(ctx context.Context, in *ShutdownRequest, opts ...grpc.CallOption) (*ShutdownResponse, error) {
	out := new(ShutdownResponse)
	err := c.cc.Invoke(ctx, "/toolkit.plugin.ToolService/Shutdown", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Ensure UnimplementedToolServiceServer implements ToolServiceServer
var _ ToolServiceServer = (*UnimplementedToolServiceServer)(nil)

// Suppress unused import
var _ = fmt.Sprintf