// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v6.31.1
// source: sdk/helper/pluginutil/multiplexing.proto

package pluginutil

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type MultiplexingSupportRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *MultiplexingSupportRequest) Reset() {
	*x = MultiplexingSupportRequest{}
	mi := &file_sdk_helper_pluginutil_multiplexing_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *MultiplexingSupportRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MultiplexingSupportRequest) ProtoMessage() {}

func (x *MultiplexingSupportRequest) ProtoReflect() protoreflect.Message {
	mi := &file_sdk_helper_pluginutil_multiplexing_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MultiplexingSupportRequest.ProtoReflect.Descriptor instead.
func (*MultiplexingSupportRequest) Descriptor() ([]byte, []int) {
	return file_sdk_helper_pluginutil_multiplexing_proto_rawDescGZIP(), []int{0}
}

type MultiplexingSupportResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Supported     bool                   `protobuf:"varint,1,opt,name=supported,proto3" json:"supported,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *MultiplexingSupportResponse) Reset() {
	*x = MultiplexingSupportResponse{}
	mi := &file_sdk_helper_pluginutil_multiplexing_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *MultiplexingSupportResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MultiplexingSupportResponse) ProtoMessage() {}

func (x *MultiplexingSupportResponse) ProtoReflect() protoreflect.Message {
	mi := &file_sdk_helper_pluginutil_multiplexing_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MultiplexingSupportResponse.ProtoReflect.Descriptor instead.
func (*MultiplexingSupportResponse) Descriptor() ([]byte, []int) {
	return file_sdk_helper_pluginutil_multiplexing_proto_rawDescGZIP(), []int{1}
}

func (x *MultiplexingSupportResponse) GetSupported() bool {
	if x != nil {
		return x.Supported
	}
	return false
}

var File_sdk_helper_pluginutil_multiplexing_proto protoreflect.FileDescriptor

const file_sdk_helper_pluginutil_multiplexing_proto_rawDesc = "" +
	"\n" +
	"(sdk/helper/pluginutil/multiplexing.proto\x12\x17pluginutil.multiplexing\"\x1c\n" +
	"\x1aMultiplexingSupportRequest\";\n" +
	"\x1bMultiplexingSupportResponse\x12\x1c\n" +
	"\tsupported\x18\x01 \x01(\bR\tsupported2\x97\x01\n" +
	"\x12PluginMultiplexing\x12\x80\x01\n" +
	"\x13MultiplexingSupport\x123.pluginutil.multiplexing.MultiplexingSupportRequest\x1a4.pluginutil.multiplexing.MultiplexingSupportResponseB5Z3github.com/openbao/openbao/sdk/v2/helper/pluginutilb\x06proto3"

var (
	file_sdk_helper_pluginutil_multiplexing_proto_rawDescOnce sync.Once
	file_sdk_helper_pluginutil_multiplexing_proto_rawDescData []byte
)

func file_sdk_helper_pluginutil_multiplexing_proto_rawDescGZIP() []byte {
	file_sdk_helper_pluginutil_multiplexing_proto_rawDescOnce.Do(func() {
		file_sdk_helper_pluginutil_multiplexing_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_sdk_helper_pluginutil_multiplexing_proto_rawDesc), len(file_sdk_helper_pluginutil_multiplexing_proto_rawDesc)))
	})
	return file_sdk_helper_pluginutil_multiplexing_proto_rawDescData
}

var file_sdk_helper_pluginutil_multiplexing_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_sdk_helper_pluginutil_multiplexing_proto_goTypes = []any{
	(*MultiplexingSupportRequest)(nil),  // 0: pluginutil.multiplexing.MultiplexingSupportRequest
	(*MultiplexingSupportResponse)(nil), // 1: pluginutil.multiplexing.MultiplexingSupportResponse
}
var file_sdk_helper_pluginutil_multiplexing_proto_depIdxs = []int32{
	0, // 0: pluginutil.multiplexing.PluginMultiplexing.MultiplexingSupport:input_type -> pluginutil.multiplexing.MultiplexingSupportRequest
	1, // 1: pluginutil.multiplexing.PluginMultiplexing.MultiplexingSupport:output_type -> pluginutil.multiplexing.MultiplexingSupportResponse
	1, // [1:2] is the sub-list for method output_type
	0, // [0:1] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_sdk_helper_pluginutil_multiplexing_proto_init() }
func file_sdk_helper_pluginutil_multiplexing_proto_init() {
	if File_sdk_helper_pluginutil_multiplexing_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_sdk_helper_pluginutil_multiplexing_proto_rawDesc), len(file_sdk_helper_pluginutil_multiplexing_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_sdk_helper_pluginutil_multiplexing_proto_goTypes,
		DependencyIndexes: file_sdk_helper_pluginutil_multiplexing_proto_depIdxs,
		MessageInfos:      file_sdk_helper_pluginutil_multiplexing_proto_msgTypes,
	}.Build()
	File_sdk_helper_pluginutil_multiplexing_proto = out.File
	file_sdk_helper_pluginutil_multiplexing_proto_goTypes = nil
	file_sdk_helper_pluginutil_multiplexing_proto_depIdxs = nil
}
