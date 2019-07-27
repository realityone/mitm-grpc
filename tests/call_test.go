package tests_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	pb "google.golang.org/grpc/reflection/grpc_testingv3"
)

const proxy = "192.168.10.165:9000"

func newClient() pb.SearchServiceV3Client {
	cc, err := grpc.Dial(proxy, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	return pb.NewSearchServiceV3Client(cc)
}

func withTarget(ctx context.Context, target string) context.Context {
	return metadata.NewOutgoingContext(ctx, metadata.MD{
		"x-mitm-target": []string{target},
	})
}

func TestCallProxy(t *testing.T) {
	client := newClient()

	target := "192.168.10.148:9100"
	ctx := withTarget(context.Background(), target)
	req := &pb.SearchRequestV3{Query: "realityone"}
	reply, err := client.Search(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, reply)
	t.Logf("Request to %q via proxy %q: req: %+v reply: %+v", target, proxy, req, reply)
}
