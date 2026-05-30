// Code generated manually (proto definition in auth.proto)

package auth

import (
	"google.golang.org/protobuf/runtime/protoimpl"
)

type VerifyRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Token string `protobuf:"bytes,1,opt,name=token,proto3" json:"token,omitempty"`
}

func (x *VerifyRequest) Reset() {
	*x = VerifyRequest{}
}

func (x *VerifyRequest) String() string {
	return x.Token
}

func (*VerifyRequest) ProtoMessage() {}

func (x *VerifyRequest) GetToken() string {
	if x != nil {
		return x.Token
	}
	return ""
}

type VerifyResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Valid   bool   `protobuf:"varint,1,opt,name=valid,proto3" json:"valid,omitempty"`
	Subject string `protobuf:"bytes,2,opt,name=subject,proto3" json:"subject,omitempty"`
	Error   string `protobuf:"bytes,3,opt,name=error,proto3" json:"error,omitempty"`
}

func (x *VerifyResponse) Reset() {
	*x = VerifyResponse{}
}

func (x *VerifyResponse) String() string {
	return x.Subject
}

func (*VerifyResponse) ProtoMessage() {}

func (x *VerifyResponse) GetValid() bool {
	if x != nil {
		return x.Valid
	}
	return false
}

func (x *VerifyResponse) GetSubject() string {
	if x != nil {
		return x.Subject
	}
	return ""
}

func (x *VerifyResponse) GetError() string {
	if x != nil {
		return x.Error
	}
	return ""
}
