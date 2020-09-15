// Copyright 2019 Google, LLC.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//      http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package plugin

import (
	"context"
	"net"
	"os"
	"strings"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"github.com/golang/glog"
	"github.com/ymchun/k8s-gcp-secret-manager-plugin/internal/secret"
	"google.golang.org/api/option"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
	"google.golang.org/grpc"
)

const (
	netProtocol    = "unix"
	apiVersion     = "v1beta1"
	runtimeName    = "GCP Secret Manager"
	runtimeVersion = "0.0.1"
)

// Plugin is a GCP Secret Manager plugin for K8S.
type Plugin struct {
	CredentialsFile  string
	MasterKeyURI     string
	PathToUnixSocket string

	client *secretmanager.Client

	// Embedding these only to shorten access to fields.
	net.Listener
	*grpc.Server
}

func (g *Plugin) Init() error {
	client, err := secretmanager.NewClient(
		context.Background(),
		option.WithCredentialsFile(g.CredentialsFile),
	)

	if err != nil {
		glog.Errorf("Failed to init secret manager client: %v\n", err)
		return err
	}

	g.client = client

	return nil
}

// Version returns the version of KMS Plugin.
func (g *Plugin) Version(ctx context.Context, request *VersionRequest) (*VersionResponse, error) {
	return &VersionResponse{Version: apiVersion, RuntimeName: runtimeName, RuntimeVersion: runtimeVersion}, nil
}

// Encrypt encrypts payload provided by K8S API Server.
func (g *Plugin) Encrypt(ctx context.Context, request *EncryptRequest) (*EncryptResponse, error) {
	glog.Infoln("Processing request for encryption.")

	// get the master key from gcp secret manager
	resp, err := g.client.AccessSecretVersion(
		context.Background(),
		&secretmanagerpb.AccessSecretVersionRequest{
			Name: g.MasterKeyURI,
		},
	)
	if err != nil {
		glog.Errorf("Failed to access secret: %v\n", err)
		return nil, err
	}

	// create secret object for encryption
	secret := &secret.Secret{
		Key: resp.Payload.Data,
	}
	ciphertext, err := secret.Encrypt(request.Plain)

	if err != nil {
		glog.Errorf("Failed to encrypt request: %v\n", err)
		return nil, err
	}

	// remove the key from RAM
	secret.Destroy()

	return &EncryptResponse{Cipher: ciphertext}, nil
}

// Decrypt decrypts payload supplied by K8S API Server.
func (g *Plugin) Decrypt(ctx context.Context, request *DecryptRequest) (*DecryptResponse, error) {
	glog.Infoln("Processing request for decryption.")

	// get the master key from gcp secret manager
	resp, err := g.client.AccessSecretVersion(
		context.Background(),
		&secretmanagerpb.AccessSecretVersionRequest{
			Name: g.MasterKeyURI,
		},
	)
	if err != nil {
		glog.Errorf("Failed to access secret: %v\n", err)
		return nil, err
	}

	// create secret object for decryption
	secret := &secret.Secret{
		Key: resp.Payload.Data,
	}

	plaintext, err := secret.Decrypt(request.Cipher)

	if err != nil {
		glog.Errorf("Failed to decrypt request: %v\n", err)
		return nil, err
	}

	// remove the key from RAM
	secret.Destroy()

	return &DecryptResponse{Plain: plaintext}, nil
}

func (g *Plugin) setupRPCServer() error {
	err := g.cleanSockFile()

	if err != nil {
		return err
	}

	listener, err := net.Listen(netProtocol, g.PathToUnixSocket)

	if err != nil {
		glog.Errorf("Failed to start listener, error: %v", err)
		return err
	}

	g.Listener = listener
	g.Server = grpc.NewServer()
	RegisterKeyManagementServiceServer(g.Server, g)

	glog.Infof("Listening on unix domain socket: %s", g.PathToUnixSocket)

	return nil
}

// ServeKMSRequests starts gRPC server or dies.
func (g *Plugin) ServeKMSRequests() (*grpc.Server, chan error) {
	errorChan := make(chan error, 1)
	err := g.setupRPCServer()

	if err != nil {
		errorChan <- err
		close(errorChan)
		return nil, errorChan
	}

	go func() {
		defer close(errorChan)
		errorChan <- g.Serve(g.Listener)
	}()

	return g.Server, errorChan
}

func (g *Plugin) cleanSockFile() error {
	// @ implies the use of Linux socket namespace - no file on disk and nothing to clean-up.
	if strings.HasPrefix(g.PathToUnixSocket, "@") {
		return nil
	}

	err := os.Remove(g.PathToUnixSocket)

	if err != nil && !os.IsNotExist(err) {
		glog.Errorf("failed to delete the socket file, error: %v", err)
		return err
	}
	return nil
}
