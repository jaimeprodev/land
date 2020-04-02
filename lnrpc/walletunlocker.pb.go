// Code generated by protoc-gen-go. DO NOT EDIT.
// source: walletunlocker.proto

package lnrpc

import (
	context "context"
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	_ "google.golang.org/genproto/googleapis/api/annotations"
	grpc "google.golang.org/grpc"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type GenSeedRequest struct {
	//*
	//aezeed_passphrase is an optional user provided passphrase that will be used
	//to encrypt the generated aezeed cipher seed. When using REST, this field
	//must be encoded as base64.
	AezeedPassphrase []byte `protobuf:"bytes,1,opt,name=aezeed_passphrase,json=aezeedPassphrase,proto3" json:"aezeed_passphrase,omitempty"`
	//*
	//seed_entropy is an optional 16-bytes generated via CSPRNG. If not
	//specified, then a fresh set of randomness will be used to create the seed.
	//When using REST, this field must be encoded as base64.
	SeedEntropy          []byte   `protobuf:"bytes,2,opt,name=seed_entropy,json=seedEntropy,proto3" json:"seed_entropy,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *GenSeedRequest) Reset()         { *m = GenSeedRequest{} }
func (m *GenSeedRequest) String() string { return proto.CompactTextString(m) }
func (*GenSeedRequest) ProtoMessage()    {}
func (*GenSeedRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_76e3ed10ed53e4fd, []int{0}
}

func (m *GenSeedRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GenSeedRequest.Unmarshal(m, b)
}
func (m *GenSeedRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GenSeedRequest.Marshal(b, m, deterministic)
}
func (m *GenSeedRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GenSeedRequest.Merge(m, src)
}
func (m *GenSeedRequest) XXX_Size() int {
	return xxx_messageInfo_GenSeedRequest.Size(m)
}
func (m *GenSeedRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_GenSeedRequest.DiscardUnknown(m)
}

var xxx_messageInfo_GenSeedRequest proto.InternalMessageInfo

func (m *GenSeedRequest) GetAezeedPassphrase() []byte {
	if m != nil {
		return m.AezeedPassphrase
	}
	return nil
}

func (m *GenSeedRequest) GetSeedEntropy() []byte {
	if m != nil {
		return m.SeedEntropy
	}
	return nil
}

type GenSeedResponse struct {
	//*
	//cipher_seed_mnemonic is a 24-word mnemonic that encodes a prior aezeed
	//cipher seed obtained by the user. This field is optional, as if not
	//provided, then the daemon will generate a new cipher seed for the user.
	//Otherwise, then the daemon will attempt to recover the wallet state linked
	//to this cipher seed.
	CipherSeedMnemonic []string `protobuf:"bytes,1,rep,name=cipher_seed_mnemonic,json=cipherSeedMnemonic,proto3" json:"cipher_seed_mnemonic,omitempty"`
	//*
	//enciphered_seed are the raw aezeed cipher seed bytes. This is the raw
	//cipher text before run through our mnemonic encoding scheme.
	EncipheredSeed       []byte   `protobuf:"bytes,2,opt,name=enciphered_seed,json=encipheredSeed,proto3" json:"enciphered_seed,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *GenSeedResponse) Reset()         { *m = GenSeedResponse{} }
func (m *GenSeedResponse) String() string { return proto.CompactTextString(m) }
func (*GenSeedResponse) ProtoMessage()    {}
func (*GenSeedResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_76e3ed10ed53e4fd, []int{1}
}

func (m *GenSeedResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GenSeedResponse.Unmarshal(m, b)
}
func (m *GenSeedResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GenSeedResponse.Marshal(b, m, deterministic)
}
func (m *GenSeedResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GenSeedResponse.Merge(m, src)
}
func (m *GenSeedResponse) XXX_Size() int {
	return xxx_messageInfo_GenSeedResponse.Size(m)
}
func (m *GenSeedResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_GenSeedResponse.DiscardUnknown(m)
}

var xxx_messageInfo_GenSeedResponse proto.InternalMessageInfo

func (m *GenSeedResponse) GetCipherSeedMnemonic() []string {
	if m != nil {
		return m.CipherSeedMnemonic
	}
	return nil
}

func (m *GenSeedResponse) GetEncipheredSeed() []byte {
	if m != nil {
		return m.EncipheredSeed
	}
	return nil
}

type InitWalletRequest struct {
	//*
	//wallet_password is the passphrase that should be used to encrypt the
	//wallet. This MUST be at least 8 chars in length. After creation, this
	//password is required to unlock the daemon. When using REST, this field
	//must be encoded as base64.
	WalletPassword []byte `protobuf:"bytes,1,opt,name=wallet_password,json=walletPassword,proto3" json:"wallet_password,omitempty"`
	//*
	//cipher_seed_mnemonic is a 24-word mnemonic that encodes a prior aezeed
	//cipher seed obtained by the user. This may have been generated by the
	//GenSeed method, or be an existing seed.
	CipherSeedMnemonic []string `protobuf:"bytes,2,rep,name=cipher_seed_mnemonic,json=cipherSeedMnemonic,proto3" json:"cipher_seed_mnemonic,omitempty"`
	//*
	//aezeed_passphrase is an optional user provided passphrase that will be used
	//to encrypt the generated aezeed cipher seed. When using REST, this field
	//must be encoded as base64.
	AezeedPassphrase []byte `protobuf:"bytes,3,opt,name=aezeed_passphrase,json=aezeedPassphrase,proto3" json:"aezeed_passphrase,omitempty"`
	//*
	//recovery_window is an optional argument specifying the address lookahead
	//when restoring a wallet seed. The recovery window applies to each
	//individual branch of the BIP44 derivation paths. Supplying a recovery
	//window of zero indicates that no addresses should be recovered, such after
	//the first initialization of the wallet.
	RecoveryWindow int32 `protobuf:"varint,4,opt,name=recovery_window,json=recoveryWindow,proto3" json:"recovery_window,omitempty"`
	//*
	//channel_backups is an optional argument that allows clients to recover the
	//settled funds within a set of channels. This should be populated if the
	//user was unable to close out all channels and sweep funds before partial or
	//total data loss occurred. If specified, then after on-chain recovery of
	//funds, lnd begin to carry out the data loss recovery protocol in order to
	//recover the funds in each channel from a remote force closed transaction.
	ChannelBackups       *ChanBackupSnapshot `protobuf:"bytes,5,opt,name=channel_backups,json=channelBackups,proto3" json:"channel_backups,omitempty"`
	XXX_NoUnkeyedLiteral struct{}            `json:"-"`
	XXX_unrecognized     []byte              `json:"-"`
	XXX_sizecache        int32               `json:"-"`
}

func (m *InitWalletRequest) Reset()         { *m = InitWalletRequest{} }
func (m *InitWalletRequest) String() string { return proto.CompactTextString(m) }
func (*InitWalletRequest) ProtoMessage()    {}
func (*InitWalletRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_76e3ed10ed53e4fd, []int{2}
}

func (m *InitWalletRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_InitWalletRequest.Unmarshal(m, b)
}
func (m *InitWalletRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_InitWalletRequest.Marshal(b, m, deterministic)
}
func (m *InitWalletRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_InitWalletRequest.Merge(m, src)
}
func (m *InitWalletRequest) XXX_Size() int {
	return xxx_messageInfo_InitWalletRequest.Size(m)
}
func (m *InitWalletRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_InitWalletRequest.DiscardUnknown(m)
}

var xxx_messageInfo_InitWalletRequest proto.InternalMessageInfo

func (m *InitWalletRequest) GetWalletPassword() []byte {
	if m != nil {
		return m.WalletPassword
	}
	return nil
}

func (m *InitWalletRequest) GetCipherSeedMnemonic() []string {
	if m != nil {
		return m.CipherSeedMnemonic
	}
	return nil
}

func (m *InitWalletRequest) GetAezeedPassphrase() []byte {
	if m != nil {
		return m.AezeedPassphrase
	}
	return nil
}

func (m *InitWalletRequest) GetRecoveryWindow() int32 {
	if m != nil {
		return m.RecoveryWindow
	}
	return 0
}

func (m *InitWalletRequest) GetChannelBackups() *ChanBackupSnapshot {
	if m != nil {
		return m.ChannelBackups
	}
	return nil
}

type InitWalletResponse struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *InitWalletResponse) Reset()         { *m = InitWalletResponse{} }
func (m *InitWalletResponse) String() string { return proto.CompactTextString(m) }
func (*InitWalletResponse) ProtoMessage()    {}
func (*InitWalletResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_76e3ed10ed53e4fd, []int{3}
}

func (m *InitWalletResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_InitWalletResponse.Unmarshal(m, b)
}
func (m *InitWalletResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_InitWalletResponse.Marshal(b, m, deterministic)
}
func (m *InitWalletResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_InitWalletResponse.Merge(m, src)
}
func (m *InitWalletResponse) XXX_Size() int {
	return xxx_messageInfo_InitWalletResponse.Size(m)
}
func (m *InitWalletResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_InitWalletResponse.DiscardUnknown(m)
}

var xxx_messageInfo_InitWalletResponse proto.InternalMessageInfo

type UnlockWalletRequest struct {
	//*
	//wallet_password should be the current valid passphrase for the daemon. This
	//will be required to decrypt on-disk material that the daemon requires to
	//function properly. When using REST, this field must be encoded as base64.
	WalletPassword []byte `protobuf:"bytes,1,opt,name=wallet_password,json=walletPassword,proto3" json:"wallet_password,omitempty"`
	//*
	//recovery_window is an optional argument specifying the address lookahead
	//when restoring a wallet seed. The recovery window applies to each
	//individual branch of the BIP44 derivation paths. Supplying a recovery
	//window of zero indicates that no addresses should be recovered, such after
	//the first initialization of the wallet.
	RecoveryWindow int32 `protobuf:"varint,2,opt,name=recovery_window,json=recoveryWindow,proto3" json:"recovery_window,omitempty"`
	//*
	//channel_backups is an optional argument that allows clients to recover the
	//settled funds within a set of channels. This should be populated if the
	//user was unable to close out all channels and sweep funds before partial or
	//total data loss occurred. If specified, then after on-chain recovery of
	//funds, lnd begin to carry out the data loss recovery protocol in order to
	//recover the funds in each channel from a remote force closed transaction.
	ChannelBackups       *ChanBackupSnapshot `protobuf:"bytes,3,opt,name=channel_backups,json=channelBackups,proto3" json:"channel_backups,omitempty"`
	XXX_NoUnkeyedLiteral struct{}            `json:"-"`
	XXX_unrecognized     []byte              `json:"-"`
	XXX_sizecache        int32               `json:"-"`
}

func (m *UnlockWalletRequest) Reset()         { *m = UnlockWalletRequest{} }
func (m *UnlockWalletRequest) String() string { return proto.CompactTextString(m) }
func (*UnlockWalletRequest) ProtoMessage()    {}
func (*UnlockWalletRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_76e3ed10ed53e4fd, []int{4}
}

func (m *UnlockWalletRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_UnlockWalletRequest.Unmarshal(m, b)
}
func (m *UnlockWalletRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_UnlockWalletRequest.Marshal(b, m, deterministic)
}
func (m *UnlockWalletRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_UnlockWalletRequest.Merge(m, src)
}
func (m *UnlockWalletRequest) XXX_Size() int {
	return xxx_messageInfo_UnlockWalletRequest.Size(m)
}
func (m *UnlockWalletRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_UnlockWalletRequest.DiscardUnknown(m)
}

var xxx_messageInfo_UnlockWalletRequest proto.InternalMessageInfo

func (m *UnlockWalletRequest) GetWalletPassword() []byte {
	if m != nil {
		return m.WalletPassword
	}
	return nil
}

func (m *UnlockWalletRequest) GetRecoveryWindow() int32 {
	if m != nil {
		return m.RecoveryWindow
	}
	return 0
}

func (m *UnlockWalletRequest) GetChannelBackups() *ChanBackupSnapshot {
	if m != nil {
		return m.ChannelBackups
	}
	return nil
}

type UnlockWalletResponse struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *UnlockWalletResponse) Reset()         { *m = UnlockWalletResponse{} }
func (m *UnlockWalletResponse) String() string { return proto.CompactTextString(m) }
func (*UnlockWalletResponse) ProtoMessage()    {}
func (*UnlockWalletResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_76e3ed10ed53e4fd, []int{5}
}

func (m *UnlockWalletResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_UnlockWalletResponse.Unmarshal(m, b)
}
func (m *UnlockWalletResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_UnlockWalletResponse.Marshal(b, m, deterministic)
}
func (m *UnlockWalletResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_UnlockWalletResponse.Merge(m, src)
}
func (m *UnlockWalletResponse) XXX_Size() int {
	return xxx_messageInfo_UnlockWalletResponse.Size(m)
}
func (m *UnlockWalletResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_UnlockWalletResponse.DiscardUnknown(m)
}

var xxx_messageInfo_UnlockWalletResponse proto.InternalMessageInfo

type ChangePasswordRequest struct {
	//*
	//current_password should be the current valid passphrase used to unlock the
	//daemon. When using REST, this field must be encoded as base64.
	CurrentPassword []byte `protobuf:"bytes,1,opt,name=current_password,json=currentPassword,proto3" json:"current_password,omitempty"`
	//*
	//new_password should be the new passphrase that will be needed to unlock the
	//daemon. When using REST, this field must be encoded as base64.
	NewPassword          []byte   `protobuf:"bytes,2,opt,name=new_password,json=newPassword,proto3" json:"new_password,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ChangePasswordRequest) Reset()         { *m = ChangePasswordRequest{} }
func (m *ChangePasswordRequest) String() string { return proto.CompactTextString(m) }
func (*ChangePasswordRequest) ProtoMessage()    {}
func (*ChangePasswordRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_76e3ed10ed53e4fd, []int{6}
}

func (m *ChangePasswordRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ChangePasswordRequest.Unmarshal(m, b)
}
func (m *ChangePasswordRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ChangePasswordRequest.Marshal(b, m, deterministic)
}
func (m *ChangePasswordRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ChangePasswordRequest.Merge(m, src)
}
func (m *ChangePasswordRequest) XXX_Size() int {
	return xxx_messageInfo_ChangePasswordRequest.Size(m)
}
func (m *ChangePasswordRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_ChangePasswordRequest.DiscardUnknown(m)
}

var xxx_messageInfo_ChangePasswordRequest proto.InternalMessageInfo

func (m *ChangePasswordRequest) GetCurrentPassword() []byte {
	if m != nil {
		return m.CurrentPassword
	}
	return nil
}

func (m *ChangePasswordRequest) GetNewPassword() []byte {
	if m != nil {
		return m.NewPassword
	}
	return nil
}

type ChangePasswordResponse struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ChangePasswordResponse) Reset()         { *m = ChangePasswordResponse{} }
func (m *ChangePasswordResponse) String() string { return proto.CompactTextString(m) }
func (*ChangePasswordResponse) ProtoMessage()    {}
func (*ChangePasswordResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_76e3ed10ed53e4fd, []int{7}
}

func (m *ChangePasswordResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ChangePasswordResponse.Unmarshal(m, b)
}
func (m *ChangePasswordResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ChangePasswordResponse.Marshal(b, m, deterministic)
}
func (m *ChangePasswordResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ChangePasswordResponse.Merge(m, src)
}
func (m *ChangePasswordResponse) XXX_Size() int {
	return xxx_messageInfo_ChangePasswordResponse.Size(m)
}
func (m *ChangePasswordResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_ChangePasswordResponse.DiscardUnknown(m)
}

var xxx_messageInfo_ChangePasswordResponse proto.InternalMessageInfo

func init() {
	proto.RegisterType((*GenSeedRequest)(nil), "lnrpc.GenSeedRequest")
	proto.RegisterType((*GenSeedResponse)(nil), "lnrpc.GenSeedResponse")
	proto.RegisterType((*InitWalletRequest)(nil), "lnrpc.InitWalletRequest")
	proto.RegisterType((*InitWalletResponse)(nil), "lnrpc.InitWalletResponse")
	proto.RegisterType((*UnlockWalletRequest)(nil), "lnrpc.UnlockWalletRequest")
	proto.RegisterType((*UnlockWalletResponse)(nil), "lnrpc.UnlockWalletResponse")
	proto.RegisterType((*ChangePasswordRequest)(nil), "lnrpc.ChangePasswordRequest")
	proto.RegisterType((*ChangePasswordResponse)(nil), "lnrpc.ChangePasswordResponse")
}

func init() { proto.RegisterFile("walletunlocker.proto", fileDescriptor_76e3ed10ed53e4fd) }

var fileDescriptor_76e3ed10ed53e4fd = []byte{
	// 595 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x9c, 0x54, 0xdf, 0x4a, 0xdc, 0x4e,
	0x14, 0x26, 0xbb, 0x3f, 0x7f, 0xc5, 0xe3, 0x36, 0xd1, 0x71, 0xb5, 0x31, 0x2a, 0xac, 0x81, 0xb2,
	0x5b, 0x0b, 0x9b, 0xd6, 0xde, 0x79, 0x69, 0x29, 0xa5, 0x17, 0x82, 0xac, 0x88, 0x50, 0x0a, 0xdb,
	0x98, 0x1c, 0x92, 0x60, 0x3c, 0x93, 0x4e, 0x66, 0x0d, 0xf6, 0xb2, 0x57, 0xbd, 0x2d, 0x7d, 0x8c,
	0x3e, 0x4e, 0x5f, 0xa1, 0x0f, 0x52, 0x32, 0x33, 0x71, 0xd5, 0xdd, 0x85, 0xb6, 0x77, 0xe1, 0x3b,
	0x7f, 0xbe, 0xf3, 0x7d, 0xe7, 0x64, 0xa0, 0x5b, 0x85, 0x79, 0x8e, 0x72, 0x42, 0x39, 0x8f, 0x2e,
	0x51, 0x0c, 0x0b, 0xc1, 0x25, 0x67, 0x4b, 0x39, 0x89, 0x22, 0xf2, 0x76, 0x12, 0xce, 0x93, 0x1c,
	0x83, 0xb0, 0xc8, 0x82, 0x90, 0x88, 0xcb, 0x50, 0x66, 0x9c, 0x4a, 0x9d, 0xe4, 0x2d, 0x8b, 0x22,
	0xd2, 0x9f, 0xfe, 0x47, 0xb0, 0xdf, 0x22, 0x9d, 0x22, 0xc6, 0x23, 0xfc, 0x34, 0xc1, 0x52, 0xb2,
	0xe7, 0xb0, 0x16, 0xe2, 0x67, 0xc4, 0x78, 0x5c, 0x84, 0x65, 0x59, 0xa4, 0x22, 0x2c, 0xd1, 0xb5,
	0x7a, 0xd6, 0xa0, 0x33, 0x5a, 0xd5, 0x81, 0x93, 0x5b, 0x9c, 0xed, 0x41, 0xa7, 0xac, 0x53, 0x91,
	0xa4, 0xe0, 0xc5, 0x8d, 0xdb, 0x52, 0x79, 0x2b, 0x35, 0xf6, 0x46, 0x43, 0x7e, 0x0e, 0xce, 0x2d,
	0x43, 0x59, 0x70, 0x2a, 0x91, 0xbd, 0x80, 0x6e, 0x94, 0x15, 0x29, 0x8a, 0xb1, 0x2a, 0xbe, 0x22,
	0xbc, 0xe2, 0x94, 0x45, 0xae, 0xd5, 0x6b, 0x0f, 0x96, 0x47, 0x4c, 0xc7, 0xea, 0x8a, 0x63, 0x13,
	0x61, 0x7d, 0x70, 0x90, 0x34, 0x8e, 0xb1, 0xaa, 0x32, 0x54, 0xf6, 0x14, 0xae, 0x0b, 0xfc, 0xaf,
	0x2d, 0x58, 0x7b, 0x47, 0x99, 0x3c, 0x57, 0xe6, 0x34, 0x9a, 0xfa, 0xe0, 0x68, 0xb7, 0x94, 0xa6,
	0x8a, 0x8b, 0xd8, 0x28, 0xb2, 0x35, 0x7c, 0x62, 0xd0, 0x85, 0x93, 0xb5, 0x16, 0x4e, 0x36, 0xd7,
	0xae, 0xf6, 0x02, 0xbb, 0xfa, 0xe0, 0x08, 0x8c, 0xf8, 0x35, 0x8a, 0x9b, 0x71, 0x95, 0x51, 0xcc,
	0x2b, 0xf7, 0xbf, 0x9e, 0x35, 0x58, 0x1a, 0xd9, 0x0d, 0x7c, 0xae, 0x50, 0x76, 0x04, 0x4e, 0x94,
	0x86, 0x44, 0x98, 0x8f, 0x2f, 0xc2, 0xe8, 0x72, 0x52, 0x94, 0xee, 0x52, 0xcf, 0x1a, 0xac, 0x1c,
	0x6c, 0x0d, 0xd5, 0x82, 0x87, 0xaf, 0xd3, 0x90, 0x8e, 0x54, 0xe4, 0x94, 0xc2, 0xa2, 0x4c, 0xb9,
	0x1c, 0xd9, 0xa6, 0x42, 0xc3, 0xa5, 0xdf, 0x05, 0x76, 0xd7, 0x09, 0xed, 0xbd, 0xff, 0xc3, 0x82,
	0xf5, 0x33, 0x75, 0x33, 0xff, 0x68, 0xd1, 0x1c, 0x0d, 0xad, 0x3f, 0xd5, 0xd0, 0xfe, 0x5b, 0x0d,
	0x9b, 0xd0, 0xbd, 0x3f, 0xac, 0x51, 0x81, 0xb0, 0x51, 0x57, 0x27, 0xd8, 0x8c, 0xd5, 0xc8, 0x78,
	0x06, 0xab, 0xd1, 0x44, 0x08, 0xa4, 0x19, 0x1d, 0x8e, 0xc1, 0x6f, 0x85, 0xec, 0x41, 0x87, 0xb0,
	0x9a, 0xa6, 0x99, 0xdb, 0x25, 0xac, 0x9a, 0x14, 0xdf, 0x85, 0xcd, 0x87, 0x34, 0x7a, 0x80, 0x83,
	0x6f, 0x6d, 0xb0, 0xf5, 0x4c, 0x67, 0xe6, 0x07, 0x64, 0xc7, 0xf0, 0xc8, 0x1c, 0x3a, 0xdb, 0x30,
	0x0a, 0xef, 0xff, 0x5a, 0xde, 0xe6, 0x43, 0xd8, 0xa8, 0x59, 0xff, 0xf2, 0xf3, 0xd7, 0xf7, 0xd6,
	0x63, 0xb6, 0x12, 0x5c, 0xbf, 0x0c, 0x12, 0xa4, 0xfa, 0xf6, 0xd8, 0x07, 0x80, 0xe9, 0xfa, 0x98,
	0x6b, 0x4a, 0x67, 0x6e, 0xdb, 0xdb, 0x9a, 0x13, 0x31, 0x7d, 0xb7, 0x54, 0xdf, 0x75, 0xdf, 0xae,
	0xfb, 0x66, 0x94, 0x49, 0xbd, 0xca, 0x43, 0x6b, 0x9f, 0xc5, 0xd0, 0xb9, 0x6b, 0x2c, 0xf3, 0x4c,
	0x97, 0x39, 0xa7, 0xe1, 0x6d, 0xcf, 0x8d, 0x19, 0x8e, 0x6d, 0xc5, 0xb1, 0xe1, 0xaf, 0xd6, 0x1c,
	0xfa, 0x31, 0x9a, 0xb2, 0xe4, 0x60, 0xdf, 0xf7, 0x8f, 0xed, 0xdc, 0xd9, 0xfd, 0xcc, 0xf6, 0xbc,
	0xdd, 0x05, 0x51, 0xc3, 0xb5, 0xab, 0xb8, 0x9e, 0xf8, 0xac, 0xe6, 0x8a, 0x54, 0x4e, 0xb3, 0xbd,
	0x43, 0x6b, 0xff, 0xa8, 0xff, 0xfe, 0x69, 0x92, 0xc9, 0x74, 0x72, 0x31, 0x8c, 0xf8, 0x55, 0x90,
	0x67, 0x49, 0x2a, 0x29, 0xa3, 0x84, 0x50, 0x56, 0x5c, 0x5c, 0x06, 0x39, 0xc5, 0x81, 0x6a, 0x7f,
	0xf1, 0xbf, 0x7a, 0xfb, 0x5e, 0xfd, 0x0e, 0x00, 0x00, 0xff, 0xff, 0x9c, 0xfa, 0x8d, 0x9e, 0x43,
	0x05, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// WalletUnlockerClient is the client API for WalletUnlocker service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type WalletUnlockerClient interface {
	//*
	//GenSeed is the first method that should be used to instantiate a new lnd
	//instance. This method allows a caller to generate a new aezeed cipher seed
	//given an optional passphrase. If provided, the passphrase will be necessary
	//to decrypt the cipherseed to expose the internal wallet seed.
	//
	//Once the cipherseed is obtained and verified by the user, the InitWallet
	//method should be used to commit the newly generated seed, and create the
	//wallet.
	GenSeed(ctx context.Context, in *GenSeedRequest, opts ...grpc.CallOption) (*GenSeedResponse, error)
	//*
	//InitWallet is used when lnd is starting up for the first time to fully
	//initialize the daemon and its internal wallet. At the very least a wallet
	//password must be provided. This will be used to encrypt sensitive material
	//on disk.
	//
	//In the case of a recovery scenario, the user can also specify their aezeed
	//mnemonic and passphrase. If set, then the daemon will use this prior state
	//to initialize its internal wallet.
	//
	//Alternatively, this can be used along with the GenSeed RPC to obtain a
	//seed, then present it to the user. Once it has been verified by the user,
	//the seed can be fed into this RPC in order to commit the new wallet.
	InitWallet(ctx context.Context, in *InitWalletRequest, opts ...grpc.CallOption) (*InitWalletResponse, error)
	//* lncli: `unlock`
	//UnlockWallet is used at startup of lnd to provide a password to unlock
	//the wallet database.
	UnlockWallet(ctx context.Context, in *UnlockWalletRequest, opts ...grpc.CallOption) (*UnlockWalletResponse, error)
	//* lncli: `changepassword`
	//ChangePassword changes the password of the encrypted wallet. This will
	//automatically unlock the wallet database if successful.
	ChangePassword(ctx context.Context, in *ChangePasswordRequest, opts ...grpc.CallOption) (*ChangePasswordResponse, error)
}

type walletUnlockerClient struct {
	cc *grpc.ClientConn
}

func NewWalletUnlockerClient(cc *grpc.ClientConn) WalletUnlockerClient {
	return &walletUnlockerClient{cc}
}

func (c *walletUnlockerClient) GenSeed(ctx context.Context, in *GenSeedRequest, opts ...grpc.CallOption) (*GenSeedResponse, error) {
	out := new(GenSeedResponse)
	err := c.cc.Invoke(ctx, "/lnrpc.WalletUnlocker/GenSeed", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *walletUnlockerClient) InitWallet(ctx context.Context, in *InitWalletRequest, opts ...grpc.CallOption) (*InitWalletResponse, error) {
	out := new(InitWalletResponse)
	err := c.cc.Invoke(ctx, "/lnrpc.WalletUnlocker/InitWallet", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *walletUnlockerClient) UnlockWallet(ctx context.Context, in *UnlockWalletRequest, opts ...grpc.CallOption) (*UnlockWalletResponse, error) {
	out := new(UnlockWalletResponse)
	err := c.cc.Invoke(ctx, "/lnrpc.WalletUnlocker/UnlockWallet", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *walletUnlockerClient) ChangePassword(ctx context.Context, in *ChangePasswordRequest, opts ...grpc.CallOption) (*ChangePasswordResponse, error) {
	out := new(ChangePasswordResponse)
	err := c.cc.Invoke(ctx, "/lnrpc.WalletUnlocker/ChangePassword", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// WalletUnlockerServer is the server API for WalletUnlocker service.
type WalletUnlockerServer interface {
	//*
	//GenSeed is the first method that should be used to instantiate a new lnd
	//instance. This method allows a caller to generate a new aezeed cipher seed
	//given an optional passphrase. If provided, the passphrase will be necessary
	//to decrypt the cipherseed to expose the internal wallet seed.
	//
	//Once the cipherseed is obtained and verified by the user, the InitWallet
	//method should be used to commit the newly generated seed, and create the
	//wallet.
	GenSeed(context.Context, *GenSeedRequest) (*GenSeedResponse, error)
	//*
	//InitWallet is used when lnd is starting up for the first time to fully
	//initialize the daemon and its internal wallet. At the very least a wallet
	//password must be provided. This will be used to encrypt sensitive material
	//on disk.
	//
	//In the case of a recovery scenario, the user can also specify their aezeed
	//mnemonic and passphrase. If set, then the daemon will use this prior state
	//to initialize its internal wallet.
	//
	//Alternatively, this can be used along with the GenSeed RPC to obtain a
	//seed, then present it to the user. Once it has been verified by the user,
	//the seed can be fed into this RPC in order to commit the new wallet.
	InitWallet(context.Context, *InitWalletRequest) (*InitWalletResponse, error)
	//* lncli: `unlock`
	//UnlockWallet is used at startup of lnd to provide a password to unlock
	//the wallet database.
	UnlockWallet(context.Context, *UnlockWalletRequest) (*UnlockWalletResponse, error)
	//* lncli: `changepassword`
	//ChangePassword changes the password of the encrypted wallet. This will
	//automatically unlock the wallet database if successful.
	ChangePassword(context.Context, *ChangePasswordRequest) (*ChangePasswordResponse, error)
}

func RegisterWalletUnlockerServer(s *grpc.Server, srv WalletUnlockerServer) {
	s.RegisterService(&_WalletUnlocker_serviceDesc, srv)
}

func _WalletUnlocker_GenSeed_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GenSeedRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WalletUnlockerServer).GenSeed(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/lnrpc.WalletUnlocker/GenSeed",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WalletUnlockerServer).GenSeed(ctx, req.(*GenSeedRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WalletUnlocker_InitWallet_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(InitWalletRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WalletUnlockerServer).InitWallet(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/lnrpc.WalletUnlocker/InitWallet",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WalletUnlockerServer).InitWallet(ctx, req.(*InitWalletRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WalletUnlocker_UnlockWallet_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UnlockWalletRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WalletUnlockerServer).UnlockWallet(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/lnrpc.WalletUnlocker/UnlockWallet",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WalletUnlockerServer).UnlockWallet(ctx, req.(*UnlockWalletRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WalletUnlocker_ChangePassword_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ChangePasswordRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WalletUnlockerServer).ChangePassword(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/lnrpc.WalletUnlocker/ChangePassword",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WalletUnlockerServer).ChangePassword(ctx, req.(*ChangePasswordRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _WalletUnlocker_serviceDesc = grpc.ServiceDesc{
	ServiceName: "lnrpc.WalletUnlocker",
	HandlerType: (*WalletUnlockerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GenSeed",
			Handler:    _WalletUnlocker_GenSeed_Handler,
		},
		{
			MethodName: "InitWallet",
			Handler:    _WalletUnlocker_InitWallet_Handler,
		},
		{
			MethodName: "UnlockWallet",
			Handler:    _WalletUnlocker_UnlockWallet_Handler,
		},
		{
			MethodName: "ChangePassword",
			Handler:    _WalletUnlocker_ChangePassword_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "walletunlocker.proto",
}
