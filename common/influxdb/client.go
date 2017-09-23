package influxdb

import (
	"os"
	"time"

	"github.com/bytearena/bytearena/common/utils"

	"github.com/influxdata/influxdb/client/v2"
)

type Client struct {
	isStub bool

	batchpointsClient client.BatchPoints
	influxdbClient    client.Client
	tickerChannel     *time.Ticker
}

func createHttpClient(addr string) (client.Client, error) {
	return client.NewHTTPClient(client.HTTPConfig{
		Addr: addr,
	})
}

func createBatchPoints(db string) (client.BatchPoints, error) {
	return client.NewBatchPoints(client.BatchPointsConfig{
		Database: db,
	})
}

func NewClient() (*Client, error) {
	influxdbAddr := os.Getenv("INFLUXDB_ADDR")
	influxdbDb := os.Getenv("INFLUXDB_DB")

	tickerChannel := time.NewTicker(5 * time.Second)

	stubClient := &Client{
		isStub: true,

		tickerChannel: tickerChannel,
	}

	if influxdbAddr == "" && influxdbDb == "" {

		utils.Debug("influxdb", "No client has been configured")
		return stubClient, nil
	} else {
		influxdbClient, clientErr := createHttpClient(influxdbAddr)

		if clientErr != nil {
			return stubClient, clientErr
		}

		batchpointsClient, batchpointsErr := createBatchPoints(influxdbDb)

		if batchpointsErr != nil {
			return stubClient, batchpointsErr
		}

		utils.Debug("influxdb", "Influxdb reporting is enabled")

		return &Client{
			isStub: false,

			influxdbClient:    influxdbClient,
			batchpointsClient: batchpointsClient,
			tickerChannel:     tickerChannel,
		}, nil
	}
}

func (c *Client) WriteAppMetric(name, app string, fields map[string]interface{}) {
	if c.isStub {
		// TODO(sven): do something? Print in the console?
		return
	}

	tags := map[string]string{"app": app}

	pt, err := client.NewPoint(name, tags, fields, time.Now())

	if err != nil {
		panic(err.Error())
	}

	c.batchpointsClient.AddPoint(pt)
	c.influxdbClient.Write(c.batchpointsClient)
}

func (c *Client) Loop(fn func()) {
	go func() {
		for {
			<-c.tickerChannel.C

			fn()
		}
	}()
}

func (c *Client) TearDown() {
	c.tickerChannel.Stop()
}
