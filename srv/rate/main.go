package main

import (
	"encoding/json"
	"log"
	"time"

	"go-microservice-examples/data"
	"go-microservice-examples/srv/rate/proto"

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

type stay struct {
	HotelID string
	InDate  string
	OutDate string
}

type Rate struct {
	rateTable map[stay]*rate.RatePlan
}

// GetRates gets rates for hotels for specific date range.
func (s *Rate) GetRates(ctx context.Context, req *rate.Request, rsp *rate.Result) error {
	md, _ := metadata.FromContext(ctx)
	traceID := md["traceID"]

	if tr, ok := trace.FromContext(ctx); ok {
		tr.LazyPrintf("traceID %s", traceID)
	}

	for _, hotelID := range req.HotelIds {
		stay := stay{
			HotelID: hotelID,
			InDate:  req.InDate,
			OutDate: req.OutDate,
		}
		if s.rateTable[stay] != nil {
			rsp.RatePlans = append(rsp.RatePlans, s.rateTable[stay])
		}
	}
	return nil
}

// loadRates loads rate codes from JSON file.
func loadRateTable(path string) map[stay]*rate.RatePlan {
	file := data.MustAsset("data/rates.json")

	rates := []*rate.RatePlan{}
	if err := json.Unmarshal(file, &rates); err != nil {
		log.Fatalf("Failed to load json: %v", err)
	}

	rateTable := make(map[stay]*rate.RatePlan)
	for _, ratePlan := range rates {
		stay := stay{
			HotelID: ratePlan.HotelId,
			InDate:  ratePlan.InDate,
			OutDate: ratePlan.OutDate,
		}
		rateTable[stay] = ratePlan
	}
	return rateTable
}

func main() {
	service := micro.NewService(
		micro.Client(client.NewClient(client.Wrap(logWrap))),
		micro.Server(server.NewServer(server.WrapHandler(logHandlerWrapper))),
		micro.Name("go.micro.srv.rate"),
		micro.RegisterTTL(time.Second*3),
		micro.RegisterInterval(time.Second*3),
	)

	service.Init()

	rate.RegisterRateHandler(service.Server(), &Rate{
		rateTable: loadRateTable("data/rates.json"),
	})

	service.Run()
}
