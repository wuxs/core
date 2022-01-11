/*
Copyright 2021 The tKeel Authors.

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

package pubsub

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/tkeel-io/core/pkg/source"

	"github.com/dapr/go-sdk/service/common"
)

type Meta struct {
	name   string
	topics []string
	Pubsub string `json:"pubsub"`
	Topics string `json:"topics"`
}

type Source struct {
	name    string
	pubsub  string
	topics  []string
	service common.Service
	ctx     context.Context
}

func OpenSource(ctx context.Context, metadata source.Metadata, service common.Service) (source.ISource, error) {
	var (
		err  error
		meta *Meta
	)

	if meta, err = acquireMeta(metadata); err != nil {
		return nil, err
	}

	return &Source{
		ctx:     ctx,
		name:    meta.name,
		pubsub:  meta.Pubsub,
		topics:  meta.topics,
		service: service,
	}, nil
}

func (s *Source) String() string {
	return s.name
}

func (s *Source) StartReceiver(handler source.Handler) error {
	for _, topic := range s.topics {
		if err := s.service.AddTopicEventHandler(
			&common.Subscription{
				PubsubName: s.pubsub,
				Topic:      topic,
			}, handler); err != nil {
			return errors.Unwrap(err)
		}
	}
	return nil
}

func (s *Source) Close() error {
	return errors.New("not implement")
}

func acquireMeta(metadata source.Metadata) (*Meta, error) {
	b, err := json.Marshal(metadata.Properties)
	if err != nil {
		return nil, errors.Unwrap(err)
	}

	meta := Meta{}
	err = json.Unmarshal(b, &meta)
	if err != nil {
		return nil, errors.Unwrap(err)
	}

	meta.name = metadata.Name
	meta.topics = strings.Split(meta.Topics, ",")

	// check name.
	if meta.Pubsub == "" {
		return &meta, errors.New("field Name required")
	}
	if len(meta.topics) == 0 {
		return &meta, errors.New("field Topics required")
	}

	return &meta, nil
}

func init() {
	source.Register(&source.BaseSourceGenerator{SourceType: source.PubSub, Generator: OpenSource})
}