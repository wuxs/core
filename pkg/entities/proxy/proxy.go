package proxy

import (
	"context"
	"fmt"

	cloudevents "github.com/cloudevents/sdk-go"
	"github.com/pkg/errors"
	pb "github.com/tkeel-io/core/api/core/v1"
	"github.com/tkeel-io/core/pkg/config"
	xerrors "github.com/tkeel-io/core/pkg/errors"
	zfield "github.com/tkeel-io/core/pkg/logger"
	"github.com/tkeel-io/core/pkg/placement"
	"github.com/tkeel-io/core/pkg/runtime/message"
	"github.com/tkeel-io/core/pkg/runtime/state"
	"github.com/tkeel-io/core/pkg/util"
	"github.com/tkeel-io/core/pkg/util/discovery"
	"github.com/tkeel-io/kit/log"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const RetryConnectNum = 3

type Proxy struct {
	info         discovery.Service
	grpcConns    map[string]pb.ProxyClient
	serviceInfos map[string]discovery.Service
	stateManager state.Manager
	coreResolver discovery.Resolver
	ctx          context.Context
}

func NewProxy(ctx context.Context, stateManager state.Manager) (*Proxy, error) {
	var (
		err      error
		resolver discovery.Resolver
		cfg      = discovery.Config{
			Endpoints:   config.Get().Discovery.Endpoints,
			HeartTime:   config.Get().Discovery.HeartTime,
			DialTimeout: config.Get().Discovery.DialTimeout,
		}
	)

	if resolver, err = discovery.New(cfg); nil != err {
		log.Error("new Resolver instance", zap.Error(err), zfield.Endpoints(cfg.Endpoints))
		return nil, errors.Wrap(err, "new Resolver instance")
	}

	info := discovery.Service{
		Name:     config.Get().Server.Name,
		AppID:    config.Get().Server.AppID,
		Host:     util.ResolveAddr(),
		Metadata: map[string]string{},
	}

	proxyInst := &Proxy{
		ctx:          ctx,
		info:         info,
		grpcConns:    make(map[string]pb.ProxyClient),
		serviceInfos: make(map[string]discovery.Service),
		stateManager: stateManager,
		coreResolver: resolver,
	}

	// resolve cluster.
	err = resolver.Resolve(ctx, []discovery.ResolveHandler{proxyInst.handleService})
	if nil != err {
		log.Error("resolve cluster", zap.Error(err), zfield.Endpoints(cfg.Endpoints))
	}
	return proxyInst, errors.Wrap(err, "new Proxy instance")
}

func (p *Proxy) RouteMessage(ctx context.Context, ev cloudevents.Event) error {
	var (
		err      error
		entityID string
	)

	ev.ExtensionAs(message.ExtEntityID, &entityID)
	selectQueue := placement.Global().Select(entityID)
	nodeName := selectQueue.NodeName

	switch nodeName {
	case p.info.Name:
		var msgCtx message.Context
		if msgCtx, err = message.From(ctx, ev); nil != err {
			log.Error("parse event", zfield.ID(ev.ID()), zfield.Event(ev))
			return errors.Wrap(err, "parse event")
		}

		err = p.stateManager.HandleMessage(ctx, msgCtx)
	default:
		// select proxy client.
		var proxyClient pb.ProxyClient
		if proxyClient, err = p.selectConn(nodeName); nil != err {
			log.Error("select proxy server", zap.Error(err),
				zfield.Name(nodeName), zfield.Event(ev))
			return errors.Wrap(err, "select proxy server")
		}

		var bytes []byte
		if bytes, err = ev.DataBytes(); nil != err {
			log.Error("encode event data",
				zap.Error(err), zfield.Event(ev))
			return errors.Wrap(err, "encode event data")
		}

		_, err = proxyClient.Route(ctx, &pb.RouteRequest{
			Header: message.GetAttributes(ev),
			Data:   bytes,
		})
	}

	return errors.Wrap(err, "route message")
}

func (p *Proxy) handleService(et discovery.EnventType, info discovery.Service) {
	switch et {
	case discovery.PUT:
		log.Info("upsert service instance", zfield.App(info.AppID),
			zfield.Name(info.Name), zfield.Host(info.Host), zap.Any("metadata", info.Metadata))
		p.serviceInfos[info.Name] = info
	case discovery.DELETE:
		log.Info("delete service instance", zfield.App(info.AppID),
			zfield.Name(info.Name), zfield.Host(info.Host), zap.Any("metadata", info.Metadata))
		delete(p.serviceInfos, info.Name)
		delete(p.grpcConns, info.Name)
	default:
	}
}

func (p *Proxy) selectConn(name string) (pb.ProxyClient, error) {
	info, has := p.serviceInfos[name]
	if !has {
		log.Error("core cluster node not exists",
			zap.Error(xerrors.ErrNodeNotExist), zfield.Name(name))
		return nil, xerrors.ErrNodeNotExist
	}

	// select from cache first.
	if conn, has := p.grpcConns[name]; has {
		return conn, nil
	}

	// connnect on proxy server.
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", info.Host, info.Port), opts...)
	if nil != err {
		log.Error("dial proxy server", zap.Error(err), zfield.App(info.AppID), zfield.Port(info.Port),
			zfield.Name(info.Name), zfield.Host(info.Host), zap.Any("metadata", info.Metadata))
	}
	proxyClient := pb.NewProxyClient(conn)

	p.grpcConns[name] = proxyClient
	return proxyClient, nil
}
