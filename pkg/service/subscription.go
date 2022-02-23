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

package service

import (
	"context"

	"github.com/pkg/errors"
	pb "github.com/tkeel-io/core/api/core/v1"
	xerrors "github.com/tkeel-io/core/pkg/errors"
	zfield "github.com/tkeel-io/core/pkg/logger"
	apim "github.com/tkeel-io/core/pkg/manager"
	"github.com/tkeel-io/core/pkg/runtime"
	"github.com/tkeel-io/core/pkg/runtime/state"
	"github.com/tkeel-io/core/pkg/runtime/subscription"
	"github.com/tkeel-io/kit/log"
	"github.com/tkeel-io/tdtl"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

type SubscriptionService struct {
	pb.UnimplementedSubscriptionServer
	ctx        context.Context
	cancel     context.CancelFunc
	inited     *atomic.Bool
	apiManager apim.APIManager
}

// NewSubscriptionService returns a new SubscriptionService.
func NewSubscriptionService(ctx context.Context) (*SubscriptionService, error) {
	ctx, cancel := context.WithCancel(ctx)

	return &SubscriptionService{
		ctx:    ctx,
		cancel: cancel,
		inited: atomic.NewBool(false),
	}, nil
}

func (s *SubscriptionService) Init(apiManager apim.APIManager) {
	s.apiManager = apiManager
	s.inited.Store(true)
}

func interface2string(in interface{}) (out string) {
	switch inString := in.(type) {
	case string:
		out = inString
	case tdtl.Node:
		out = inString.String()
	default:
		out = ""
	}
	return
}

func (s *SubscriptionService) entity2SubscriptionResponse(entity *Entity) (out *pb.SubscriptionResponse) {
	if entity == nil {
		return
	}

	out = &pb.SubscriptionResponse{}

	out.Id = entity.ID
	out.Owner = entity.Owner
	out.Source = entity.Source
	out.Subscription = &pb.SubscriptionObject{}
	out.Subscription.Filter = interface2string(entity.Properties[subscription.SubscriptionFieldFilter])
	out.Subscription.Topic = interface2string(entity.Properties[subscription.SubscriptionFieldTopic])
	out.Subscription.Mode = interface2string(entity.Properties[subscription.SubscriptionFieldMode])
	out.Subscription.PubsubName = interface2string(entity.Properties[subscription.SubscriptionFieldPubsubName])
	return out
}

func (s *SubscriptionService) CreateSubscription(ctx context.Context, req *pb.CreateSubscriptionRequest) (out *pb.SubscriptionResponse, err error) {
	if !s.inited.Load() {
		log.Warn("service not ready", zfield.Eid(req.Id))
		return nil, errors.Wrap(xerrors.ErrServerNotReady, "service not ready")
	}

	var entity = new(Entity)
	if req.Id != "" {
		entity.ID = req.Id
	}

	entity.Owner = req.Owner
	entity.Source = req.Source
	entity.Type = runtime.SMTypeSubscription
	parseHeaderFrom(ctx, entity)
	entity.Properties = map[string]tdtl.Node{
		runtime.SMFieldType:                      tdtl.StringNode(entity.Type),
		runtime.SMFieldOwner:                     tdtl.StringNode(entity.Owner),
		runtime.SMFieldSource:                    tdtl.StringNode(entity.Source),
		subscription.SubscriptionFieldMode:       tdtl.StringNode(req.Subscription.Mode),
		subscription.SubscriptionFieldTopic:      tdtl.StringNode(req.Subscription.Topic),
		subscription.SubscriptionFieldFilter:     tdtl.StringNode(req.Subscription.Filter),
		subscription.SubscriptionFieldPubsubName: tdtl.StringNode(req.Subscription.PubsubName),
	}

	if err = s.apiManager.CheckSubscription(ctx, entity); nil != err {
		log.Error("create subscription", zap.Error(err), zfield.Eid(req.Id))
		return
	}

	// set mapper.
	entity.Mappers = []state.Mapper{{
		Name: "subscription",
		TQL:  entity.Properties[subscription.SubscriptionFieldFilter].String(),
	}}

	// set properties.
	if entity, err = s.apiManager.CreateEntity(ctx, entity); nil != err {
		log.Error("create subscription", zap.Error(err), zfield.Eid(req.Id))
		return
	}

	if err = s.apiManager.AppendMapper(ctx, entity); nil != err {
		log.Error("create subscription", zap.Error(err), zfield.Eid(req.Id))
		if err0 := s.apiManager.DeleteEntity(ctx, entity); nil != err0 {
			log.Error("destroy subscription", zap.Error(err0), zfield.Eid(req.Id))
		}
		return
	}

	out = s.entity2SubscriptionResponse(entity)
	return out, errors.Wrap(err, "create subscription")
}

func (s *SubscriptionService) UpdateSubscription(ctx context.Context, req *pb.UpdateSubscriptionRequest) (out *pb.SubscriptionResponse, err error) {
	if !s.inited.Load() {
		log.Warn("service not ready", zfield.Eid(req.Id))
		return nil, errors.Wrap(xerrors.ErrServerNotReady, "service not ready")
	}

	var entity = new(Entity)

	entity.ID = req.Id
	entity.Owner = req.Owner
	entity.Source = req.Source
	entity.Type = runtime.SMTypeSubscription
	parseHeaderFrom(ctx, entity)
	entity.Properties = map[string]tdtl.Node{
		subscription.SubscriptionFieldFilter:     tdtl.StringNode(req.Subscription.Filter),
		subscription.SubscriptionFieldTopic:      tdtl.StringNode(req.Subscription.Topic),
		subscription.SubscriptionFieldMode:       tdtl.StringNode(req.Subscription.Mode),
		subscription.SubscriptionFieldPubsubName: tdtl.StringNode(req.Subscription.PubsubName),
	}

	// set properties.
	if entity, err = s.apiManager.UpdateEntityProps(ctx, entity); nil != err {
		log.Error("update subscription", zap.Error(err), zfield.Eid(req.Id))
		return
	}

	// set mapper.
	entity.Mappers = []state.Mapper{{
		Name: "subscription",
		TQL:  entity.Properties[subscription.SubscriptionFieldFilter].String(),
	}}

	if err = s.apiManager.AppendMapper(ctx, entity); nil != err {
		log.Error("update subscription", zap.Error(err), zfield.Eid(req.Id))
		return
	}

	out = s.entity2SubscriptionResponse(entity)

	return out, errors.Wrap(err, "update subscription")
}

func (s *SubscriptionService) DeleteSubscription(ctx context.Context, req *pb.DeleteSubscriptionRequest) (out *pb.DeleteSubscriptionResponse, err error) {
	if !s.inited.Load() {
		log.Warn("service not ready", zfield.Eid(req.Id))
		return nil, errors.Wrap(xerrors.ErrServerNotReady, "service not ready")
	}

	var entity = new(Entity)
	entity.ID = req.Id
	entity.Type = runtime.SMTypeSubscription
	entity.Owner = req.Owner
	entity.Source = req.Source
	parseHeaderFrom(ctx, entity)
	if err = s.apiManager.DeleteEntity(ctx, entity); nil != err {
		log.Error("delete subscription", zap.Error(err), zfield.Eid(req.Id))
		return
	}

	out = &pb.DeleteSubscriptionResponse{Id: req.Id, Status: "ok"}
	return
}

func (s *SubscriptionService) GetSubscription(ctx context.Context, req *pb.GetSubscriptionRequest) (out *pb.SubscriptionResponse, err error) {
	if !s.inited.Load() {
		log.Warn("service not ready", zfield.Eid(req.Id))
		return nil, errors.Wrap(xerrors.ErrServerNotReady, "service not ready")
	}

	var entity = new(Entity)
	entity.ID = req.Id
	entity.Type = runtime.SMTypeSubscription
	entity.Owner = req.Owner
	entity.Source = req.Source
	parseHeaderFrom(ctx, entity)
	if entity, err = s.apiManager.GetEntity(ctx, entity); nil != err {
		log.Error("get subscription", zap.Error(err), zfield.Eid(req.Id))
		return
	}
	out = s.entity2SubscriptionResponse(entity)
	return
}

func (s *SubscriptionService) ListSubscription(ctx context.Context, req *pb.ListSubscriptionRequest) (out *pb.ListSubscriptionResponse, err error) {
	if !s.inited.Load() {
		log.Warn("service not ready")
		return nil, errors.Wrap(xerrors.ErrServerNotReady, "service not ready")
	}

	return &pb.ListSubscriptionResponse{}, nil
}
