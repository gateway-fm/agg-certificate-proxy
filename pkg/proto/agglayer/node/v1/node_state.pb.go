// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v5.29.3
// source: agglayer/node/v1/node_state.proto

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

// Error kind for GetCertificateHeader RPC.
type GetCertificateHeaderErrorKind int32

const (
	// Unspecified error.
	GetCertificateHeaderErrorKind_GET_CERTIFICATE_HEADER_ERROR_KIND_UNSPECIFIED GetCertificateHeaderErrorKind = 0
	// Missing field.
	GetCertificateHeaderErrorKind_GET_CERTIFICATE_HEADER_ERROR_KIND_MISSING_FIELD GetCertificateHeaderErrorKind = 1
	// Invalid data.
	GetCertificateHeaderErrorKind_GET_CERTIFICATE_HEADER_ERROR_KIND_INVALID_DATA GetCertificateHeaderErrorKind = 2
	// Certificate not found.
	GetCertificateHeaderErrorKind_GET_CERTIFICATE_HEADER_ERROR_KIND_NOT_FOUND GetCertificateHeaderErrorKind = 3
)

// Enum value maps for GetCertificateHeaderErrorKind.
var (
	GetCertificateHeaderErrorKind_name = map[int32]string{
		0: "GET_CERTIFICATE_HEADER_ERROR_KIND_UNSPECIFIED",
		1: "GET_CERTIFICATE_HEADER_ERROR_KIND_MISSING_FIELD",
		2: "GET_CERTIFICATE_HEADER_ERROR_KIND_INVALID_DATA",
		3: "GET_CERTIFICATE_HEADER_ERROR_KIND_NOT_FOUND",
	}
	GetCertificateHeaderErrorKind_value = map[string]int32{
		"GET_CERTIFICATE_HEADER_ERROR_KIND_UNSPECIFIED":   0,
		"GET_CERTIFICATE_HEADER_ERROR_KIND_MISSING_FIELD": 1,
		"GET_CERTIFICATE_HEADER_ERROR_KIND_INVALID_DATA":  2,
		"GET_CERTIFICATE_HEADER_ERROR_KIND_NOT_FOUND":     3,
	}
)

func (x GetCertificateHeaderErrorKind) Enum() *GetCertificateHeaderErrorKind {
	p := new(GetCertificateHeaderErrorKind)
	*p = x
	return p
}

func (x GetCertificateHeaderErrorKind) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (GetCertificateHeaderErrorKind) Descriptor() protoreflect.EnumDescriptor {
	return file_agglayer_node_v1_node_state_proto_enumTypes[0].Descriptor()
}

func (GetCertificateHeaderErrorKind) Type() protoreflect.EnumType {
	return &file_agglayer_node_v1_node_state_proto_enumTypes[0]
}

func (x GetCertificateHeaderErrorKind) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use GetCertificateHeaderErrorKind.Descriptor instead.
func (GetCertificateHeaderErrorKind) EnumDescriptor() ([]byte, []int) {
	return file_agglayer_node_v1_node_state_proto_rawDescGZIP(), []int{0}
}

// Error kind for GetLatestCertificateHeader RPC.
type GetLatestCertificateHeaderErrorKind int32

const (
	// Unspecified error.
	GetLatestCertificateHeaderErrorKind_GET_LATEST_CERTIFICATE_HEADER_ERROR_KIND_UNSPECIFIED GetLatestCertificateHeaderErrorKind = 0
	// Missing field.
	GetLatestCertificateHeaderErrorKind_GET_LATEST_CERTIFICATE_HEADER_ERROR_KIND_MISSING_FIELD GetLatestCertificateHeaderErrorKind = 1
	// Invalid data.
	GetLatestCertificateHeaderErrorKind_GET_LATEST_CERTIFICATE_HEADER_ERROR_KIND_INVALID_DATA GetLatestCertificateHeaderErrorKind = 2
	// Certificate not found.
	GetLatestCertificateHeaderErrorKind_GET_LATEST_CERTIFICATE_HEADER_ERROR_KIND_NOT_FOUND GetLatestCertificateHeaderErrorKind = 3
)

// Enum value maps for GetLatestCertificateHeaderErrorKind.
var (
	GetLatestCertificateHeaderErrorKind_name = map[int32]string{
		0: "GET_LATEST_CERTIFICATE_HEADER_ERROR_KIND_UNSPECIFIED",
		1: "GET_LATEST_CERTIFICATE_HEADER_ERROR_KIND_MISSING_FIELD",
		2: "GET_LATEST_CERTIFICATE_HEADER_ERROR_KIND_INVALID_DATA",
		3: "GET_LATEST_CERTIFICATE_HEADER_ERROR_KIND_NOT_FOUND",
	}
	GetLatestCertificateHeaderErrorKind_value = map[string]int32{
		"GET_LATEST_CERTIFICATE_HEADER_ERROR_KIND_UNSPECIFIED":   0,
		"GET_LATEST_CERTIFICATE_HEADER_ERROR_KIND_MISSING_FIELD": 1,
		"GET_LATEST_CERTIFICATE_HEADER_ERROR_KIND_INVALID_DATA":  2,
		"GET_LATEST_CERTIFICATE_HEADER_ERROR_KIND_NOT_FOUND":     3,
	}
)

func (x GetLatestCertificateHeaderErrorKind) Enum() *GetLatestCertificateHeaderErrorKind {
	p := new(GetLatestCertificateHeaderErrorKind)
	*p = x
	return p
}

func (x GetLatestCertificateHeaderErrorKind) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (GetLatestCertificateHeaderErrorKind) Descriptor() protoreflect.EnumDescriptor {
	return file_agglayer_node_v1_node_state_proto_enumTypes[1].Descriptor()
}

func (GetLatestCertificateHeaderErrorKind) Type() protoreflect.EnumType {
	return &file_agglayer_node_v1_node_state_proto_enumTypes[1]
}

func (x GetLatestCertificateHeaderErrorKind) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use GetLatestCertificateHeaderErrorKind.Descriptor instead.
func (GetLatestCertificateHeaderErrorKind) EnumDescriptor() ([]byte, []int) {
	return file_agglayer_node_v1_node_state_proto_rawDescGZIP(), []int{1}
}

// The type of latest certificate we want to get.
type LatestCertificateRequestType int32

const (
	// Default value
	LatestCertificateRequestType_LATEST_CERTIFICATE_REQUEST_TYPE_UNSPECIFIED LatestCertificateRequestType = 0
	// Pending certificate.
	LatestCertificateRequestType_LATEST_CERTIFICATE_REQUEST_TYPE_PENDING LatestCertificateRequestType = 1
	// Settled certificate.
	LatestCertificateRequestType_LATEST_CERTIFICATE_REQUEST_TYPE_SETTLED LatestCertificateRequestType = 2
)

// Enum value maps for LatestCertificateRequestType.
var (
	LatestCertificateRequestType_name = map[int32]string{
		0: "LATEST_CERTIFICATE_REQUEST_TYPE_UNSPECIFIED",
		1: "LATEST_CERTIFICATE_REQUEST_TYPE_PENDING",
		2: "LATEST_CERTIFICATE_REQUEST_TYPE_SETTLED",
	}
	LatestCertificateRequestType_value = map[string]int32{
		"LATEST_CERTIFICATE_REQUEST_TYPE_UNSPECIFIED": 0,
		"LATEST_CERTIFICATE_REQUEST_TYPE_PENDING":     1,
		"LATEST_CERTIFICATE_REQUEST_TYPE_SETTLED":     2,
	}
)

func (x LatestCertificateRequestType) Enum() *LatestCertificateRequestType {
	p := new(LatestCertificateRequestType)
	*p = x
	return p
}

func (x LatestCertificateRequestType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (LatestCertificateRequestType) Descriptor() protoreflect.EnumDescriptor {
	return file_agglayer_node_v1_node_state_proto_enumTypes[2].Descriptor()
}

func (LatestCertificateRequestType) Type() protoreflect.EnumType {
	return &file_agglayer_node_v1_node_state_proto_enumTypes[2]
}

func (x LatestCertificateRequestType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use LatestCertificateRequestType.Descriptor instead.
func (LatestCertificateRequestType) EnumDescriptor() ([]byte, []int) {
	return file_agglayer_node_v1_node_state_proto_rawDescGZIP(), []int{2}
}

// Request to get a CertificateHeader for a particular CertificateId.
type GetCertificateHeaderRequest struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// The certificate identifier.
	CertificateId *v1.CertificateId `protobuf:"bytes,1,opt,name=certificate_id,json=certificateId,proto3" json:"certificate_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetCertificateHeaderRequest) Reset() {
	*x = GetCertificateHeaderRequest{}
	mi := &file_agglayer_node_v1_node_state_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetCertificateHeaderRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetCertificateHeaderRequest) ProtoMessage() {}

func (x *GetCertificateHeaderRequest) ProtoReflect() protoreflect.Message {
	mi := &file_agglayer_node_v1_node_state_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetCertificateHeaderRequest.ProtoReflect.Descriptor instead.
func (*GetCertificateHeaderRequest) Descriptor() ([]byte, []int) {
	return file_agglayer_node_v1_node_state_proto_rawDescGZIP(), []int{0}
}

func (x *GetCertificateHeaderRequest) GetCertificateId() *v1.CertificateId {
	if x != nil {
		return x.CertificateId
	}
	return nil
}

// Response to the CertificateHeader request.
type GetCertificateHeaderResponse struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// The certificate header.
	CertificateHeader *v1.CertificateHeader `protobuf:"bytes,1,opt,name=certificate_header,json=certificateHeader,proto3" json:"certificate_header,omitempty"`
	unknownFields     protoimpl.UnknownFields
	sizeCache         protoimpl.SizeCache
}

func (x *GetCertificateHeaderResponse) Reset() {
	*x = GetCertificateHeaderResponse{}
	mi := &file_agglayer_node_v1_node_state_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetCertificateHeaderResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetCertificateHeaderResponse) ProtoMessage() {}

func (x *GetCertificateHeaderResponse) ProtoReflect() protoreflect.Message {
	mi := &file_agglayer_node_v1_node_state_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetCertificateHeaderResponse.ProtoReflect.Descriptor instead.
func (*GetCertificateHeaderResponse) Descriptor() ([]byte, []int) {
	return file_agglayer_node_v1_node_state_proto_rawDescGZIP(), []int{1}
}

func (x *GetCertificateHeaderResponse) GetCertificateHeader() *v1.CertificateHeader {
	if x != nil {
		return x.CertificateHeader
	}
	return nil
}

// Request to get the latest known/pending/settled certificate header for a network.
type GetLatestCertificateHeaderRequest struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Which type of latest certificate we want to get.
	Type LatestCertificateRequestType `protobuf:"varint,1,opt,name=type,proto3,enum=agglayer.node.v1.LatestCertificateRequestType" json:"type,omitempty"`
	// The network identifier.
	NetworkId     uint32 `protobuf:"varint,2,opt,name=network_id,json=networkId,proto3" json:"network_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetLatestCertificateHeaderRequest) Reset() {
	*x = GetLatestCertificateHeaderRequest{}
	mi := &file_agglayer_node_v1_node_state_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetLatestCertificateHeaderRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetLatestCertificateHeaderRequest) ProtoMessage() {}

func (x *GetLatestCertificateHeaderRequest) ProtoReflect() protoreflect.Message {
	mi := &file_agglayer_node_v1_node_state_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetLatestCertificateHeaderRequest.ProtoReflect.Descriptor instead.
func (*GetLatestCertificateHeaderRequest) Descriptor() ([]byte, []int) {
	return file_agglayer_node_v1_node_state_proto_rawDescGZIP(), []int{2}
}

func (x *GetLatestCertificateHeaderRequest) GetType() LatestCertificateRequestType {
	if x != nil {
		return x.Type
	}
	return LatestCertificateRequestType_LATEST_CERTIFICATE_REQUEST_TYPE_UNSPECIFIED
}

func (x *GetLatestCertificateHeaderRequest) GetNetworkId() uint32 {
	if x != nil {
		return x.NetworkId
	}
	return 0
}

// Response to the latest known/pending/settled certificate header request.
type GetLatestCertificateHeaderResponse struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// The latest certificate header.
	CertificateHeader *v1.CertificateHeader `protobuf:"bytes,1,opt,name=certificate_header,json=certificateHeader,proto3" json:"certificate_header,omitempty"`
	unknownFields     protoimpl.UnknownFields
	sizeCache         protoimpl.SizeCache
}

func (x *GetLatestCertificateHeaderResponse) Reset() {
	*x = GetLatestCertificateHeaderResponse{}
	mi := &file_agglayer_node_v1_node_state_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetLatestCertificateHeaderResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetLatestCertificateHeaderResponse) ProtoMessage() {}

func (x *GetLatestCertificateHeaderResponse) ProtoReflect() protoreflect.Message {
	mi := &file_agglayer_node_v1_node_state_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetLatestCertificateHeaderResponse.ProtoReflect.Descriptor instead.
func (*GetLatestCertificateHeaderResponse) Descriptor() ([]byte, []int) {
	return file_agglayer_node_v1_node_state_proto_rawDescGZIP(), []int{3}
}

func (x *GetLatestCertificateHeaderResponse) GetCertificateHeader() *v1.CertificateHeader {
	if x != nil {
		return x.CertificateHeader
	}
	return nil
}

var File_agglayer_node_v1_node_state_proto protoreflect.FileDescriptor

const file_agglayer_node_v1_node_state_proto_rawDesc = "" +
	"\n" +
	"!agglayer/node/v1/node_state.proto\x12\x10agglayer.node.v1\x1a(agglayer/node/types/v1/certificate.proto\x1a/agglayer/node/types/v1/certificate_header.proto\x1a+agglayer/node/types/v1/certificate_id.proto\"k\n" +
	"\x1bGetCertificateHeaderRequest\x12L\n" +
	"\x0ecertificate_id\x18\x01 \x01(\v2%.agglayer.node.types.v1.CertificateIdR\rcertificateId\"x\n" +
	"\x1cGetCertificateHeaderResponse\x12X\n" +
	"\x12certificate_header\x18\x01 \x01(\v2).agglayer.node.types.v1.CertificateHeaderR\x11certificateHeader\"\x86\x01\n" +
	"!GetLatestCertificateHeaderRequest\x12B\n" +
	"\x04type\x18\x01 \x01(\x0e2..agglayer.node.v1.LatestCertificateRequestTypeR\x04type\x12\x1d\n" +
	"\n" +
	"network_id\x18\x02 \x01(\rR\tnetworkId\"~\n" +
	"\"GetLatestCertificateHeaderResponse\x12X\n" +
	"\x12certificate_header\x18\x01 \x01(\v2).agglayer.node.types.v1.CertificateHeaderR\x11certificateHeader*\xec\x01\n" +
	"\x1dGetCertificateHeaderErrorKind\x121\n" +
	"-GET_CERTIFICATE_HEADER_ERROR_KIND_UNSPECIFIED\x10\x00\x123\n" +
	"/GET_CERTIFICATE_HEADER_ERROR_KIND_MISSING_FIELD\x10\x01\x122\n" +
	".GET_CERTIFICATE_HEADER_ERROR_KIND_INVALID_DATA\x10\x02\x12/\n" +
	"+GET_CERTIFICATE_HEADER_ERROR_KIND_NOT_FOUND\x10\x03*\x8e\x02\n" +
	"#GetLatestCertificateHeaderErrorKind\x128\n" +
	"4GET_LATEST_CERTIFICATE_HEADER_ERROR_KIND_UNSPECIFIED\x10\x00\x12:\n" +
	"6GET_LATEST_CERTIFICATE_HEADER_ERROR_KIND_MISSING_FIELD\x10\x01\x129\n" +
	"5GET_LATEST_CERTIFICATE_HEADER_ERROR_KIND_INVALID_DATA\x10\x02\x126\n" +
	"2GET_LATEST_CERTIFICATE_HEADER_ERROR_KIND_NOT_FOUND\x10\x03*\xa9\x01\n" +
	"\x1cLatestCertificateRequestType\x12/\n" +
	"+LATEST_CERTIFICATE_REQUEST_TYPE_UNSPECIFIED\x10\x00\x12+\n" +
	"'LATEST_CERTIFICATE_REQUEST_TYPE_PENDING\x10\x01\x12+\n" +
	"'LATEST_CERTIFICATE_REQUEST_TYPE_SETTLED\x10\x022\x93\x02\n" +
	"\x10NodeStateService\x12u\n" +
	"\x14GetCertificateHeader\x12-.agglayer.node.v1.GetCertificateHeaderRequest\x1a..agglayer.node.v1.GetCertificateHeaderResponse\x12\x87\x01\n" +
	"\x1aGetLatestCertificateHeader\x123.agglayer.node.v1.GetLatestCertificateHeaderRequest\x1a4.agglayer.node.v1.GetLatestCertificateHeaderResponseb\x06proto3"

var (
	file_agglayer_node_v1_node_state_proto_rawDescOnce sync.Once
	file_agglayer_node_v1_node_state_proto_rawDescData []byte
)

func file_agglayer_node_v1_node_state_proto_rawDescGZIP() []byte {
	file_agglayer_node_v1_node_state_proto_rawDescOnce.Do(func() {
		file_agglayer_node_v1_node_state_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_agglayer_node_v1_node_state_proto_rawDesc), len(file_agglayer_node_v1_node_state_proto_rawDesc)))
	})
	return file_agglayer_node_v1_node_state_proto_rawDescData
}

var file_agglayer_node_v1_node_state_proto_enumTypes = make([]protoimpl.EnumInfo, 3)
var file_agglayer_node_v1_node_state_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_agglayer_node_v1_node_state_proto_goTypes = []any{
	(GetCertificateHeaderErrorKind)(0),         // 0: agglayer.node.v1.GetCertificateHeaderErrorKind
	(GetLatestCertificateHeaderErrorKind)(0),   // 1: agglayer.node.v1.GetLatestCertificateHeaderErrorKind
	(LatestCertificateRequestType)(0),          // 2: agglayer.node.v1.LatestCertificateRequestType
	(*GetCertificateHeaderRequest)(nil),        // 3: agglayer.node.v1.GetCertificateHeaderRequest
	(*GetCertificateHeaderResponse)(nil),       // 4: agglayer.node.v1.GetCertificateHeaderResponse
	(*GetLatestCertificateHeaderRequest)(nil),  // 5: agglayer.node.v1.GetLatestCertificateHeaderRequest
	(*GetLatestCertificateHeaderResponse)(nil), // 6: agglayer.node.v1.GetLatestCertificateHeaderResponse
	(*v1.CertificateId)(nil),                   // 7: agglayer.node.types.v1.CertificateId
	(*v1.CertificateHeader)(nil),               // 8: agglayer.node.types.v1.CertificateHeader
}
var file_agglayer_node_v1_node_state_proto_depIdxs = []int32{
	7, // 0: agglayer.node.v1.GetCertificateHeaderRequest.certificate_id:type_name -> agglayer.node.types.v1.CertificateId
	8, // 1: agglayer.node.v1.GetCertificateHeaderResponse.certificate_header:type_name -> agglayer.node.types.v1.CertificateHeader
	2, // 2: agglayer.node.v1.GetLatestCertificateHeaderRequest.type:type_name -> agglayer.node.v1.LatestCertificateRequestType
	8, // 3: agglayer.node.v1.GetLatestCertificateHeaderResponse.certificate_header:type_name -> agglayer.node.types.v1.CertificateHeader
	3, // 4: agglayer.node.v1.NodeStateService.GetCertificateHeader:input_type -> agglayer.node.v1.GetCertificateHeaderRequest
	5, // 5: agglayer.node.v1.NodeStateService.GetLatestCertificateHeader:input_type -> agglayer.node.v1.GetLatestCertificateHeaderRequest
	4, // 6: agglayer.node.v1.NodeStateService.GetCertificateHeader:output_type -> agglayer.node.v1.GetCertificateHeaderResponse
	6, // 7: agglayer.node.v1.NodeStateService.GetLatestCertificateHeader:output_type -> agglayer.node.v1.GetLatestCertificateHeaderResponse
	6, // [6:8] is the sub-list for method output_type
	4, // [4:6] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_agglayer_node_v1_node_state_proto_init() }
func file_agglayer_node_v1_node_state_proto_init() {
	if File_agglayer_node_v1_node_state_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_agglayer_node_v1_node_state_proto_rawDesc), len(file_agglayer_node_v1_node_state_proto_rawDesc)),
			NumEnums:      3,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_agglayer_node_v1_node_state_proto_goTypes,
		DependencyIndexes: file_agglayer_node_v1_node_state_proto_depIdxs,
		EnumInfos:         file_agglayer_node_v1_node_state_proto_enumTypes,
		MessageInfos:      file_agglayer_node_v1_node_state_proto_msgTypes,
	}.Build()
	File_agglayer_node_v1_node_state_proto = out.File
	file_agglayer_node_v1_node_state_proto_goTypes = nil
	file_agglayer_node_v1_node_state_proto_depIdxs = nil
}
