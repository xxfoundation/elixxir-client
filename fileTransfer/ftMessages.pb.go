///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.17.3
// source: fileTransfer/ftMessages.proto

package fileTransfer

import (
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

type NewFileTransfer struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	FileName    string  `protobuf:"bytes,1,opt,name=fileName,proto3" json:"fileName,omitempty"`       // Name of the file; max 48 characters
	FileType    string  `protobuf:"bytes,2,opt,name=fileType,proto3" json:"fileType,omitempty"`       // Type of file; max 8 characters
	TransferKey []byte  `protobuf:"bytes,3,opt,name=transferKey,proto3" json:"transferKey,omitempty"` // 256 bit encryption key to identify the transfer
	TransferMac []byte  `protobuf:"bytes,4,opt,name=transferMac,proto3" json:"transferMac,omitempty"` // 256 bit MAC of the entire file
	NumParts    uint32  `protobuf:"varint,5,opt,name=numParts,proto3" json:"numParts,omitempty"`      // Number of file parts
	Size        uint32  `protobuf:"varint,6,opt,name=size,proto3" json:"size,omitempty"`              // The size of the file; max of 4 mB
	Retry       float32 `protobuf:"fixed32,7,opt,name=retry,proto3" json:"retry,omitempty"`           // Used to determine how many times to retry sending
	Preview     []byte  `protobuf:"bytes,8,opt,name=preview,proto3" json:"preview,omitempty"`         // A preview of the file; max of 4 kB
}

func (x *NewFileTransfer) Reset() {
	*x = NewFileTransfer{}
	if protoimpl.UnsafeEnabled {
		mi := &file_fileTransfer_ftMessages_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *NewFileTransfer) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NewFileTransfer) ProtoMessage() {}

func (x *NewFileTransfer) ProtoReflect() protoreflect.Message {
	mi := &file_fileTransfer_ftMessages_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NewFileTransfer.ProtoReflect.Descriptor instead.
func (*NewFileTransfer) Descriptor() ([]byte, []int) {
	return file_fileTransfer_ftMessages_proto_rawDescGZIP(), []int{0}
}

func (x *NewFileTransfer) GetFileName() string {
	if x != nil {
		return x.FileName
	}
	return ""
}

func (x *NewFileTransfer) GetFileType() string {
	if x != nil {
		return x.FileType
	}
	return ""
}

func (x *NewFileTransfer) GetTransferKey() []byte {
	if x != nil {
		return x.TransferKey
	}
	return nil
}

func (x *NewFileTransfer) GetTransferMac() []byte {
	if x != nil {
		return x.TransferMac
	}
	return nil
}

func (x *NewFileTransfer) GetNumParts() uint32 {
	if x != nil {
		return x.NumParts
	}
	return 0
}

func (x *NewFileTransfer) GetSize() uint32 {
	if x != nil {
		return x.Size
	}
	return 0
}

func (x *NewFileTransfer) GetRetry() float32 {
	if x != nil {
		return x.Retry
	}
	return 0
}

func (x *NewFileTransfer) GetPreview() []byte {
	if x != nil {
		return x.Preview
	}
	return nil
}

var File_fileTransfer_ftMessages_proto protoreflect.FileDescriptor

var file_fileTransfer_ftMessages_proto_rawDesc = []byte{
	0x0a, 0x1d, 0x66, 0x69, 0x6c, 0x65, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x66, 0x65, 0x72, 0x2f, 0x66,
	0x74, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x05, 0x70, 0x61, 0x72, 0x73, 0x65, 0x22, 0xed, 0x01, 0x0a, 0x0f, 0x4e, 0x65, 0x77, 0x46, 0x69,
	0x6c, 0x65, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x66, 0x65, 0x72, 0x12, 0x1a, 0x0a, 0x08, 0x66, 0x69,
	0x6c, 0x65, 0x4e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x66, 0x69,
	0x6c, 0x65, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x1a, 0x0a, 0x08, 0x66, 0x69, 0x6c, 0x65, 0x54, 0x79,
	0x70, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x66, 0x69, 0x6c, 0x65, 0x54, 0x79,
	0x70, 0x65, 0x12, 0x20, 0x0a, 0x0b, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x66, 0x65, 0x72, 0x4b, 0x65,
	0x79, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x0b, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x66, 0x65,
	0x72, 0x4b, 0x65, 0x79, 0x12, 0x20, 0x0a, 0x0b, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x66, 0x65, 0x72,
	0x4d, 0x61, 0x63, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x0b, 0x74, 0x72, 0x61, 0x6e, 0x73,
	0x66, 0x65, 0x72, 0x4d, 0x61, 0x63, 0x12, 0x1a, 0x0a, 0x08, 0x6e, 0x75, 0x6d, 0x50, 0x61, 0x72,
	0x74, 0x73, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x08, 0x6e, 0x75, 0x6d, 0x50, 0x61, 0x72,
	0x74, 0x73, 0x12, 0x12, 0x0a, 0x04, 0x73, 0x69, 0x7a, 0x65, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0d,
	0x52, 0x04, 0x73, 0x69, 0x7a, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x72, 0x65, 0x74, 0x72, 0x79, 0x18,
	0x07, 0x20, 0x01, 0x28, 0x02, 0x52, 0x05, 0x72, 0x65, 0x74, 0x72, 0x79, 0x12, 0x18, 0x0a, 0x07,
	0x70, 0x72, 0x65, 0x76, 0x69, 0x65, 0x77, 0x18, 0x08, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x07, 0x70,
	0x72, 0x65, 0x76, 0x69, 0x65, 0x77, 0x42, 0x0f, 0x5a, 0x0d, 0x66, 0x69, 0x6c, 0x65, 0x54, 0x72,
	0x61, 0x6e, 0x73, 0x66, 0x65, 0x72, 0x2f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_fileTransfer_ftMessages_proto_rawDescOnce sync.Once
	file_fileTransfer_ftMessages_proto_rawDescData = file_fileTransfer_ftMessages_proto_rawDesc
)

func file_fileTransfer_ftMessages_proto_rawDescGZIP() []byte {
	file_fileTransfer_ftMessages_proto_rawDescOnce.Do(func() {
		file_fileTransfer_ftMessages_proto_rawDescData = protoimpl.X.CompressGZIP(file_fileTransfer_ftMessages_proto_rawDescData)
	})
	return file_fileTransfer_ftMessages_proto_rawDescData
}

var file_fileTransfer_ftMessages_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_fileTransfer_ftMessages_proto_goTypes = []interface{}{
	(*NewFileTransfer)(nil), // 0: parse.NewFileTransfer
}
var file_fileTransfer_ftMessages_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_fileTransfer_ftMessages_proto_init() }
func file_fileTransfer_ftMessages_proto_init() {
	if File_fileTransfer_ftMessages_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_fileTransfer_ftMessages_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*NewFileTransfer); i {
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
			RawDescriptor: file_fileTransfer_ftMessages_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_fileTransfer_ftMessages_proto_goTypes,
		DependencyIndexes: file_fileTransfer_ftMessages_proto_depIdxs,
		MessageInfos:      file_fileTransfer_ftMessages_proto_msgTypes,
	}.Build()
	File_fileTransfer_ftMessages_proto = out.File
	file_fileTransfer_ftMessages_proto_rawDesc = nil
	file_fileTransfer_ftMessages_proto_goTypes = nil
	file_fileTransfer_ftMessages_proto_depIdxs = nil
}
