package main

import (
	"encoding/json"
	"errors"
	"log"
	"time"

	"examples/booking/data"
	"examples/booking/srv/auth/proto"

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

type Auth struct {
	customers map[string]*auth.Customer
}

// VerifyToken returns a customer from authentication token.
func (s *Auth) VerifyToken(ctx context.Context, req *auth.Request, rsp *auth.Result) error {
	md, _ := metadata.FromContext(ctx)
	traceID := md["traceID"]

	if tr, ok := trace.FromContext(ctx); ok {
		tr.LazyPrintf("traceID %s", traceID)
	}

	customer := s.customers[req.AuthToken]
	if customer == nil {
		return errors.New("Invalid Token")
	}

	rsp.Customer = customer
	return nil
}

// loadCustomers loads customers from a JSON file.
func loadCustomerData(path string) map[string]*auth.Customer {
	file := data.MustAsset(path)
	customers := []*auth.Customer{}

	// unmarshal JSON
	if err := json.Unmarshal(file, &customers); err != nil {
		log.Fatalf("Failed to unmarshal json: %v", err)
	}

	// create customer lookup map
	cache := make(map[string]*auth.Customer)
	for _, c := range customers {
		cache[c.AuthToken] = c
	}
	return cache
}

func main() {
	service := micro.NewService(
		micro.Client(client.NewClient(client.Wrap(logWrap))),
		micro.Server(server.NewServer(server.WrapHandler(logHandlerWrapper))),
		micro.Name("go.micro.srv.auth"),
		micro.RegisterTTL(time.Second*3),
		micro.RegisterInterval(time.Second*3),
	)

	service.Init()

	auth.RegisterAuthHandler(service.Server(), &Auth{
		customers: loadCustomerData("data/customers.json"),
	})

	service.Run()
}
