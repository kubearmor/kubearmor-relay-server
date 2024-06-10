package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync/atomic"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/dustin/go-humanize"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esutil"
	"github.com/google/uuid"
	kg "github.com/kubearmor/kubearmor-relay-server/relay-server/log"

	kl "github.com/kubearmor/kubearmor-relay-server/relay-server/common"

	// kif "github.com/kubearmor/kubearmor-relay-server/relay-server/informers"
	"github.com/kubearmor/kubearmor-relay-server/relay-server/server"
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
	alertCh     chan interface{}
	logCh       chan interface{}
	// client      *kif.Client
}

// NewElasticsearchClient creates a new Elasticsearch client with the given Elasticsearch URL
// and kubearmor LogClient with endpoint. It has a retry mechanism for certain HTTP status codes and a backoff function for retry delays.
// It then creates a new NewBulkIndexer with the esClient
func NewElasticsearchClient(esURL, Endpoint string) (*ElasticsearchClient, error) {
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
	bi, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Client:        esClient,         // The Elasticsearch client
		FlushBytes:    1000000,          // The flush threshold in bytes [1mb]
		FlushInterval: 30 * time.Second, // The periodic flush interval [30 secs]
	})
	if err != nil {
		log.Fatalf("Error creating the indexer: %s", err)
	}
	alertCh := make(chan interface{}, 10000)
	logCh := make(chan interface{}, 10000)
	kaClient := server.NewClient(Endpoint)

	// k8sClient := kif.GetK8sClient()
	// cc := &kif.ClusterCache{
	//
	// 	mu: &sync.RWMutex{},
	//
	// 	ipPodCache: make(map[string]PodServiceInfo),
	// }
	// client := &kif.Client{
	// 	k8sClient:      k8sClient,
	// 	ClusterIPCache: cc,
	// }

	// client := kif.InitializeClient()
	// go kif.StartInformers(client)

	// TODO: remove this informers
	return &ElasticsearchClient{kaClient: kaClient, bulkIndexer: bi, esClient: esClient, alertCh: alertCh, logCh: logCh}, nil
}

// bulkIndex takes an interface and index name and adds the data to the Elasticsearch bulk indexer.
// The bulk indexer flushes after the FlushBytes or FlushInterval thresholds are reached.
// The method generates a UUID as the document ID and includes success and failure callbacks for each item added to the bulk indexer.
func (ecl *ElasticsearchClient) bulkIndex(a interface{}, index string) {
	countEntered++
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

	if err != nil {
		log.Fatalf("Error adding items to bulk indexer: %s", err)
	}
}

// Start starts the Elasticsearch client by performing a health check on the gRPC server
// and starting goroutines to consume messages from the alert channel and bulk index them.
// The method starts a goroutine for each stream and waits for messages to be received.
// Additional goroutines consume alert from the alert channel and bulk index them.
func (ecl *ElasticsearchClient) Start() error {
	start = time.Now()
	client := ecl.kaClient
	ecl.ctx, ecl.cancel = context.WithCancel(context.Background())

	// do healthcheck
	if ok := client.DoHealthCheck(); !ok {
		return fmt.Errorf("failed to check the liveness of the gRPC server")
	}
	kg.Printf("Checked the liveness of the gRPC server")

	client.WgServer.Go(func() error {
		for client.Running {
			res, err := client.AlertStream.Recv()
			if err != nil {
				return fmt.Errorf("failed to receive an alert (%s) %s", client.Server, err)
			}
			tel, _ := json.Marshal(res)
			fmt.Printf("%s\n", string(tel))

			select {
			case ecl.alertCh <- res:
			case <-client.Context.Done():
				// The context is over, stop processing results
				return nil
			default:
				//not able to add it to Log buffer
			}
		}
		return nil
	})

	for i := 0; i < 5; i++ {
		go func() {
			for {
				select {
				case alert := <-ecl.alertCh:
					ecl.bulkIndex(alert, "alert")
				case <-ecl.ctx.Done():
					close(ecl.alertCh)
					return
				}
			}
		}()
	}

	client.WgServer.Go(func() error {
		for client.Running {
			res, err := client.LogStream.Recv()
			if err != nil {
				kg.Warnf("Failed to receive an log (%s)", client.Server)
				break
			}

			if containsKprobe := strings.Contains(res.Data, "kprobe"); containsKprobe {

				resourceMap := kl.Extractdata(res.GetResource())
				remoteIP := resourceMap["remoteip"]
				podserviceInfo, found := ecl.kaClient.Ifclient.ClusterIPCache.Get(remoteIP)

				if found {
					switch podserviceInfo.Type {
					case "POD":
						resource := res.GetResource() + fmt.Sprintf(" hostname=%s podname=%s namespace=%s", podserviceInfo.DeploymentName, podserviceInfo.PodName, podserviceInfo.NamespaceName)
						data := res.GetData() + fmt.Sprintf(" ownertype=pod")
						res.Data = data

						res.Resource = resource
						// kg.Printf("logData:%s", res.Data)
						break
					case "SERVICE":
						resource := res.GetResource() + fmt.Sprintf(" hostname=%s servicename=%s namespace=%s", podserviceInfo.DeploymentName, podserviceInfo.ServiceName, podserviceInfo.NamespaceName)

						data := res.GetData() + fmt.Sprintf(" ownertype=service")
						res.Data = data
						res.Resource = resource
						// kg.Printf("logData:%s", res.Data)

						break
					}
				}

			}
			tel, _ := json.Marshal(res)
			fmt.Printf("%s\n", string(tel))
			ecl.logCh <- res
		}
		return nil
	})

	for i := 0; i < 5; i++ {
		go func() {
			for {
				select {
				case log := <-ecl.logCh:
					ecl.bulkIndex(log, "log")
				case <-ecl.ctx.Done():
					close(ecl.logCh)
					return
				}
			}
		}()
	}
	return nil
}

// Stop stops the Elasticsearch client and performs necessary cleanup operations.
// It stops the Kubearmor Relay client, closes the BulkIndexer and cancels the context.
func (ecl *ElasticsearchClient) Stop() error {
	logClient := ecl.kaClient
	logClient.Running = false
	time.Sleep(2 * time.Second)

	//Destoy KubeArmor Relay Client
	if err := logClient.DestroyClient(); err != nil {
		return fmt.Errorf("failed to destroy the kubearmor relay gRPC client (%s)", err.Error())
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

// PrintBulkStats prints data on the bulk indexing process, including the number of indexed documents,
// the number of errors, and the indexing rate , after elasticsearch client stops
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
