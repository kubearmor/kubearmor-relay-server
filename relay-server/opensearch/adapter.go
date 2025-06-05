package opensearch

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	opensearch "github.com/opensearch-project/opensearch-go"
	"github.com/opensearch-project/opensearch-go/opensearchapi"
)

type OpenSearchClient struct {
	client     *opensearch.Client
	teleCh     chan interface{}
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	batchSize  int
	flushAfter time.Duration
	indexName  string
}

func NewOpenSearchClient(
	batchSize int,
	flushAfter time.Duration,
) (*OpenSearchClient, error) {

	osURL := os.Getenv("OS_URL")
	osUser := os.Getenv("OS_USERNAME")
	osPassword := os.Getenv("OS_PASSWORD")
	osCaCertPath := os.Getenv("OS_CA_CERT_PATH")

	osAllowInsecureTLS := false
	if os.Getenv("OS_ALLOW_INSECURE_TLS") != "" {
		osAllowInsecureTLS = true
	}
	osAlertsIndex := os.Getenv("OS_ALERTS_INDEX")

	cfg := opensearch.Config{
		Addresses: []string{osURL},
		Username:  osUser,
		Password:  osPassword,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: osAllowInsecureTLS,
			},
		},
	}

	if osCaCertPath != "" {
		caCertBytes, err := os.ReadFile(osCaCertPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA cert: %w", err)
		}
		cfg.CACert = caCertBytes
	}

	client, err := opensearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenSearch client: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	osc := &OpenSearchClient{
		client:     client,
		teleCh:     make(chan interface{}, 10000),
		ctx:        ctx,
		cancel:     cancel,
		batchSize:  batchSize,
		flushAfter: flushAfter,
		indexName:  osAlertsIndex,
	}
	log.Print("os client: ", osc)

	go osc.runBulkWorker()
	return osc, nil
}
func (osClient *OpenSearchClient) runBulkWorker() {

	osClient.wg.Add(1)
	defer osClient.wg.Done()
	ticker := time.NewTicker(osClient.flushAfter)
	defer ticker.Stop()

	var batch []string

	flush := func() {
		if len(batch) == 0 {
			return
		}

		var buf bytes.Buffer

		for _, entry := range batch {
			meta := fmt.Sprintf(`{ "index" : { "_index" : "%s", "_id" : "%s" } }%s`, osClient.indexName, uuid.New().String(), "\n")
			buf.WriteString(meta)
			buf.WriteString(entry + "\n")
		}

		req := opensearchapi.BulkRequest{
			Body: strings.NewReader(buf.String()),
		}
		resp, err := req.Do(osClient.ctx, osClient.client)
		if err != nil {
			log.Printf("Bulk indexing error: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.IsError() {
			log.Printf("Bulk request error: %s", resp.String())
		}
		batch = batch[:0]

	}

	for {
		select {
		case <-osClient.ctx.Done():
			return
		case alert := <-osClient.teleCh:
			alertJson, err := json.Marshal(alert)
			if err != nil {
				log.Printf("Error Marshalling json %v", err)
				continue
			}
			batch = append(batch, string(alertJson))
			if len(batch) > osClient.batchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}

}

func (osClient *OpenSearchClient) SendTelemetryToBuffer(alert interface{}) {
	select {
	case osClient.teleCh <- alert:
		log.Printf("received alert")
	default:
		log.Println("Warning: alert channel is full, dropping alert")
	}
}

func (osClient *OpenSearchClient) Stop() error {
	osClient.cancel()
	osClient.wg.Wait()
	log.Println("OpenSearch client stopped")
	return nil
}
