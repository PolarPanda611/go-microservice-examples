package main

import (
	"encoding/json"
	"log"

	"github.com/hailocab/go-geoindex"
	"go-microservice-examples/data"
	"go-microservice-examples/srv/geo/proto"

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

const (
	maxSearchRadius  = 10
	maxSearchResults = 5
)

type point struct {
	Pid  string  `json:"hotelId"`
	Plat float64 `json:"lat"`
	Plon float64 `json:"lon"`
}

// Implement Point interface
func (p *point) Lat() float64 { return p.Plat }
func (p *point) Lon() float64 { return p.Plon }
func (p *point) Id() string   { return p.Pid }

type Geo struct {
	index *geoindex.ClusteringIndex
}

// Nearby returns all hotels within a given distance.
func (s *Geo) Nearby(ctx context.Context, req *geo.Request, rsp *geo.Result) error {
	md, _ := metadata.FromContext(ctx)
	traceID := md["traceID"]

	if tr, ok := trace.FromContext(ctx); ok {
		tr.LazyPrintf("traceID %s", traceID)
	}

	// create center point for query
	center := &geoindex.GeoPoint{
		Pid:  "",
		Plat: float64(req.Lat),
		Plon: float64(req.Lon),
	}

	// find points around center point
	points := s.index.KNearest(center, maxSearchResults, geoindex.Km(maxSearchRadius), func(p geoindex.Point) bool {
		return true
	})

	for _, p := range points {
		rsp.HotelIds = append(rsp.HotelIds, p.Id())
	}
	return nil
}

// newGeoIndex returns a geo index with points loaded
func newGeoIndex(path string) *geoindex.ClusteringIndex {
	file := data.MustAsset(path)

	// unmarshal json points
	var points []*point
	if err := json.Unmarshal(file, &points); err != nil {
		log.Fatalf("Failed to load hotels: %v", err)
	}

	// add points to index
	index := geoindex.NewClusteringIndex()
	for _, point := range points {
		index.Add(point)
	}
	return index
}

func main() {
	service := micro.NewService(
		micro.Client(client.NewClient(client.Wrap(logWrap))),
		micro.Server(server.NewServer(server.WrapHandler(logHandlerWrapper))),
		micro.Name("go.micro.srv.geo"),
	)

	service.Init()

	geo.RegisterGeoHandler(service.Server(), &Geo{
		index: newGeoIndex("data/locations.json"),
	})

	service.Run()
}
