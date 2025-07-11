// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v5.29.3
// source: agglayer/interop/types/v1/imported_bridge_exit.proto

package v1

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

// Represents a token bridge exit originating on another network but claimed on
// the current network.
type ImportedBridgeExit struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// / The bridge exit initiated on another network, called the "sending"
	// / network. Need to verify that the destination network matches the
	// / current network, and that the bridge exit is included in an imported
	// / LER
	BridgeExit *BridgeExit `protobuf:"bytes,1,opt,name=bridge_exit,json=bridgeExit,proto3" json:"bridge_exit,omitempty"`
	// / The global index of the imported bridge exit.
	GlobalIndex *FixedBytes32 `protobuf:"bytes,2,opt,name=global_index,json=globalIndex,proto3" json:"global_index,omitempty"`
	// Which type of claim the imported bridge exit is from.
	//
	// Types that are valid to be assigned to Claim:
	//
	//	*ImportedBridgeExit_Mainnet
	//	*ImportedBridgeExit_Rollup
	Claim         isImportedBridgeExit_Claim `protobuf_oneof:"claim"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ImportedBridgeExit) Reset() {
	*x = ImportedBridgeExit{}
	mi := &file_agglayer_interop_types_v1_imported_bridge_exit_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ImportedBridgeExit) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ImportedBridgeExit) ProtoMessage() {}

func (x *ImportedBridgeExit) ProtoReflect() protoreflect.Message {
	mi := &file_agglayer_interop_types_v1_imported_bridge_exit_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ImportedBridgeExit.ProtoReflect.Descriptor instead.
func (*ImportedBridgeExit) Descriptor() ([]byte, []int) {
	return file_agglayer_interop_types_v1_imported_bridge_exit_proto_rawDescGZIP(), []int{0}
}

func (x *ImportedBridgeExit) GetBridgeExit() *BridgeExit {
	if x != nil {
		return x.BridgeExit
	}
	return nil
}

func (x *ImportedBridgeExit) GetGlobalIndex() *FixedBytes32 {
	if x != nil {
		return x.GlobalIndex
	}
	return nil
}

func (x *ImportedBridgeExit) GetClaim() isImportedBridgeExit_Claim {
	if x != nil {
		return x.Claim
	}
	return nil
}

func (x *ImportedBridgeExit) GetMainnet() *ClaimFromMainnet {
	if x != nil {
		if x, ok := x.Claim.(*ImportedBridgeExit_Mainnet); ok {
			return x.Mainnet
		}
	}
	return nil
}

func (x *ImportedBridgeExit) GetRollup() *ClaimFromRollup {
	if x != nil {
		if x, ok := x.Claim.(*ImportedBridgeExit_Rollup); ok {
			return x.Rollup
		}
	}
	return nil
}

type isImportedBridgeExit_Claim interface {
	isImportedBridgeExit_Claim()
}

type ImportedBridgeExit_Mainnet struct {
	// / The claim originated from the mainnet.
	Mainnet *ClaimFromMainnet `protobuf:"bytes,3,opt,name=mainnet,proto3,oneof"`
}

type ImportedBridgeExit_Rollup struct {
	// / The claim originated from the rollup.
	Rollup *ClaimFromRollup `protobuf:"bytes,4,opt,name=rollup,proto3,oneof"`
}

func (*ImportedBridgeExit_Mainnet) isImportedBridgeExit_Claim() {}

func (*ImportedBridgeExit_Rollup) isImportedBridgeExit_Claim() {}

var File_agglayer_interop_types_v1_imported_bridge_exit_proto protoreflect.FileDescriptor

const file_agglayer_interop_types_v1_imported_bridge_exit_proto_rawDesc = "" +
	"\n" +
	"4agglayer/interop/types/v1/imported_bridge_exit.proto\x12\x19agglayer.interop.types.v1\x1a+agglayer/interop/types/v1/bridge_exit.proto\x1a%agglayer/interop/types/v1/bytes.proto\x1a%agglayer/interop/types/v1/claim.proto\"\xc0\x02\n" +
	"\x12ImportedBridgeExit\x12F\n" +
	"\vbridge_exit\x18\x01 \x01(\v2%.agglayer.interop.types.v1.BridgeExitR\n" +
	"bridgeExit\x12J\n" +
	"\fglobal_index\x18\x02 \x01(\v2'.agglayer.interop.types.v1.FixedBytes32R\vglobalIndex\x12G\n" +
	"\amainnet\x18\x03 \x01(\v2+.agglayer.interop.types.v1.ClaimFromMainnetH\x00R\amainnet\x12D\n" +
	"\x06rollup\x18\x04 \x01(\v2*.agglayer.interop.types.v1.ClaimFromRollupH\x00R\x06rollupB\a\n" +
	"\x05claimb\x06proto3"

var (
	file_agglayer_interop_types_v1_imported_bridge_exit_proto_rawDescOnce sync.Once
	file_agglayer_interop_types_v1_imported_bridge_exit_proto_rawDescData []byte
)

func file_agglayer_interop_types_v1_imported_bridge_exit_proto_rawDescGZIP() []byte {
	file_agglayer_interop_types_v1_imported_bridge_exit_proto_rawDescOnce.Do(func() {
		file_agglayer_interop_types_v1_imported_bridge_exit_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_agglayer_interop_types_v1_imported_bridge_exit_proto_rawDesc), len(file_agglayer_interop_types_v1_imported_bridge_exit_proto_rawDesc)))
	})
	return file_agglayer_interop_types_v1_imported_bridge_exit_proto_rawDescData
}

var file_agglayer_interop_types_v1_imported_bridge_exit_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_agglayer_interop_types_v1_imported_bridge_exit_proto_goTypes = []any{
	(*ImportedBridgeExit)(nil), // 0: agglayer.interop.types.v1.ImportedBridgeExit
	(*BridgeExit)(nil),         // 1: agglayer.interop.types.v1.BridgeExit
	(*FixedBytes32)(nil),       // 2: agglayer.interop.types.v1.FixedBytes32
	(*ClaimFromMainnet)(nil),   // 3: agglayer.interop.types.v1.ClaimFromMainnet
	(*ClaimFromRollup)(nil),    // 4: agglayer.interop.types.v1.ClaimFromRollup
}
var file_agglayer_interop_types_v1_imported_bridge_exit_proto_depIdxs = []int32{
	1, // 0: agglayer.interop.types.v1.ImportedBridgeExit.bridge_exit:type_name -> agglayer.interop.types.v1.BridgeExit
	2, // 1: agglayer.interop.types.v1.ImportedBridgeExit.global_index:type_name -> agglayer.interop.types.v1.FixedBytes32
	3, // 2: agglayer.interop.types.v1.ImportedBridgeExit.mainnet:type_name -> agglayer.interop.types.v1.ClaimFromMainnet
	4, // 3: agglayer.interop.types.v1.ImportedBridgeExit.rollup:type_name -> agglayer.interop.types.v1.ClaimFromRollup
	4, // [4:4] is the sub-list for method output_type
	4, // [4:4] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_agglayer_interop_types_v1_imported_bridge_exit_proto_init() }
func file_agglayer_interop_types_v1_imported_bridge_exit_proto_init() {
	if File_agglayer_interop_types_v1_imported_bridge_exit_proto != nil {
		return
	}
	file_agglayer_interop_types_v1_bridge_exit_proto_init()
	file_agglayer_interop_types_v1_bytes_proto_init()
	file_agglayer_interop_types_v1_claim_proto_init()
	file_agglayer_interop_types_v1_imported_bridge_exit_proto_msgTypes[0].OneofWrappers = []any{
		(*ImportedBridgeExit_Mainnet)(nil),
		(*ImportedBridgeExit_Rollup)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_agglayer_interop_types_v1_imported_bridge_exit_proto_rawDesc), len(file_agglayer_interop_types_v1_imported_bridge_exit_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_agglayer_interop_types_v1_imported_bridge_exit_proto_goTypes,
		DependencyIndexes: file_agglayer_interop_types_v1_imported_bridge_exit_proto_depIdxs,
		MessageInfos:      file_agglayer_interop_types_v1_imported_bridge_exit_proto_msgTypes,
	}.Build()
	File_agglayer_interop_types_v1_imported_bridge_exit_proto = out.File
	file_agglayer_interop_types_v1_imported_bridge_exit_proto_goTypes = nil
	file_agglayer_interop_types_v1_imported_bridge_exit_proto_depIdxs = nil
}
