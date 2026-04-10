package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/exp/api/remote"
	writev2 "github.com/prometheus/client_golang/exp/api/remote/genproto/v2"
)

func main() {
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	apiOpts := []remote.APIOption{remote.WithAPIHTTPClient(httpClient)}
	api, err := remote.NewAPI("http://localhost:9090/", apiOpts...)
	if err != nil {
		log.Fatalf("remote API: %v", err)
	}

	sym := writev2.NewSymbolTable()
	labelsRefs := []string{
		"__name__", "prom_forge_client_golang_gpu",
		"host", "node-01",
		"gpu", "0",
		"region", "us-east-1",
	}
	help := "prom_forge_client_golang_gpu help"
	unit := "_ratio"

	ts := &writev2.TimeSeries{
		LabelsRefs: sym.SymbolizeLabels(labelsRefs, nil),
		Samples: []*writev2.Sample{
			{Value: 100.0, Timestamp: time.Now().UnixMilli()},
		},
		Metadata: &writev2.Metadata{
			Type:    writev2.Metadata_METRIC_TYPE_GAUGE,
			HelpRef: sym.Symbolize(help),
			UnitRef: sym.Symbolize(unit),
		},
	}

	req := &writev2.Request{
		Symbols:    sym.Symbols(),
		Timeseries: []*writev2.TimeSeries{ts},
	}
	req.Symbols = sym.Symbols()

	fmt.Printf("request: %+v\n", req)

	stats, err := api.Write(context.Background(), remote.WriteV2MessageType, req)
	if err != nil {
		log.Fatalf("remote_write v2 failed: %v", err)
	}
	log.Printf("remote_write v2 ok samples=%d", stats.Samples)
}
