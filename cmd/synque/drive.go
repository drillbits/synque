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
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/drillbits/synque"
	"github.com/google/subcommands"
	"github.com/mitchellh/go-homedir"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
)

type driveListCmd struct {
	confdir string
	config  synque.Config
	parent  string
	// parents stringsFlagValue
}

func (*driveListCmd) Name() string {
	return "drivelist"
}

func (*driveListCmd) Synopsis() string {
	return "list drive files."
}

func (*driveListCmd) Usage() string {
	return `drivelist [-config] <config dir> [-p] <folder id>:
  List drive files.
`
}

func (cmd *driveListCmd) SetFlags(f *flag.FlagSet) {
	home, err := homedir.Dir()
	if err != nil {
		panic(err)
	}
	f.StringVar(&cmd.confdir, "config", filepath.Join(home, ".config", "synque"), "config directory")
	// f.Var(&cmd.parents, "p", "list of the parent folders' id which contain the file.")
	f.StringVar(&cmd.parent, "p", "root", "folder id")

	path := filepath.Join(cmd.confdir, "config.toml")
	config, err := synque.LoadConfig(path)
	if err != nil {
		panic(err)
	}
	cmd.config = *config
}

func (cmd *driveListCmd) Execute(ctx context.Context, flagset *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	client, err := synque.NewDriveClient(ctx, cmd.confdir)
	if err != nil {
		log.Printf("failed to create client: %s", err)
		return subcommands.ExitFailure
	}

	service, err := drive.New(client)
	if err != nil {
		log.Printf("failed to create drive service: %s", err)
		return subcommands.ExitFailure
	}

	// https://developers.google.com/drive/api/v3/reference/files
	// fields := "id,name,md5Checksum,mimeType,size,createdTime,parents"
	fields := "id,name,mimeType,properties,appProperties,spaces,createdTime,modifiedTime,owners,permissions,size"
	fileFields := []googleapi.Field{
		googleapi.Field(fields),
	}
	filesFields := []googleapi.Field{
		"nextPageToken",
		googleapi.Field(fmt.Sprintf("files(%s)", fields)),
	}

	f, err := service.Files.Get(cmd.parent).Fields(fileFields...).Do()
	if err != nil {
		switch e := err.(type) {
		case *googleapi.Error:
			switch e.Code {
			case 404:
				fmt.Printf("%s: %s: No such folder\n", cmd.Name(), cmd.parent)
			default:
				log.Printf("failed to get folder: %s", err)
			}
		default:
			log.Printf("failed to get folder: %s", err)
		}
		return subcommands.ExitFailure
	}

	// see parameter setting on https://developers.google.com/drive/v3/web/search-parameters#fn4
	query := fmt.Sprintf("'%s' in parents", cmd.parent)
	fl, err := service.Files.List().Q(query).Fields(filesFields...).Do()
	if err != nil {
		log.Printf("failed to list files: %s", err)
		return subcommands.ExitFailure
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 1, 1, ' ', 0)
	w.Write([]byte(fmt.Sprintf("total %d\n", len(fl.Files))))
	w.Write([]byte(stringFile(f)))
	for _, f := range fl.Files {
		w.Write([]byte(stringFile(f)))
	}
	w.Flush()

	return subcommands.ExitSuccess
}

func stringFile(f *drive.File) string {
	fileType := f.MimeType
	fileType = strings.Replace(fileType, "application/vnd.google-apps.", "", -1)

	loc := time.FixedZone("Asia/Tokyo", 9*60*60)
	d, _ := time.Parse(time.RFC3339Nano, f.ModifiedTime)
	d = d.In(loc)
	timestamp := d.Format("Jan _2 15:04")
	return fmt.Sprintf("%s\t%s\t%d\t%s\t%s\n", f.Id, fileType, f.Size, timestamp, f.Name)
}
