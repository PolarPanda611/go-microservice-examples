package main

import (
	"encoding/json"
	"log"

	"examples/booking/data"
	"examples/booking/srv/profile/proto"

	"context"
	"golang.org/x/net/trace"

	"fmt"
	"github.com/micro/go-micro"
	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/metadata"
	"github.com/micro/go-micro/server"
)

type logWrapper struct {
	client.Client
}

func (l *logWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	md, _ := metadata.FromContext(ctx)
	fmt.Printf("[Log Wrapper] ctx: %v service: %s method: %s\n", md, req.Service(), req.Method())
	return l.Client.Call(ctx, req, rsp)
}

// Implements client.Wrapper as logWrapper
func logWrap(c client.Client) client.Client {
	return &logWrapper{c}
}

// Implements the server.HandlerWrapper
func logHandlerWrapper(fn server.HandlerFunc) server.HandlerFunc {
	return func(ctx context.Context, req server.Request, rsp interface{}) error {
		fmt.Printf("[Log Wrapper] Before serving request method: %v\n", req.Method())
		err := fn(ctx, req, rsp)
		fmt.Println("[Log Wrapper] After serving request")
		return err
	}
}

type Profile struct {
	hotels map[string]*profile.Hotel
}

// GetProfiles returns hotel profiles for requested IDs
func (s *Profile) GetProfiles(ctx context.Context, req *profile.Request, rsp *profile.Result) error {
	md, _ := metadata.FromContext(ctx)
	traceID := md["traceID"]
	if tr, ok := trace.FromContext(ctx); ok {
		tr.LazyPrintf("traceID %s", traceID)
	}

	for _, i := range req.HotelIds {
		rsp.Hotels = append(rsp.Hotels, s.hotels[i])
	}
	return nil
}

// loadProfiles loads hotel profiles from a JSON file.
func loadProfiles(path string) map[string]*profile.Hotel {
	file := data.MustAsset(path)

	// unmarshal json profiles
	hotels := []*profile.Hotel{}
	if err := json.Unmarshal(file, &hotels); err != nil {
		log.Fatalf("Failed to load json: %v", err)
	}

	profiles := make(map[string]*profile.Hotel)
	for _, hotel := range hotels {
		profiles[hotel.Id] = hotel
	}
	return profiles
}

func main() {
	service := micro.NewService(
		micro.Client(client.NewClient(client.Wrap(logWrap))),
		micro.Server(server.NewServer(server.WrapHandler(logHandlerWrapper))),
		micro.Name("go.micro.srv.profile"),
	)

	service.Init()

	profile.RegisterProfileHandler(service.Server(), &Profile{
		hotels: loadProfiles("data/profiles.json"),
	})

	service.Run()
}
