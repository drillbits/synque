// Copyright 2019 drillbits
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/drillbits/synque"

	"github.com/google/subcommands"
	"github.com/mitchellh/go-homedir"
)

var config synque.Config

type runCmd struct {
	confdir string
	config  synque.Config
}

func (*runCmd) Name() string {
	return "run"
}

func (*runCmd) Synopsis() string {
	return "run synque server."
}

func (*runCmd) Usage() string {
	return `run [-config] <config dir>:
  Run synque server.
`
}

func (cmd *runCmd) SetFlags(f *flag.FlagSet) {
	home, err := homedir.Dir()
	if err != nil {
		panic(err)
	}
	f.StringVar(&cmd.confdir, "config", filepath.Join(home, ".config", "synque"), "config directory")

	path := filepath.Join(cmd.confdir, "config.toml")
	config, err := synque.LoadConfig(path)
	if err != nil {
		panic(err)
	}
	cmd.config = *config
}

func (cmd *runCmd) Execute(ctx context.Context, flagset *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	client, err := synque.NewDriveClient(ctx, cmd.confdir)
	if err != nil {
		log.Printf("failed to create client: %s", err)
		return subcommands.ExitFailure
	}

	maxWorkers := cmd.config.MaxWorkerSize
	maxQueues := cmd.config.MaxQueueSize
	d := synque.NewDispatcher(client, maxWorkers, maxQueues)

	addr := cmd.config.Address
	srv := synque.NewServer(d, addr)
	log.Printf("listen %s", addr)
	go srv.ListenAndServe()

	d.Start()
	d.Wait()

	return subcommands.ExitSuccess
}

func main() {
	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(subcommands.FlagsCommand(), "")
	subcommands.Register(subcommands.CommandsCommand(), "")
	subcommands.Register(&runCmd{}, "")

	subcommands.Register(&driveListCmd{}, "")

	flag.Parse()
	ctx := context.Background()
	os.Exit(int(subcommands.Execute(ctx)))
}
