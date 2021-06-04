/**
 * Copyright 2021 IBM Corp.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package grpc_client

import (
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

const (
	errMsg = "establishing grpc-client connection failed"
)

var (
	errMsgString  = errors.New(errMsg)
	testString    = "test_endpoint"
	sockeEndpoint = &testString
)

type fakeClientConn1 struct {
	fcc1 fakeClConn1
}

type fakeClConn1 interface {
	Connect(target string, opts ...grpc.DialOption) (*(grpc.ClientConn), error)
	Close() error
}

func (gs *fakeClientConn1) Connect(target string, opts ...grpc.DialOption) (*(grpc.ClientConn), error) {
	var err error
	fakeConn := grpc.ClientConn{}
	return &fakeConn, err
}

func (gs *fakeClientConn1) Close() error {
	return nil
}

type fakeClientConn2 struct {
	fcc2 fakeClConn2
}

type fakeClConn2 interface {
	Connect(target string, opts ...grpc.DialOption) (*(grpc.ClientConn), error)
	Close() error
}

func (gs *fakeClientConn2) Connect(target string, opts ...grpc.DialOption) (*(grpc.ClientConn), error) {
	return nil, errMsgString
}

func (gs *fakeClientConn2) Close() error {
	return nil
}

func getFakeGrpcSession(gcon *grpc.ClientConn, conn ClientConn) GrpcSession {
	return &GrpcSes{
		conn: gcon,
		cc:   conn,
	}
}

func Test_NewGrpcSession_Positive(t *testing.T) {
	f := &ConnObjFactory{}
	grpcSess := f.NewGrpcSession()
	assert.NotNil(t, grpcSess)
}

var cc1 fakeClConn1 = &fakeClientConn1{}
var cc2 fakeClConn2 = &fakeClientConn2{}
var gcon = &grpc.ClientConn{}

func Test_GrpcDial_Positive(t *testing.T) {
	grSess := getFakeGrpcSession(gcon, &fakeClientConn1{fcc1: cc1})
	_, err := grSess.GrpcDial(cc1, *sockeEndpoint, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithDialer(UnixConnect))
	assert.NoError(t, err)
}

func Test_GrpcDial_Error(t *testing.T) {
	grSess := getFakeGrpcSession(gcon, &fakeClientConn2{fcc2: cc2})
	_, err := grSess.GrpcDial(cc2, *sockeEndpoint, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithDialer(UnixConnect))

	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), errMsg)
	}
}

func UnixConnect(addr string, t time.Duration) (net.Conn, error) {
	unix_addr, err := net.ResolveUnixAddr("unix", addr)
	conn, err := net.DialUnix("unix", nil, unix_addr)
	return conn, err
}
