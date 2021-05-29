// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0
// 	protoc        v3.14.0
// source: messages/join_party.proto

package messages

import (
	proto "github.com/golang/protobuf/proto"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// This is a compile-time assertion that a sufficiently up-to-date version
// of the legacy proto package is being used.
const _ = proto.ProtoPackageIsVersion4

type JoinPartyLeaderComm_ResponseType int32

const (
	JoinPartyLeaderComm_Unknown        JoinPartyLeaderComm_ResponseType = 0
	JoinPartyLeaderComm_Success        JoinPartyLeaderComm_ResponseType = 1
	JoinPartyLeaderComm_Timeout        JoinPartyLeaderComm_ResponseType = 2
	JoinPartyLeaderComm_LeaderNotReady JoinPartyLeaderComm_ResponseType = 3
	JoinPartyLeaderComm_UnknownPeer    JoinPartyLeaderComm_ResponseType = 4
)

// Enum value maps for JoinPartyLeaderComm_ResponseType.
var (
	JoinPartyLeaderComm_ResponseType_name = map[int32]string{
		0: "Unknown",
		1: "Success",
		2: "Timeout",
		3: "LeaderNotReady",
		4: "UnknownPeer",
	}
	JoinPartyLeaderComm_ResponseType_value = map[string]int32{
		"Unknown":        0,
		"Success":        1,
		"Timeout":        2,
		"LeaderNotReady": 3,
		"UnknownPeer":    4,
	}
)

func (x JoinPartyLeaderComm_ResponseType) Enum() *JoinPartyLeaderComm_ResponseType {
	p := new(JoinPartyLeaderComm_ResponseType)
	*p = x
	return p
}

func (x JoinPartyLeaderComm_ResponseType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (JoinPartyLeaderComm_ResponseType) Descriptor() protoreflect.EnumDescriptor {
	return file_messages_join_party_proto_enumTypes[0].Descriptor()
}

func (JoinPartyLeaderComm_ResponseType) Type() protoreflect.EnumType {
	return &file_messages_join_party_proto_enumTypes[0]
}

func (x JoinPartyLeaderComm_ResponseType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use JoinPartyLeaderComm_ResponseType.Descriptor instead.
func (JoinPartyLeaderComm_ResponseType) EnumDescriptor() ([]byte, []int) {
	return file_messages_join_party_proto_rawDescGZIP(), []int{1, 0}
}

type JoinPartyRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ID string `protobuf:"bytes,1,opt,name=ID,proto3" json:"ID,omitempty"` // the unique hash id
}

func (x *JoinPartyRequest) Reset() {
	*x = JoinPartyRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_messages_join_party_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *JoinPartyRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*JoinPartyRequest) ProtoMessage() {}

func (x *JoinPartyRequest) ProtoReflect() protoreflect.Message {
	mi := &file_messages_join_party_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use JoinPartyRequest.ProtoReflect.Descriptor instead.
func (*JoinPartyRequest) Descriptor() ([]byte, []int) {
	return file_messages_join_party_proto_rawDescGZIP(), []int{0}
}

func (x *JoinPartyRequest) GetID() string {
	if x != nil {
		return x.ID
	}
	return ""
}

type JoinPartyLeaderComm struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ID      string                           `protobuf:"bytes,1,opt,name=ID,proto3" json:"ID,omitempty"`                                                     // unique hash id
	MsgType string                           `protobuf:"bytes,2,opt,name=MsgType,proto3" json:"MsgType,omitempty"`                                           // unique hash id
	Type    JoinPartyLeaderComm_ResponseType `protobuf:"varint,3,opt,name=type,proto3,enum=messages.JoinPartyLeaderComm_ResponseType" json:"type,omitempty"` // result
	PeerIDs []string                         `protobuf:"bytes,4,rep,name=PeerIDs,proto3" json:"PeerIDs,omitempty"`                                           // if Success , this will be the list of peers to form the ceremony, if fail , this will be the peers that are available
}

func (x *JoinPartyLeaderComm) Reset() {
	*x = JoinPartyLeaderComm{}
	if protoimpl.UnsafeEnabled {
		mi := &file_messages_join_party_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *JoinPartyLeaderComm) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*JoinPartyLeaderComm) ProtoMessage() {}

func (x *JoinPartyLeaderComm) ProtoReflect() protoreflect.Message {
	mi := &file_messages_join_party_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use JoinPartyLeaderComm.ProtoReflect.Descriptor instead.
func (*JoinPartyLeaderComm) Descriptor() ([]byte, []int) {
	return file_messages_join_party_proto_rawDescGZIP(), []int{1}
}

func (x *JoinPartyLeaderComm) GetID() string {
	if x != nil {
		return x.ID
	}
	return ""
}

func (x *JoinPartyLeaderComm) GetMsgType() string {
	if x != nil {
		return x.MsgType
	}
	return ""
}

func (x *JoinPartyLeaderComm) GetType() JoinPartyLeaderComm_ResponseType {
	if x != nil {
		return x.Type
	}
	return JoinPartyLeaderComm_Unknown
}

func (x *JoinPartyLeaderComm) GetPeerIDs() []string {
	if x != nil {
		return x.PeerIDs
	}
	return nil
}

var File_messages_join_party_proto protoreflect.FileDescriptor

var file_messages_join_party_proto_rawDesc = []byte{
	0x0a, 0x19, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x73, 0x2f, 0x6a, 0x6f, 0x69, 0x6e, 0x5f,
	0x70, 0x61, 0x72, 0x74, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x08, 0x6d, 0x65, 0x73,
	0x73, 0x61, 0x67, 0x65, 0x73, 0x22, 0x22, 0x0a, 0x10, 0x4a, 0x6f, 0x69, 0x6e, 0x50, 0x61, 0x72,
	0x74, 0x79, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x0e, 0x0a, 0x02, 0x49, 0x44, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x49, 0x44, 0x22, 0xf5, 0x01, 0x0a, 0x13, 0x4a, 0x6f,
	0x69, 0x6e, 0x50, 0x61, 0x72, 0x74, 0x79, 0x4c, 0x65, 0x61, 0x64, 0x65, 0x72, 0x43, 0x6f, 0x6d,
	0x6d, 0x12, 0x0e, 0x0a, 0x02, 0x49, 0x44, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x49,
	0x44, 0x12, 0x18, 0x0a, 0x07, 0x4d, 0x73, 0x67, 0x54, 0x79, 0x70, 0x65, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x07, 0x4d, 0x73, 0x67, 0x54, 0x79, 0x70, 0x65, 0x12, 0x3e, 0x0a, 0x04, 0x74,
	0x79, 0x70, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x2a, 0x2e, 0x6d, 0x65, 0x73, 0x73,
	0x61, 0x67, 0x65, 0x73, 0x2e, 0x4a, 0x6f, 0x69, 0x6e, 0x50, 0x61, 0x72, 0x74, 0x79, 0x4c, 0x65,
	0x61, 0x64, 0x65, 0x72, 0x43, 0x6f, 0x6d, 0x6d, 0x2e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x54, 0x79, 0x70, 0x65, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x50,
	0x65, 0x65, 0x72, 0x49, 0x44, 0x73, 0x18, 0x04, 0x20, 0x03, 0x28, 0x09, 0x52, 0x07, 0x50, 0x65,
	0x65, 0x72, 0x49, 0x44, 0x73, 0x22, 0x5a, 0x0a, 0x0c, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x54, 0x79, 0x70, 0x65, 0x12, 0x0b, 0x0a, 0x07, 0x55, 0x6e, 0x6b, 0x6e, 0x6f, 0x77, 0x6e,
	0x10, 0x00, 0x12, 0x0b, 0x0a, 0x07, 0x53, 0x75, 0x63, 0x63, 0x65, 0x73, 0x73, 0x10, 0x01, 0x12,
	0x0b, 0x0a, 0x07, 0x54, 0x69, 0x6d, 0x65, 0x6f, 0x75, 0x74, 0x10, 0x02, 0x12, 0x12, 0x0a, 0x0e,
	0x4c, 0x65, 0x61, 0x64, 0x65, 0x72, 0x4e, 0x6f, 0x74, 0x52, 0x65, 0x61, 0x64, 0x79, 0x10, 0x03,
	0x12, 0x0f, 0x0a, 0x0b, 0x55, 0x6e, 0x6b, 0x6e, 0x6f, 0x77, 0x6e, 0x50, 0x65, 0x65, 0x72, 0x10,
	0x04, 0x42, 0x2a, 0x5a, 0x28, 0x67, 0x69, 0x74, 0x6c, 0x61, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x74, 0x68, 0x6f, 0x72, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x2f, 0x74, 0x73, 0x73, 0x2f, 0x67, 0x6f,
	0x2d, 0x74, 0x73, 0x73, 0x2f, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x73, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_messages_join_party_proto_rawDescOnce sync.Once
	file_messages_join_party_proto_rawDescData = file_messages_join_party_proto_rawDesc
)

func file_messages_join_party_proto_rawDescGZIP() []byte {
	file_messages_join_party_proto_rawDescOnce.Do(func() {
		file_messages_join_party_proto_rawDescData = protoimpl.X.CompressGZIP(file_messages_join_party_proto_rawDescData)
	})
	return file_messages_join_party_proto_rawDescData
}

var file_messages_join_party_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_messages_join_party_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_messages_join_party_proto_goTypes = []interface{}{
	(JoinPartyLeaderComm_ResponseType)(0), // 0: messages.JoinPartyLeaderComm.ResponseType
	(*JoinPartyRequest)(nil),              // 1: messages.JoinPartyRequest
	(*JoinPartyLeaderComm)(nil),           // 2: messages.JoinPartyLeaderComm
}
var file_messages_join_party_proto_depIdxs = []int32{
	0, // 0: messages.JoinPartyLeaderComm.type:type_name -> messages.JoinPartyLeaderComm.ResponseType
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_messages_join_party_proto_init() }
func file_messages_join_party_proto_init() {
	if File_messages_join_party_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_messages_join_party_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*JoinPartyRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_messages_join_party_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*JoinPartyLeaderComm); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_messages_join_party_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_messages_join_party_proto_goTypes,
		DependencyIndexes: file_messages_join_party_proto_depIdxs,
		EnumInfos:         file_messages_join_party_proto_enumTypes,
		MessageInfos:      file_messages_join_party_proto_msgTypes,
	}.Build()
	File_messages_join_party_proto = out.File
	file_messages_join_party_proto_rawDesc = nil
	file_messages_join_party_proto_goTypes = nil
	file_messages_join_party_proto_depIdxs = nil
}
