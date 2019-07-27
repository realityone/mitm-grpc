package main

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	pb "google.golang.org/grpc/reflection/grpc_testingv3"
)

// Service is
type Service struct {
}

// Search is
func (s *Service) Search(ctx context.Context, req *pb.SearchRequestV3) (*pb.SearchResponseV3, error) {
	if strings.HasPrefix(req.Query, "error:") {
		e := strings.TrimPrefix(req.Query, "error:")
		return nil, errors.New(e)
	}
	reply := &pb.SearchResponseV3{
		Results: []*pb.SearchResponseV3_Result{
			{
				Url:   "http://reality0ne.com/pages/1",
				Title: "testing-realitye0ne-com",
				Snippets: []string{
					"realityone's blog",
					"hello world",
					req.Query,
				},
				Metadata: map[string]*pb.SearchResponseV3_Result_Value{
					"name": {
						Val: &pb.SearchResponseV3_Result_Value_Str{
							Str: "realityone",
						},
					},
					"age": {
						&pb.SearchResponseV3_Result_Value_Int{
							Int: 18,
						},
					},
					"weight": {
						Val: &pb.SearchResponseV3_Result_Value_Real{
							Real: 48.6,
						},
					},
				},
			},
		},
		State: pb.SearchResponseV3_FRESH,
	}
	return reply, nil
}

// StreamingSearch is
func (s *Service) StreamingSearch(stream pb.SearchServiceV3_StreamingSearchServer) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		reply, err := s.Search(stream.Context(), req)
		if err != nil {
			return err
		}
		if err := stream.Send(reply); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	addr := "0.0.0.0:9100"
	logrus.Infof("Starting testing backend server on %q...", addr)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}

	server := grpc.NewServer()
	pb.RegisterSearchServiceV3Server(server, &Service{})
	logrus.Fatal(server.Serve(lis))
}
