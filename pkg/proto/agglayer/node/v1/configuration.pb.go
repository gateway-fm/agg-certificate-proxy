// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v5.29.3
// source: agglayer/node/v1/configuration.proto

package v1

import (
	v1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/node/types/v1"
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

// The kind of error that occurred and that are reported by the configuration
// service.
type GetEpochConfigurationErrorKind int32

const (
	// Unspecified error.
	GetEpochConfigurationErrorKind_GET_EPOCH_CONFIGURATION_ERROR_KIND_UNSPECIFIED GetEpochConfigurationErrorKind = 0
	// The AggLayer isn't configured with a BlockClock configuration, thus no
	// EpochConfiguration is available.
	GetEpochConfigurationErrorKind_GET_EPOCH_CONFIGURATION_ERROR_KIND_UNEXPECTED_CLOCK_CONFIGURATION GetEpochConfigurationErrorKind = 1
)

// Enum value maps for GetEpochConfigurationErrorKind.
var (
	GetEpochConfigurationErrorKind_name = map[int32]string{
		0: "GET_EPOCH_CONFIGURATION_ERROR_KIND_UNSPECIFIED",
		1: "GET_EPOCH_CONFIGURATION_ERROR_KIND_UNEXPECTED_CLOCK_CONFIGURATION",
	}
	GetEpochConfigurationErrorKind_value = map[string]int32{
		"GET_EPOCH_CONFIGURATION_ERROR_KIND_UNSPECIFIED":                    0,
		"GET_EPOCH_CONFIGURATION_ERROR_KIND_UNEXPECTED_CLOCK_CONFIGURATION": 1,
	}
)

func (x GetEpochConfigurationErrorKind) Enum() *GetEpochConfigurationErrorKind {
	p := new(GetEpochConfigurationErrorKind)
	*p = x
	return p
}

func (x GetEpochConfigurationErrorKind) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (GetEpochConfigurationErrorKind) Descriptor() protoreflect.EnumDescriptor {
	return file_agglayer_node_v1_configuration_proto_enumTypes[0].Descriptor()
}

func (GetEpochConfigurationErrorKind) Type() protoreflect.EnumType {
	return &file_agglayer_node_v1_configuration_proto_enumTypes[0]
}

func (x GetEpochConfigurationErrorKind) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use GetEpochConfigurationErrorKind.Descriptor instead.
func (GetEpochConfigurationErrorKind) EnumDescriptor() ([]byte, []int) {
	return file_agglayer_node_v1_configuration_proto_rawDescGZIP(), []int{0}
}

// Request to get the current epoch configuration.
type GetEpochConfigurationRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetEpochConfigurationRequest) Reset() {
	*x = GetEpochConfigurationRequest{}
	mi := &file_agglayer_node_v1_configuration_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetEpochConfigurationRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetEpochConfigurationRequest) ProtoMessage() {}

func (x *GetEpochConfigurationRequest) ProtoReflect() protoreflect.Message {
	mi := &file_agglayer_node_v1_configuration_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetEpochConfigurationRequest.ProtoReflect.Descriptor instead.
func (*GetEpochConfigurationRequest) Descriptor() ([]byte, []int) {
	return file_agglayer_node_v1_configuration_proto_rawDescGZIP(), []int{0}
}

// Response to the current epoch configuration request.
type GetEpochConfigurationResponse struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// The epoch configuration.
	EpochConfiguration *v1.EpochConfiguration `protobuf:"bytes,1,opt,name=epoch_configuration,json=epochConfiguration,proto3" json:"epoch_configuration,omitempty"`
	unknownFields      protoimpl.UnknownFields
	sizeCache          protoimpl.SizeCache
}

func (x *GetEpochConfigurationResponse) Reset() {
	*x = GetEpochConfigurationResponse{}
	mi := &file_agglayer_node_v1_configuration_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetEpochConfigurationResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetEpochConfigurationResponse) ProtoMessage() {}

func (x *GetEpochConfigurationResponse) ProtoReflect() protoreflect.Message {
	mi := &file_agglayer_node_v1_configuration_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetEpochConfigurationResponse.ProtoReflect.Descriptor instead.
func (*GetEpochConfigurationResponse) Descriptor() ([]byte, []int) {
	return file_agglayer_node_v1_configuration_proto_rawDescGZIP(), []int{1}
}

func (x *GetEpochConfigurationResponse) GetEpochConfiguration() *v1.EpochConfiguration {
	if x != nil {
		return x.EpochConfiguration
	}
	return nil
}

var File_agglayer_node_v1_configuration_proto protoreflect.FileDescriptor

const file_agglayer_node_v1_configuration_proto_rawDesc = "" +
	"\n" +
	"$agglayer/node/v1/configuration.proto\x12\x10agglayer.node.v1\x1a0agglayer/node/types/v1/epoch_configuration.proto\"\x1e\n" +
	"\x1cGetEpochConfigurationRequest\"|\n" +
	"\x1dGetEpochConfigurationResponse\x12[\n" +
	"\x13epoch_configuration\x18\x01 \x01(\v2*.agglayer.node.types.v1.EpochConfigurationR\x12epochConfiguration*\x9b\x01\n" +
	"\x1eGetEpochConfigurationErrorKind\x122\n" +
	".GET_EPOCH_CONFIGURATION_ERROR_KIND_UNSPECIFIED\x10\x00\x12E\n" +
	"AGET_EPOCH_CONFIGURATION_ERROR_KIND_UNEXPECTED_CLOCK_CONFIGURATION\x10\x012\x90\x01\n" +
	"\x14ConfigurationService\x12x\n" +
	"\x15GetEpochConfiguration\x12..agglayer.node.v1.GetEpochConfigurationRequest\x1a/.agglayer.node.v1.GetEpochConfigurationResponseb\x06proto3"

var (
	file_agglayer_node_v1_configuration_proto_rawDescOnce sync.Once
	file_agglayer_node_v1_configuration_proto_rawDescData []byte
)

func file_agglayer_node_v1_configuration_proto_rawDescGZIP() []byte {
	file_agglayer_node_v1_configuration_proto_rawDescOnce.Do(func() {
		file_agglayer_node_v1_configuration_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_agglayer_node_v1_configuration_proto_rawDesc), len(file_agglayer_node_v1_configuration_proto_rawDesc)))
	})
	return file_agglayer_node_v1_configuration_proto_rawDescData
}

var file_agglayer_node_v1_configuration_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_agglayer_node_v1_configuration_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_agglayer_node_v1_configuration_proto_goTypes = []any{
	(GetEpochConfigurationErrorKind)(0),   // 0: agglayer.node.v1.GetEpochConfigurationErrorKind
	(*GetEpochConfigurationRequest)(nil),  // 1: agglayer.node.v1.GetEpochConfigurationRequest
	(*GetEpochConfigurationResponse)(nil), // 2: agglayer.node.v1.GetEpochConfigurationResponse
	(*v1.EpochConfiguration)(nil),         // 3: agglayer.node.types.v1.EpochConfiguration
}
var file_agglayer_node_v1_configuration_proto_depIdxs = []int32{
	3, // 0: agglayer.node.v1.GetEpochConfigurationResponse.epoch_configuration:type_name -> agglayer.node.types.v1.EpochConfiguration
	1, // 1: agglayer.node.v1.ConfigurationService.GetEpochConfiguration:input_type -> agglayer.node.v1.GetEpochConfigurationRequest
	2, // 2: agglayer.node.v1.ConfigurationService.GetEpochConfiguration:output_type -> agglayer.node.v1.GetEpochConfigurationResponse
	2, // [2:3] is the sub-list for method output_type
	1, // [1:2] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_agglayer_node_v1_configuration_proto_init() }
func file_agglayer_node_v1_configuration_proto_init() {
	if File_agglayer_node_v1_configuration_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_agglayer_node_v1_configuration_proto_rawDesc), len(file_agglayer_node_v1_configuration_proto_rawDesc)),
			NumEnums:      1,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_agglayer_node_v1_configuration_proto_goTypes,
		DependencyIndexes: file_agglayer_node_v1_configuration_proto_depIdxs,
		EnumInfos:         file_agglayer_node_v1_configuration_proto_enumTypes,
		MessageInfos:      file_agglayer_node_v1_configuration_proto_msgTypes,
	}.Build()
	File_agglayer_node_v1_configuration_proto = out.File
	file_agglayer_node_v1_configuration_proto_goTypes = nil
	file_agglayer_node_v1_configuration_proto_depIdxs = nil
}
