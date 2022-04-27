// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package signrpc

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

// SignerClient is the client API for Signer service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type SignerClient interface {
	//
	//SignOutputRaw is a method that can be used to generated a signature for a
	//set of inputs/outputs to a transaction. Each request specifies details
	//concerning how the outputs should be signed, which keys they should be
	//signed with, and also any optional tweaks. The return value is a fixed
	//64-byte signature (the same format as we use on the wire in Lightning).
	//
	//If we are  unable to sign using the specified keys, then an error will be
	//returned.
	SignOutputRaw(ctx context.Context, in *SignReq, opts ...grpc.CallOption) (*SignResp, error)
	//
	//ComputeInputScript generates a complete InputIndex for the passed
	//transaction with the signature as defined within the passed SignDescriptor.
	//This method should be capable of generating the proper input script for
	//both regular p2wkh output and p2wkh outputs nested within a regular p2sh
	//output.
	//
	//Note that when using this method to sign inputs belonging to the wallet,
	//the only items of the SignDescriptor that need to be populated are pkScript
	//in the TxOut field, the value in that same field, and finally the input
	//index.
	ComputeInputScript(ctx context.Context, in *SignReq, opts ...grpc.CallOption) (*InputScriptResp, error)
	//
	//SignMessage signs a message with the key specified in the key locator. The
	//returned signature is fixed-size LN wire format encoded.
	//
	//The main difference to SignMessage in the main RPC is that a specific key is
	//used to sign the message instead of the node identity private key.
	SignMessage(ctx context.Context, in *SignMessageReq, opts ...grpc.CallOption) (*SignMessageResp, error)
	//
	//VerifyMessage verifies a signature over a message using the public key
	//provided. The signature must be fixed-size LN wire format encoded.
	//
	//The main difference to VerifyMessage in the main RPC is that the public key
	//used to sign the message does not have to be a node known to the network.
	VerifyMessage(ctx context.Context, in *VerifyMessageReq, opts ...grpc.CallOption) (*VerifyMessageResp, error)
	//
	//DeriveSharedKey returns a shared secret key by performing Diffie-Hellman key
	//derivation between the ephemeral public key in the request and the node's
	//key specified in the key_desc parameter. Either a key locator or a raw
	//public key is expected in the key_desc, if neither is supplied, defaults to
	//the node's identity private key:
	//P_shared = privKeyNode * ephemeralPubkey
	//The resulting shared public key is serialized in the compressed format and
	//hashed with sha256, resulting in the final key length of 256bit.
	DeriveSharedKey(ctx context.Context, in *SharedKeyRequest, opts ...grpc.CallOption) (*SharedKeyResponse, error)
	//
	//MuSig2CombineKeys (experimental!) is a stateless helper RPC that can be used
	//to calculate the combined MuSig2 public key from a list of all participating
	//signers' public keys. This RPC is completely stateless and deterministic and
	//does not create any signing session. It can be used to determine the Taproot
	//public key that should be put in an on-chain output once all public keys are
	//known. A signing session is only needed later when that output should be
	//_spent_ again.
	//
	//NOTE: The MuSig2 BIP is not final yet and therefore this API must be
	//considered to be HIGHLY EXPERIMENTAL and subject to change in upcoming
	//releases. Backward compatibility is not guaranteed!
	MuSig2CombineKeys(ctx context.Context, in *MuSig2CombineKeysRequest, opts ...grpc.CallOption) (*MuSig2CombineKeysResponse, error)
	//
	//MuSig2CreateSession (experimental!) creates a new MuSig2 signing session
	//using the local key identified by the key locator. The complete list of all
	//public keys of all signing parties must be provided, including the public
	//key of the local signing key. If nonces of other parties are already known,
	//they can be submitted as well to reduce the number of RPC calls necessary
	//later on.
	//
	//NOTE: The MuSig2 BIP is not final yet and therefore this API must be
	//considered to be HIGHLY EXPERIMENTAL and subject to change in upcoming
	//releases. Backward compatibility is not guaranteed!
	MuSig2CreateSession(ctx context.Context, in *MuSig2SessionRequest, opts ...grpc.CallOption) (*MuSig2SessionResponse, error)
	//
	//MuSig2RegisterNonces (experimental!) registers one or more public nonces of
	//other signing participants for a session identified by its ID. This RPC can
	//be called multiple times until all nonces are registered.
	//
	//NOTE: The MuSig2 BIP is not final yet and therefore this API must be
	//considered to be HIGHLY EXPERIMENTAL and subject to change in upcoming
	//releases. Backward compatibility is not guaranteed!
	MuSig2RegisterNonces(ctx context.Context, in *MuSig2RegisterNoncesRequest, opts ...grpc.CallOption) (*MuSig2RegisterNoncesResponse, error)
	//
	//MuSig2Sign (experimental!) creates a partial signature using the local
	//signing key that was specified when the session was created. This can only
	//be called when all public nonces of all participants are known and have been
	//registered with the session. If this node isn't responsible for combining
	//all the partial signatures, then the cleanup flag should be set, indicating
	//that the session can be removed from memory once the signature was produced.
	//
	//NOTE: The MuSig2 BIP is not final yet and therefore this API must be
	//considered to be HIGHLY EXPERIMENTAL and subject to change in upcoming
	//releases. Backward compatibility is not guaranteed!
	MuSig2Sign(ctx context.Context, in *MuSig2SignRequest, opts ...grpc.CallOption) (*MuSig2SignResponse, error)
	//
	//MuSig2CombineSig (experimental!) combines the given partial signature(s)
	//with the local one, if it already exists. Once a partial signature of all
	//participants is registered, the final signature will be combined and
	//returned.
	//
	//NOTE: The MuSig2 BIP is not final yet and therefore this API must be
	//considered to be HIGHLY EXPERIMENTAL and subject to change in upcoming
	//releases. Backward compatibility is not guaranteed!
	MuSig2CombineSig(ctx context.Context, in *MuSig2CombineSigRequest, opts ...grpc.CallOption) (*MuSig2CombineSigResponse, error)
}

type signerClient struct {
	cc grpc.ClientConnInterface
}

func NewSignerClient(cc grpc.ClientConnInterface) SignerClient {
	return &signerClient{cc}
}

func (c *signerClient) SignOutputRaw(ctx context.Context, in *SignReq, opts ...grpc.CallOption) (*SignResp, error) {
	out := new(SignResp)
	err := c.cc.Invoke(ctx, "/signrpc.Signer/SignOutputRaw", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *signerClient) ComputeInputScript(ctx context.Context, in *SignReq, opts ...grpc.CallOption) (*InputScriptResp, error) {
	out := new(InputScriptResp)
	err := c.cc.Invoke(ctx, "/signrpc.Signer/ComputeInputScript", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *signerClient) SignMessage(ctx context.Context, in *SignMessageReq, opts ...grpc.CallOption) (*SignMessageResp, error) {
	out := new(SignMessageResp)
	err := c.cc.Invoke(ctx, "/signrpc.Signer/SignMessage", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *signerClient) VerifyMessage(ctx context.Context, in *VerifyMessageReq, opts ...grpc.CallOption) (*VerifyMessageResp, error) {
	out := new(VerifyMessageResp)
	err := c.cc.Invoke(ctx, "/signrpc.Signer/VerifyMessage", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *signerClient) DeriveSharedKey(ctx context.Context, in *SharedKeyRequest, opts ...grpc.CallOption) (*SharedKeyResponse, error) {
	out := new(SharedKeyResponse)
	err := c.cc.Invoke(ctx, "/signrpc.Signer/DeriveSharedKey", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *signerClient) MuSig2CombineKeys(ctx context.Context, in *MuSig2CombineKeysRequest, opts ...grpc.CallOption) (*MuSig2CombineKeysResponse, error) {
	out := new(MuSig2CombineKeysResponse)
	err := c.cc.Invoke(ctx, "/signrpc.Signer/MuSig2CombineKeys", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *signerClient) MuSig2CreateSession(ctx context.Context, in *MuSig2SessionRequest, opts ...grpc.CallOption) (*MuSig2SessionResponse, error) {
	out := new(MuSig2SessionResponse)
	err := c.cc.Invoke(ctx, "/signrpc.Signer/MuSig2CreateSession", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *signerClient) MuSig2RegisterNonces(ctx context.Context, in *MuSig2RegisterNoncesRequest, opts ...grpc.CallOption) (*MuSig2RegisterNoncesResponse, error) {
	out := new(MuSig2RegisterNoncesResponse)
	err := c.cc.Invoke(ctx, "/signrpc.Signer/MuSig2RegisterNonces", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *signerClient) MuSig2Sign(ctx context.Context, in *MuSig2SignRequest, opts ...grpc.CallOption) (*MuSig2SignResponse, error) {
	out := new(MuSig2SignResponse)
	err := c.cc.Invoke(ctx, "/signrpc.Signer/MuSig2Sign", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *signerClient) MuSig2CombineSig(ctx context.Context, in *MuSig2CombineSigRequest, opts ...grpc.CallOption) (*MuSig2CombineSigResponse, error) {
	out := new(MuSig2CombineSigResponse)
	err := c.cc.Invoke(ctx, "/signrpc.Signer/MuSig2CombineSig", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// SignerServer is the server API for Signer service.
// All implementations must embed UnimplementedSignerServer
// for forward compatibility
type SignerServer interface {
	//
	//SignOutputRaw is a method that can be used to generated a signature for a
	//set of inputs/outputs to a transaction. Each request specifies details
	//concerning how the outputs should be signed, which keys they should be
	//signed with, and also any optional tweaks. The return value is a fixed
	//64-byte signature (the same format as we use on the wire in Lightning).
	//
	//If we are  unable to sign using the specified keys, then an error will be
	//returned.
	SignOutputRaw(context.Context, *SignReq) (*SignResp, error)
	//
	//ComputeInputScript generates a complete InputIndex for the passed
	//transaction with the signature as defined within the passed SignDescriptor.
	//This method should be capable of generating the proper input script for
	//both regular p2wkh output and p2wkh outputs nested within a regular p2sh
	//output.
	//
	//Note that when using this method to sign inputs belonging to the wallet,
	//the only items of the SignDescriptor that need to be populated are pkScript
	//in the TxOut field, the value in that same field, and finally the input
	//index.
	ComputeInputScript(context.Context, *SignReq) (*InputScriptResp, error)
	//
	//SignMessage signs a message with the key specified in the key locator. The
	//returned signature is fixed-size LN wire format encoded.
	//
	//The main difference to SignMessage in the main RPC is that a specific key is
	//used to sign the message instead of the node identity private key.
	SignMessage(context.Context, *SignMessageReq) (*SignMessageResp, error)
	//
	//VerifyMessage verifies a signature over a message using the public key
	//provided. The signature must be fixed-size LN wire format encoded.
	//
	//The main difference to VerifyMessage in the main RPC is that the public key
	//used to sign the message does not have to be a node known to the network.
	VerifyMessage(context.Context, *VerifyMessageReq) (*VerifyMessageResp, error)
	//
	//DeriveSharedKey returns a shared secret key by performing Diffie-Hellman key
	//derivation between the ephemeral public key in the request and the node's
	//key specified in the key_desc parameter. Either a key locator or a raw
	//public key is expected in the key_desc, if neither is supplied, defaults to
	//the node's identity private key:
	//P_shared = privKeyNode * ephemeralPubkey
	//The resulting shared public key is serialized in the compressed format and
	//hashed with sha256, resulting in the final key length of 256bit.
	DeriveSharedKey(context.Context, *SharedKeyRequest) (*SharedKeyResponse, error)
	//
	//MuSig2CombineKeys (experimental!) is a stateless helper RPC that can be used
	//to calculate the combined MuSig2 public key from a list of all participating
	//signers' public keys. This RPC is completely stateless and deterministic and
	//does not create any signing session. It can be used to determine the Taproot
	//public key that should be put in an on-chain output once all public keys are
	//known. A signing session is only needed later when that output should be
	//_spent_ again.
	//
	//NOTE: The MuSig2 BIP is not final yet and therefore this API must be
	//considered to be HIGHLY EXPERIMENTAL and subject to change in upcoming
	//releases. Backward compatibility is not guaranteed!
	MuSig2CombineKeys(context.Context, *MuSig2CombineKeysRequest) (*MuSig2CombineKeysResponse, error)
	//
	//MuSig2CreateSession (experimental!) creates a new MuSig2 signing session
	//using the local key identified by the key locator. The complete list of all
	//public keys of all signing parties must be provided, including the public
	//key of the local signing key. If nonces of other parties are already known,
	//they can be submitted as well to reduce the number of RPC calls necessary
	//later on.
	//
	//NOTE: The MuSig2 BIP is not final yet and therefore this API must be
	//considered to be HIGHLY EXPERIMENTAL and subject to change in upcoming
	//releases. Backward compatibility is not guaranteed!
	MuSig2CreateSession(context.Context, *MuSig2SessionRequest) (*MuSig2SessionResponse, error)
	//
	//MuSig2RegisterNonces (experimental!) registers one or more public nonces of
	//other signing participants for a session identified by its ID. This RPC can
	//be called multiple times until all nonces are registered.
	//
	//NOTE: The MuSig2 BIP is not final yet and therefore this API must be
	//considered to be HIGHLY EXPERIMENTAL and subject to change in upcoming
	//releases. Backward compatibility is not guaranteed!
	MuSig2RegisterNonces(context.Context, *MuSig2RegisterNoncesRequest) (*MuSig2RegisterNoncesResponse, error)
	//
	//MuSig2Sign (experimental!) creates a partial signature using the local
	//signing key that was specified when the session was created. This can only
	//be called when all public nonces of all participants are known and have been
	//registered with the session. If this node isn't responsible for combining
	//all the partial signatures, then the cleanup flag should be set, indicating
	//that the session can be removed from memory once the signature was produced.
	//
	//NOTE: The MuSig2 BIP is not final yet and therefore this API must be
	//considered to be HIGHLY EXPERIMENTAL and subject to change in upcoming
	//releases. Backward compatibility is not guaranteed!
	MuSig2Sign(context.Context, *MuSig2SignRequest) (*MuSig2SignResponse, error)
	//
	//MuSig2CombineSig (experimental!) combines the given partial signature(s)
	//with the local one, if it already exists. Once a partial signature of all
	//participants is registered, the final signature will be combined and
	//returned.
	//
	//NOTE: The MuSig2 BIP is not final yet and therefore this API must be
	//considered to be HIGHLY EXPERIMENTAL and subject to change in upcoming
	//releases. Backward compatibility is not guaranteed!
	MuSig2CombineSig(context.Context, *MuSig2CombineSigRequest) (*MuSig2CombineSigResponse, error)
	mustEmbedUnimplementedSignerServer()
}

// UnimplementedSignerServer must be embedded to have forward compatible implementations.
type UnimplementedSignerServer struct {
}

func (UnimplementedSignerServer) SignOutputRaw(context.Context, *SignReq) (*SignResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SignOutputRaw not implemented")
}
func (UnimplementedSignerServer) ComputeInputScript(context.Context, *SignReq) (*InputScriptResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ComputeInputScript not implemented")
}
func (UnimplementedSignerServer) SignMessage(context.Context, *SignMessageReq) (*SignMessageResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SignMessage not implemented")
}
func (UnimplementedSignerServer) VerifyMessage(context.Context, *VerifyMessageReq) (*VerifyMessageResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method VerifyMessage not implemented")
}
func (UnimplementedSignerServer) DeriveSharedKey(context.Context, *SharedKeyRequest) (*SharedKeyResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeriveSharedKey not implemented")
}
func (UnimplementedSignerServer) MuSig2CombineKeys(context.Context, *MuSig2CombineKeysRequest) (*MuSig2CombineKeysResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method MuSig2CombineKeys not implemented")
}
func (UnimplementedSignerServer) MuSig2CreateSession(context.Context, *MuSig2SessionRequest) (*MuSig2SessionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method MuSig2CreateSession not implemented")
}
func (UnimplementedSignerServer) MuSig2RegisterNonces(context.Context, *MuSig2RegisterNoncesRequest) (*MuSig2RegisterNoncesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method MuSig2RegisterNonces not implemented")
}
func (UnimplementedSignerServer) MuSig2Sign(context.Context, *MuSig2SignRequest) (*MuSig2SignResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method MuSig2Sign not implemented")
}
func (UnimplementedSignerServer) MuSig2CombineSig(context.Context, *MuSig2CombineSigRequest) (*MuSig2CombineSigResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method MuSig2CombineSig not implemented")
}
func (UnimplementedSignerServer) mustEmbedUnimplementedSignerServer() {}

// UnsafeSignerServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to SignerServer will
// result in compilation errors.
type UnsafeSignerServer interface {
	mustEmbedUnimplementedSignerServer()
}

func RegisterSignerServer(s grpc.ServiceRegistrar, srv SignerServer) {
	s.RegisterService(&Signer_ServiceDesc, srv)
}

func _Signer_SignOutputRaw_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SignReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SignerServer).SignOutputRaw(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/signrpc.Signer/SignOutputRaw",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SignerServer).SignOutputRaw(ctx, req.(*SignReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _Signer_ComputeInputScript_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SignReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SignerServer).ComputeInputScript(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/signrpc.Signer/ComputeInputScript",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SignerServer).ComputeInputScript(ctx, req.(*SignReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _Signer_SignMessage_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SignMessageReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SignerServer).SignMessage(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/signrpc.Signer/SignMessage",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SignerServer).SignMessage(ctx, req.(*SignMessageReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _Signer_VerifyMessage_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(VerifyMessageReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SignerServer).VerifyMessage(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/signrpc.Signer/VerifyMessage",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SignerServer).VerifyMessage(ctx, req.(*VerifyMessageReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _Signer_DeriveSharedKey_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SharedKeyRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SignerServer).DeriveSharedKey(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/signrpc.Signer/DeriveSharedKey",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SignerServer).DeriveSharedKey(ctx, req.(*SharedKeyRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Signer_MuSig2CombineKeys_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MuSig2CombineKeysRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SignerServer).MuSig2CombineKeys(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/signrpc.Signer/MuSig2CombineKeys",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SignerServer).MuSig2CombineKeys(ctx, req.(*MuSig2CombineKeysRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Signer_MuSig2CreateSession_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MuSig2SessionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SignerServer).MuSig2CreateSession(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/signrpc.Signer/MuSig2CreateSession",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SignerServer).MuSig2CreateSession(ctx, req.(*MuSig2SessionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Signer_MuSig2RegisterNonces_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MuSig2RegisterNoncesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SignerServer).MuSig2RegisterNonces(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/signrpc.Signer/MuSig2RegisterNonces",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SignerServer).MuSig2RegisterNonces(ctx, req.(*MuSig2RegisterNoncesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Signer_MuSig2Sign_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MuSig2SignRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SignerServer).MuSig2Sign(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/signrpc.Signer/MuSig2Sign",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SignerServer).MuSig2Sign(ctx, req.(*MuSig2SignRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Signer_MuSig2CombineSig_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MuSig2CombineSigRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SignerServer).MuSig2CombineSig(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/signrpc.Signer/MuSig2CombineSig",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SignerServer).MuSig2CombineSig(ctx, req.(*MuSig2CombineSigRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Signer_ServiceDesc is the grpc.ServiceDesc for Signer service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Signer_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "signrpc.Signer",
	HandlerType: (*SignerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "SignOutputRaw",
			Handler:    _Signer_SignOutputRaw_Handler,
		},
		{
			MethodName: "ComputeInputScript",
			Handler:    _Signer_ComputeInputScript_Handler,
		},
		{
			MethodName: "SignMessage",
			Handler:    _Signer_SignMessage_Handler,
		},
		{
			MethodName: "VerifyMessage",
			Handler:    _Signer_VerifyMessage_Handler,
		},
		{
			MethodName: "DeriveSharedKey",
			Handler:    _Signer_DeriveSharedKey_Handler,
		},
		{
			MethodName: "MuSig2CombineKeys",
			Handler:    _Signer_MuSig2CombineKeys_Handler,
		},
		{
			MethodName: "MuSig2CreateSession",
			Handler:    _Signer_MuSig2CreateSession_Handler,
		},
		{
			MethodName: "MuSig2RegisterNonces",
			Handler:    _Signer_MuSig2RegisterNonces_Handler,
		},
		{
			MethodName: "MuSig2Sign",
			Handler:    _Signer_MuSig2Sign_Handler,
		},
		{
			MethodName: "MuSig2CombineSig",
			Handler:    _Signer_MuSig2CombineSig_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "signrpc/signer.proto",
}
