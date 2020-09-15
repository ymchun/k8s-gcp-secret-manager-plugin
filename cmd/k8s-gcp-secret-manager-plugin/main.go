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

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/golang/glog"
	"github.com/ymchun/k8s-gcp-secret-manager-plugin/internal/plugin"
)

var (
	credentialsFile  = flag.String("credentials", "", "Path to GCP Secret Manager service account credentials JSON file")
	masterKeyURI     = flag.String("key-uri", "", "Resource ID of the secret in the format `projects/*/secrets/*/versions/*`")
	pathToUnixSocket = flag.String("unix-socket", "/var/run/k8splugin.sock", "Full path to Unix socket that is used for communicating with KubeAPI Server, or Linux socket namespace object - must start with @")
)

func main() {
	flag.Parse()
	mustValidateFlags()

	plugin := &plugin.Plugin{
		CredentialsFile:  *credentialsFile,
		MasterKeyURI:     *masterKeyURI,
		PathToUnixSocket: *pathToUnixSocket,
	}
	err := plugin.Init()

	if err != nil {
		glog.Exitf("failed to instantiate secret manager client: %v", err)
	}

	glog.Exit(run(plugin))
}

func run(p *plugin.Plugin) error {
	signalsChan := make(chan os.Signal, 1)
	signal.Notify(signalsChan, syscall.SIGINT, syscall.SIGTERM)

	gRPCSrv, kmsErrorChan := p.ServeKMSRequests()
	defer gRPCSrv.GracefulStop()

	for {
		select {
		case sig := <-signalsChan:
			return fmt.Errorf("Captured %v, shutting down kms-plugin", sig)
		case kmsError := <-kmsErrorChan:
			return kmsError
		}
	}
}

func mustValidateFlags() {
	socketDir := filepath.Dir(*pathToUnixSocket)
	if _, err := os.Stat(socketDir); err != nil {
		glog.Exitf(" Directory %q portion of path-to-unix-socket flag:%q does not seem to exist.", socketDir, *pathToUnixSocket)
	}
	glog.Infof("Communication between KUBE API and Secret Manager Plugin will be via %q", *pathToUnixSocket)
}
