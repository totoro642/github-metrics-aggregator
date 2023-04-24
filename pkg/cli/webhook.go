// Copyright 2023 The Authors (see AUTHORS file)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cli

import (
	"context"
	"fmt"
	"net/http"

	"github.com/abcxyz/github-metrics-aggregator/pkg/version"
	"github.com/abcxyz/github-metrics-aggregator/pkg/webhook"
	"github.com/abcxyz/pkg/cli"
	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/pkg/serving"
	"google.golang.org/api/option"
)

var _ cli.Command = (*WebhookServerCommand)(nil)

type WebhookServerCommand struct {
	cli.BaseCommand

	cfg *webhook.Config

	// testFlagSetOpts is only used for testing.
	testFlagSetOpts []cli.Option

	testPubSubClientOptions []option.ClientOption
}

func (c *WebhookServerCommand) Desc() string {
	return `Start a webhook server for GitHub Metrics Aggregator`
}

func (c *WebhookServerCommand) Help() string {
	return `
Usage: {{ COMMAND }} [options]
  Start a webhook server for GitHub Metrics Aggregator.
`
}

func (c *WebhookServerCommand) Flags() *cli.FlagSet {
	c.cfg = &webhook.Config{}
	set := cli.NewFlagSet(c.testFlagSetOpts...)
	return c.cfg.ToFlags(set)
}

func (c *WebhookServerCommand) Run(ctx context.Context, args []string) error {
	server, mux, err := c.RunUnstarted(ctx, args)
	if err != nil {
		return err
	}

	return server.StartHTTPHandler(ctx, mux)
}

func (c *WebhookServerCommand) RunUnstarted(ctx context.Context, args []string) (*serving.Server, http.Handler, error) {
	f := c.Flags()
	if err := f.Parse(args); err != nil {
		return nil, nil, fmt.Errorf("failed to parse flags: %w", err)
	}
	args = f.Args()
	if len(args) > 0 {
		return nil, nil, fmt.Errorf("unexpected arguments: %q", args)
	}

	logger := logging.FromContext(ctx)
	logger.Debugw("server starting",
		"name", version.Name,
		"commit", version.Commit,
		"version", version.Version)

	if err := c.cfg.Validate(); err != nil {
		return nil, nil, fmt.Errorf("invalid configuration: %w", err)
	}
	logger.Debugw("loaded configuration", "config", c.cfg)

	pubsubClientOpts := append([]option.ClientOption{option.WithUserAgent("abcxyz/github-metrics-aggregator")}, c.testPubSubClientOptions...)
	webhookServer, err := webhook.NewServer(ctx, c.cfg, pubsubClientOpts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create server: %w", err)
	}

	mux := webhookServer.Routes(ctx)

	server, err := serving.New(c.cfg.Port)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create serving infrastructure: %w", err)
	}

	return server, mux, nil
}
