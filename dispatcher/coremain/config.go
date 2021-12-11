//     Copyright (C) 2020-2021, IrineSistiana
//
//     This file is part of mosdns.
//
//     mosdns is free software: you can redistribute it and/or modify
//     it under the terms of the GNU General Public License as published by
//     the Free Software Foundation, either version 3 of the License, or
//     (at your option) any later version.
//
//     mosdns is distributed in the hope that it will be useful,
//     but WITHOUT ANY WARRANTY; without even the implied warranty of
//     MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//     GNU General Public License for more details.
//
//     You should have received a copy of the GNU General Public License
//     along with this program.  If not, see <https://www.gnu.org/licenses/>.

package coremain

import (
	"bytes"
	"context"
	"fmt"
	"github.com/IrineSistiana/mosdns/v2/dispatcher/handler"
	"github.com/IrineSistiana/mosdns/v2/dispatcher/pkg/ext_exec"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"regexp"
	"time"
)

// Config is config
type Config struct {
	Log struct {
		Level string `yaml:"level"`
		File  string `yaml:"file"`
	} `yaml:"log"`
	Library []string          `yaml:"library"`
	Plugin  []*handler.Config `yaml:"plugin"`
	Include []string          `yaml:"include"`
}

var expSyntax = regexp.MustCompile("\\${{.*}}")

// parseConfig loads a yaml config from path f.
func parseConfig(f string) (*Config, error) {
	b, err := ioutil.ReadFile(f)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	b = expSyntax.ReplaceAllFunc(b, func(syntax []byte) []byte {
		if err != nil {
			return syntax
		}
		cmd := string(bytes.Trim(syntax[3:len(syntax)-2], " "))
		out, innerErr := ext_exec.GetOutputFromCmd(ctx, cmd)
		if innerErr != nil {
			err = fmt.Errorf("cmd [%s] failed with err: %w", cmd, innerErr)
			return syntax
		}
		return bytes.Trim(out, "\r\n")
	})
	if err != nil {
		return nil, err
	}

	c := new(Config)
	if err := yaml.Unmarshal(b, c); err != nil {
		return nil, err
	}

	return c, nil
}

// GenConfig generates a config template to path p.
func GenConfig(p string) error {
	c := new(Config)
	c.Log.Level = "info"

	c.Plugin = append(
		c.Plugin,
		&handler.Config{
			Tag:  "server",
			Type: "server",
			Args: map[string]interface{}{
				"entry": []interface{}{"forward_google"},
				"server": []interface{}{
					map[string]interface{}{
						"protocol": "udp",
						"addr":     "127.0.0.1:53",
					},
					map[string]interface{}{
						"protocol": "tcp",
						"addr":     "127.0.0.1:53",
					},
					map[string]interface{}{
						"protocol": "udp",
						"addr":     "[::1]:53",
					},
					map[string]interface{}{
						"protocol": "tcp",
						"addr":     "[::1]:53",
					},
				},
			},
		},
	)

	c.Plugin = append(c.Plugin, &handler.Config{
		Tag:  "forward_google",
		Type: "forward",
		Args: map[string]interface{}{
			"upstream": []interface{}{
				map[string]interface{}{
					"addr": "https://dns.google/dns-query",
					"ip_addr": []interface{}{
						"8.8.8.8", "2001:4860:4860::8888",
					},
				},
			},
		},
	})
	return c.Save(p)
}

func (c *Config) Save(p string) error {
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := yaml.NewEncoder(f)
	encoder.SetIndent(2)
	defer encoder.Close()
	err = encoder.Encode(c)
	if err != nil {
		return err
	}

	return err
}
