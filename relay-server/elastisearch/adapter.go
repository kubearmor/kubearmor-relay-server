package elastisearch

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/cenkalti/backoff/v4"
	"github.com/dustin/go-humanize"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esutil"
	"github.com/google/uuid"
	kg "github.com/kubearmor/kubearmor-relay-server/relay-server/log"
	"github.com/kubearmor/kubearmor-relay-server/relay-server/server"
	"log"
	"strings"
	"sync/atomic"
	"time"
)

var (
	countSuccessful uint64
	countEntered    uint64
	start           time.Time
)

// ElasticsearchClient Structure
type ElasticsearchClient struct {
	kaClient    *server.LogClient
	esClient    *elasticsearch.Client
	cancel      context.CancelFunc
	bulkIndexer esutil.BulkIndexer
	ctx         context.Context
	messageCh   chan interface{}
	alertCh     chan interface{}
	logCh       chan interface{}
}

func NewElasticsearchClient(esURL, Endpoint string) (*ElasticsearchClient, error) {
	//Endpoint = "172.26.40.47:32767"
	retryBackoff := backoff.NewExponentialBackOff()
	cfg := elasticsearch.Config{
		Addresses: []string{esURL},

		// Retry on 429 TooManyRequests statuses
		RetryOnStatus: []int{502, 503, 504, 429},

		// Configure the backoff function
		RetryBackoff: func(i int) time.Duration {
			if i == 1 {
				retryBackoff.Reset()
			}
			return retryBackoff.NextBackOff()
		},
		MaxRetries: 5,
	}

	esClient, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Elasticsearch client: %v", err)
	}
	res, err := esClient.Ping()
	if err != nil {
		log.Fatalf("Error pinging the cluster: %s", err)
	}
	println(res)
	bi, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Client:        esClient,         // The Elasticsearch client
		FlushBytes:    1000000,          // The flush threshold in bytes [3mb]
		FlushInterval: 30 * time.Second, // The periodic flush interval [30 secs]
	})
	if err != nil {
		log.Fatalf("Error creating the indexer: %s", err)
	}
	//messageCh := make(chan interface{}, 1000000)
	alertCh := make(chan interface{}, 10000)
	//logCh := make(chan interface{}, 1000000)
	kaClient := server.NewClient(Endpoint)
	return &ElasticsearchClient{kaClient: kaClient, bulkIndexer: bi, esClient: esClient, alertCh: alertCh}, nil
}

// GetElasticsearchClient returns an elastisearch client
func GetElasticsearchClient() (*ElasticsearchClient, error) {
	elastisearchServiceURL := flag.String("esSvcURL", "http://127.0.0.1:9200", "es Svc URL")
	return NewElasticsearchClient(*elastisearchServiceURL, "localhost:32767")
}

// bulk Index takes an interface and index name and
// adds to the bulkIndexer which will get flushed
// after FlushBytes or FlushInterval has reached
func (ecl *ElasticsearchClient) bulkIndex(a interface{}, index string) {
	countEntered++
	//fmt.Printf("Entered Bulk Index : %s, %s\n", countEntered, index)
	data, err := json.Marshal(a)
	if err != nil {
		log.Fatalf("Error marshaling data: %s", err)
	}

	err = ecl.bulkIndexer.Add(
		ecl.ctx,
		esutil.BulkIndexerItem{
			Index:      index,
			Action:     "index",
			DocumentID: uuid.New().String(),
			Body:       bytes.NewReader(data),
			OnSuccess: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem) {
				atomic.AddUint64(&countSuccessful, 1)
				//println("SUCCESS")
			},
			OnFailure: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem, err error) {
				if err != nil {
					log.Printf("ERROR: %s", err)
				} else {
					log.Printf("ERROR: %s: %s", res.Error.Type, res.Error.Reason)
				}
			},
		},
	)
	//println("ADDED")

	if err != nil {
		log.Fatalf("Error adding items to bulk indexer: %s", err)
	}
}

func (ecl *ElasticsearchClient) Start() error {

	start = time.Now()
	client := ecl.kaClient
	ecl.ctx, ecl.cancel = context.WithCancel(context.Background())

	// do healthcheck
	if ok := client.DoHealthCheck(); !ok {
		return fmt.Errorf("failed to check the liveness of the gRPC server")
	}
	kg.Printf("Checked the liveness of the gRPC server")

	// start goroutines for each stream
	client.WgServer.Add(1)
	//go func() {
	//	defer client.WgServer.Done()
	//	for client.Running {
	//		res, err := client.MsgStream.Recv()
	//		if err != nil {
	//			kg.Warnf("Failed to receive a message (%s)", client.Server)
	//			break
	//		}
	//		//tel, _ := json.Marshal(res)
	//		//fmt.Printf("%s\n", string(tel))
	//		ecl.messageCh <- *res
	//	}
	//}()
	go func() {
		defer client.WgServer.Done()
		for client.Running {
			res, err := client.AlertStream.Recv()
			if err != nil {
				kg.Warnf("Failed to receive an alert (%s)", client.Server)
				break
			}
			//tel, _ := json.Marshal(res)
			//fmt.Printf("%s\n", string(tel))
			ecl.alertCh <- res
		}
	}()
	//go func() {
	//	defer client.WgServer.Done()
	//	for client.Running {
	//		res, err := client.LogStream.Recv()
	//		if err != nil {
	//			kg.Warnf("Failed to receive a log (%s)", client.Server)
	//			break
	//		}
	//		//tel, _ := json.Marshal(res)
	//		//fmt.Printf("%s\n", string(tel))
	//		ecl.logCh <- res
	//
	//	}
	//}()

	// start goroutines to consume messages from the channels and bulk index them
	for i := 0; i < 5; i++ {
		go func() {
			for {
				select {
				//case msg := <-ecl.messageCh:
				//	ecl.bulkIndex(msg, "message")
				case alert := <-ecl.alertCh:
					ecl.bulkIndex(alert, "alert")
				//case log := <-ecl.logCh:
				//	ecl.bulkIndex(log, "log")
				case <-ecl.ctx.Done():
					return
				}
			}
		}()
	}
	return nil
}

func (ecl *ElasticsearchClient) Stop() error {
	logClient := ecl.kaClient
	logClient.Running = false
	time.Sleep(2 * time.Second)

	//Destoy KubeArmor Relay Client
	if err := logClient.DestroyClient(); err != nil {
		return fmt.Errorf("Failed to destroy the kubearmor relay gRPC client (%s)\n", err.Error())
	}
	kg.Printf("Destroyed kubearmor relay gRPC client")

	//Close BulkIndexer
	if err := ecl.bulkIndexer.Close(ecl.ctx); err != nil {
		kg.Errf("Unexpected error: %s", err)
	}

	ecl.cancel()

	kg.Printf("Stopped kubearmor receiver")
	time.Sleep(2 * time.Second)
	ecl.PrintBulkStats()
	return nil
}

func (ecl *ElasticsearchClient) PrintBulkStats() {
	biStats := ecl.bulkIndexer.Stats()
	println(strings.Repeat("▔", 80))

	dur := time.Since(start)

	if biStats.NumFailed > 0 {
		fmt.Printf(
			"Indexed [%s] documents with [%s] errors in %s (%s docs/sec)",
			humanize.Comma(int64(biStats.NumFlushed)),
			humanize.Comma(int64(biStats.NumFailed)),
			dur.Truncate(time.Millisecond),
			humanize.Comma(int64(1000.0/float64(dur/time.Millisecond)*float64(biStats.NumFlushed))),
		)
	} else {
		log.Printf(
			"Sucessfuly indexed [%s] documents in %s (%s docs/sec)",
			humanize.Comma(int64(biStats.NumFlushed)),
			dur.Truncate(time.Millisecond),
			humanize.Comma(int64(1000.0/float64(dur/time.Millisecond)*float64(biStats.NumFlushed))),
		)
	}
	println(strings.Repeat("▔", 80))

}
