// Code generated by protoc-gen-go. DO NOT EDIT.
// source: hello.proto

/*
Package hello is a generated protocol buffer package.

It is generated from these files:
	hello.proto

It has these top-level messages:
	HiMessage
	Empty
*/
package hello

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// The request message containing the user's name.
type HiMessage struct {
	Value string `protobuf:"bytes,1,opt,name=value" json:"value,omitempty"`
}

func (m *HiMessage) Reset()                    { *m = HiMessage{} }
func (m *HiMessage) String() string            { return proto.CompactTextString(m) }
func (*HiMessage) ProtoMessage()               {}
func (*HiMessage) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *HiMessage) GetValue() string {
	if m != nil {
		return m.Value
	}
	return ""
}

type Empty struct {
}

func (m *Empty) Reset()                    { *m = Empty{} }
func (m *Empty) String() string            { return proto.CompactTextString(m) }
func (*Empty) ProtoMessage()               {}
func (*Empty) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func init() {
	proto.RegisterType((*HiMessage)(nil), "HiMessage")
	proto.RegisterType((*Empty)(nil), "Empty")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// Client API for Greeter service

type GreeterClient interface {
	SayHi(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*HiMessage, error)
}

type greeterClient struct {
	cc *grpc.ClientConn
}

func NewGreeterClient(cc *grpc.ClientConn) GreeterClient {
	return &greeterClient{cc}
}

func (c *greeterClient) SayHi(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*HiMessage, error) {
	out := new(HiMessage)
	err := grpc.Invoke(ctx, "/Greeter/SayHi", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for Greeter service

type GreeterServer interface {
	SayHi(context.Context, *Empty) (*HiMessage, error)
}

func RegisterGreeterServer(s *grpc.Server, srv GreeterServer) {
	s.RegisterService(&_Greeter_serviceDesc, srv)
}

func _Greeter_SayHi_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GreeterServer).SayHi(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Greeter/SayHi",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GreeterServer).SayHi(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

var _Greeter_serviceDesc = grpc.ServiceDesc{
	ServiceName: "Greeter",
	HandlerType: (*GreeterServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "SayHi",
			Handler:    _Greeter_SayHi_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "hello.proto",
}

func init() { proto.RegisterFile("hello.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 114 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0xe2, 0xe2, 0xce, 0x48, 0xcd, 0xc9,
	0xc9, 0xd7, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x57, 0x52, 0xe4, 0xe2, 0xf4, 0xc8, 0xf4, 0x4d, 0x2d,
	0x2e, 0x4e, 0x4c, 0x4f, 0x15, 0x12, 0xe1, 0x62, 0x2d, 0x4b, 0xcc, 0x29, 0x4d, 0x95, 0x60, 0x54,
	0x60, 0xd4, 0xe0, 0x0c, 0x82, 0x70, 0x94, 0xd8, 0xb9, 0x58, 0x5d, 0x73, 0x0b, 0x4a, 0x2a, 0x8d,
	0x34, 0xb8, 0xd8, 0xdd, 0x8b, 0x52, 0x53, 0x4b, 0x52, 0x8b, 0x84, 0x64, 0xb9, 0x58, 0x83, 0x13,
	0x2b, 0x3d, 0x32, 0x85, 0xd8, 0xf4, 0xc0, 0x72, 0x52, 0x5c, 0x7a, 0x70, 0x63, 0x94, 0x18, 0x92,
	0xd8, 0xc0, 0x86, 0x1b, 0x03, 0x02, 0x00, 0x00, 0xff, 0xff, 0xbd, 0xb4, 0x74, 0xc9, 0x6b, 0x00,
	0x00, 0x00,
}
