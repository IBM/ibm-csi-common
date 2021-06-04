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

package fake_grpc

import (
	"errors"

	grpcClient "github.com/IBM/ibm-csi-common/pkg/utils/grpc-client"
	"google.golang.org/grpc"
)

//FakeGrpcSessionFactory implements grpcClient.GrpcSessionFactory
type FakeGrpcSessionFactory struct {
	//FailGrpcConnection ...
	FailGrpcConnection bool
	//FailGrpcConnectionErr with specific error msg...
	FailGrpcConnectionErr string
	//PassGrpcConnection ...
	PassGrpcConnection bool
}

var _ grpcClient.GrpcSessionFactory = (*FakeGrpcSessionFactory)(nil)

//fakeGrpcSession implements grpcClient.GrpcSession
type fakeGrpcSession struct {
	factory *FakeGrpcSessionFactory
}

// NewGrpcSession method creates a new fakeGrpcSession session
func (f *FakeGrpcSessionFactory) NewGrpcSession() grpcClient.GrpcSession {
	return &fakeGrpcSession{
		factory: f,
	}
}

// GrpcDial method creates a fake-grpc-client connection
func (c *fakeGrpcSession) GrpcDial(clientConn grpcClient.ClientConn, target string, opts ...grpc.DialOption) (conn *grpc.ClientConn, err error) {
	if c.factory.FailGrpcConnection {
		return conn, errors.New(c.factory.FailGrpcConnectionErr)
	}
	return conn, err
}
