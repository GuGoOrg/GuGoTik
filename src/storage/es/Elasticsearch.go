package es

import (
	"GuGoTik/src/constant/config"
	"log"

	es "github.com/elastic/go-elasticsearch/v7"
)

var EsClient *es.Client

func init() {
	cfg := es.Config{
		Addresses: []string{
			config.EnvCfg.ElasticsearchUrl,
		},
	}
	var err error
	EsClient, err = es.NewClient(cfg)
	if err != nil {
		log.Fatalf("elasticsearch.NewClient: %v", err)
	}

	_, err = EsClient.Info()
	if err != nil {
		log.Fatalf("Error getting response: %s", err)
	}

	_, err = EsClient.API.Indices.Create("Message")

	if err != nil {
		log.Fatalf("create index error: %s", err)
	}

}
