/*
Copyright 2019 The Fission Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package spec

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"

	"github.com/fission/fission/pkg/controller/client"
	"github.com/fission/fission/pkg/fission-cli/cliwrapper/cli"
	spectypes "github.com/fission/fission/pkg/fission-cli/cmd/spec/types"
	flagkey "github.com/fission/fission/pkg/fission-cli/flag/key"
	"github.com/fission/fission/pkg/fission-cli/util"
)

type InitSubCommand struct {
	client       *client.Client
	deployConfig *spectypes.DeploymentConfig
}

func Init(input cli.Input) error {
	c, err := util.GetServer(input)
	if err != nil {
		return err
	}
	opts := InitSubCommand{
		client: c,
	}
	return opts.do(input)
}

func (opts *InitSubCommand) do(input cli.Input) error {
	err := opts.complete(input)
	if err != nil {
		return err
	}
	return opts.run(input)
}

func (opts *InitSubCommand) complete(input cli.Input) error {
	// Figure out spec directory
	specDir := util.GetSpecDir(input)

	name := input.String(flagkey.SpecName)
	if len(name) == 0 {
		// come up with a name using the current dir
		dir, err := filepath.Abs(".")
		if err != nil {
			return errors.Wrap(err, "error getting current working directory")
		}
		basename := filepath.Base(dir)
		name = util.KubifyName(basename)
	}

	deployID := input.String(flagkey.SpecDeployID)
	if len(deployID) == 0 {
		deployID = uuid.NewV4().String()
	}

	// Create spec dir
	fmt.Printf("Creating fission spec directory '%v'\n", specDir)
	err := os.MkdirAll(specDir, 0755)
	if err != nil {
		return errors.Wrapf(err, "create spec directory '%v'", specDir)
	}

	// Write the deployment config
	opts.deployConfig = &spectypes.DeploymentConfig{
		TypeMeta: spectypes.TypeMeta{
			APIVersion: SPEC_API_VERSION,
			Kind:       "DeploymentConfig",
		},
		Name: name,

		// All resources will be annotated with the UID when they're created. This allows
		// us to be idempotent, as well as to delete resources when their specs are
		// removed.
		UID: deployID,
	}
	return nil
}

// run just initializes an empty spec directory and adds some
// sample YAMLs in there that might be useful.
func (opts *InitSubCommand) run(input cli.Input) error {
	specDir := util.GetSpecDir(input)

	readme := filepath.Join(specDir, "README")
	config := filepath.Join(specDir, "fission-deployment-config.yaml")

	if _, err := os.Stat(config); err == nil {
		return errors.Errorf("Spec DeploymentConfig already exists in directory '%v'", specDir)
	}

	// Add a bit of documentation to the spec dir here
	err := ioutil.WriteFile(readme, []byte(SPEC_README), 0644)
	if err != nil {
		return err
	}

	err = writeDeploymentConfig(config, opts.deployConfig)
	if err != nil {
		return errors.Wrap(err, "error writing deployment config")
	}

	// Other possible things to do here:
	// - add example specs to the dir to make it easy to manually
	//   add new ones
	return nil
}

// writeDeploymentConfig serializes the DeploymentConfig to YAML and writes it to a new
// fission-config.yaml in specDir.
func writeDeploymentConfig(file string, dc *spectypes.DeploymentConfig) error {
	y, err := yaml.Marshal(dc)
	if err != nil {
		return err
	}

	msg := []byte("# This file is generated by the 'fission spec init' command.\n" +
		"# See the README in this directory for background and usage information.\n" +
		"# Do not edit the UID below: that will break 'fission spec apply'\n")

	err = ioutil.WriteFile(file, append(msg, y...), 0644)
	if err != nil {
		return err
	}
	return nil
}
