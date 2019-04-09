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

package synque

import (
	"github.com/BurntSushi/toml"
)

type Config struct {
	Address       string `toml:"addr"`
	MaxWorkerSize int    `toml:"max_worker_size"`
	MaxQueueSize  int    `toml:"max_queue_size"`
}

func LoadConfig(path string) (*Config, error) {
	var config Config
	if _, err := toml.DecodeFile(path, &config); err != nil {
		return nil, err
	}

	if config.Address == "" {
		config.Address = ":5119"
	}

	if config.MaxWorkerSize <= 0 {
		config.MaxWorkerSize = 1
	}

	if config.MaxQueueSize <= 0 {
		config.MaxQueueSize = 100
	}

	return &config, nil
}
