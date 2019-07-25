package server

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/resolver"
)

// Server is
type Server struct {
	*grpc.Server

	clientLock sync.RWMutex
	clients    map[string]*grpc.ClientConn
}

type dummyMessage struct {
	payload []byte
}

func (dm *dummyMessage) Reset()                   { dm.payload = dm.payload[:0] }
func (dm *dummyMessage) String() string           { return fmt.Sprintf("%+v", dm.payload) }
func (dm *dummyMessage) ProtoMessage()            {}
func (dm *dummyMessage) Marshal() ([]byte, error) { return dm.payload, nil }
func (dm *dummyMessage) Unmarshal(in []byte) error {
	dm.payload = append(dm.payload[:0], in...)
	return nil
}

// Handler is
func (s *Server) Handler(srv interface{}, stream grpc.ServerStream) error {
	serviceMethod, _ := grpc.MethodFromServerStream(stream) // already checked
	service, method, err := splitServiceMethod(serviceMethod)
	if err != nil {
		return grpc.Errorf(codes.FailedPrecondition, err.Error())
	}

	logrus.Infof("Handler: service: %+v: method: %+v", service, method)
	ctx := stream.Context()
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		logrus.Infof("In coming metadata: %+v", md)
	}
	req := &dummyMessage{}
	if err := stream.RecvMsg(req); err != nil {
		return err
	}

	conn := func() (*grpc.ClientConn, error) {
		s.clientLock.RLock()
		cc, ok := s.clients[service]
		s.clientLock.RUnlock()
		if ok {
			return cc, nil
		}

		target, ok := targetFromMetadata(md)
		if !ok {
			target = fmt.Sprintf("mitm-proxy://%s/", service)
		}

		newCC, err := grpc.DialContext(ctx, target, grpc.WithBlock(), grpc.WithInsecure())
		if err != nil {
			return nil, err
		}
		s.clientLock.Lock()
		defer s.clientLock.Unlock()
		cc, ok = s.clients[service]
		if ok {
			logrus.Debugf("Already has established connection for %q", service)
			newCC.Close()
			return cc, nil
		}
		s.clients[service] = newCC
		return newCC, nil
	}

	cc, err := conn()
	if err != nil {
		return err
	}
	reply := &dummyMessage{}
	if err := cc.Invoke(ctx, serviceMethod, req, reply); err != nil {
		return err
	}
	if err := stream.SendHeader(md); err != nil {
		return err
	}
	if err := stream.SendMsg(reply); err != nil {
		return err
	}
	logrus.Infof("Request: %+v, Reply: %+v", req, reply)
	return nil
}

func wrapped(handler grpc.StreamHandler) grpc.StreamHandler {
	return func(srv interface{}, stream grpc.ServerStream) error {
		serviceMethod, ok := grpc.MethodFromServerStream(stream)
		if !ok {
			return grpc.Errorf(codes.Internal, "failed to get method from stream")
		}
		if err := handler(srv, stream); err != nil {
			logrus.Errorf("Failed to handle request stream: method: %q: %+v", serviceMethod, err)
			return err
		}
		return nil
	}
}

// NewServer is
func NewServer(resolver resolver.Resolver) *Server {

	server := &Server{
		clients: map[string]*grpc.ClientConn{},
	}
	server.Server = grpc.NewServer(
		grpc.UnknownServiceHandler(wrapped(server.Handler)),
	)
	return server
}

func splitServiceMethod(serviceMethod string) (string, string, error) {
	if serviceMethod != "" && serviceMethod[0] == '/' {
		serviceMethod = serviceMethod[1:]
	}
	pos := strings.LastIndex(serviceMethod, "/")
	if pos == -1 {
		return "", "", errors.Errorf("malformed method name: %q", serviceMethod)
	}
	service := serviceMethod[:pos]
	method := serviceMethod[pos+1:]
	return service, method, nil
}

func targetFromMetadata(md metadata.MD) (string, bool) {
	target := md["x-mitm-target"]
	if len(target) <= 0 {
		return "", false
	}
	return target[rand.Intn(len(target))], true
}
