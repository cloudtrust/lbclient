package influx

import (
	"fmt"
	"testing"
	"time"
	"github.com/influxdata/influxdb/client/v2"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/assert"
	"sync"
)

var queryResponse = &client.Response{}

type MockClient struct {
	pingSuccess bool
	writeSuccess bool
	querySuccess bool
	closeSuccess bool
}
func (m *MockClient)Ping(timeout time.Duration) (time.Duration, string, error) {
	if m.pingSuccess {
		return 10, "success", nil
	}
	return 0, "", errors.New("Catastrophic failure")
}

func (m *MockClient)Write(bp client.BatchPoints) error {
	if m.writeSuccess {
		return nil
	}
	return errors.New("Catastrophic failure")
}

func (m *MockClient)Query(q client.Query) (*client.Response, error) {
	if m.querySuccess {
		return queryResponse, nil
	}
	return nil, errors.New("Catastrophic failure")
}
func (m *MockClient)Close() error {
	if m.closeSuccess {
		return nil
	}
	return errors.New("Catastrophic failure")
}

func TestPing(b *testing.T) {
	var clients []client.Client
	for i := 0; i<3; i++ {
		var influxClient client.Client = &MockClient{true, true, false, true}
		clients = append(clients, influxClient)
	}
	var lbi client.Client = NewLoadBalancedClient(clients,1, 10 * time.Second, )
	defer lbi.Close()
	{
		var err error
		var s string
		var t time.Duration
		t, s, err = lbi.Ping(0)
		require.Nil(b, err, "Error while pinging!")
		assert.Equal(b, t, time.Duration(10))
		assert.Equal(b, s, "success")
	}
}

func TestLimiter(b *testing.T) {
	var clients []client.Client
	for i := 0; i<3; i++ {
		var influxClient client.Client = &MockClient{true, true, false, true}
		clients = append(clients, influxClient)
	}
	var qpsLimit float64 = 3
	var capacity int64 = 3
	var limiter = NewTokenBucketLimiter(qpsLimit, capacity)
	var lbi client.Client = NewLoadBalancedClient(clients,1, 10 * time.Second, limiter)

	defer lbi.Close()
	var wg sync.WaitGroup
	wg.Add(5)
	var numtest int64 = 5
	var errnum int64 = 0
	for i:=0; i<5; i++ {
		go func() {
			defer wg.Done()
			var err error
			_, _, err = lbi.Ping(0)
			if err != nil {
				errnum++
			}
		}()
	}
	wg.Wait()
	assert.Equal(b, errnum, numtest - capacity, "Wrong number of errors happened!")
}

func TestQueryWrite(b *testing.T) {
	var clients []client.Client
	for i := 0; i<3; i++ {
		var influxClient client.Client = &MockClient{true, true, true, true}
		clients = append(clients, influxClient)
	}
	var database string = "supertesting"
	var lbi client.Client = NewLoadBalancedClient(clients,1, 10 * time.Second, )
	defer lbi.Close()
	//Create DB
	{
		var err error
		var qResp *client.Response
		qResp, err = lbi.Query(client.NewQuery(fmt.Sprintf("CREATE DATABASE %s", database), "", ""))
		require.Nil(b, err, "query failure ", err)
		assert.Equal(b, qResp, queryResponse, "Not expected return!!")
	}

	//Setup write to DB
	var batchPoints client.BatchPoints
	{
		var err error
		batchPoints, err = client.NewBatchPoints(client.BatchPointsConfig{
			Database:  database,
			Precision: "s",
		})
		require.Nil(b, err, "New Batch Points failure ", err)
		var tags map[string]string = map[string]string {"tag": "hello",}
		var fields map[string]interface{} = map[string]interface{} {"field":"world",}
		var point *client.Point
		{
			var err error
			point, err = client.NewPoint("testing", tags, fields, time.Now())
			require.Nil(b, err, "New Point failure ", err)
		}
		batchPoints.AddPoint(point)
	}
	//Write to DB
	{
		var err error
		err = lbi.Write(batchPoints)
		assert.Nil(b, err, "Write failed")
	}
}

func TestBadPing(b *testing.T) {
	var clients []client.Client
	for i := 0; i<3; i++ {
		var influxClient client.Client = &MockClient{false, true, false, true}
		clients = append(clients, influxClient)
	}
	var lbi client.Client = NewLoadBalancedClient(clients,1, 10 * time.Second, )
	defer lbi.Close()
	{
		var err error
		_, _, err = lbi.Ping(1)
		require.NotNil(b, err, err.Error())
	}
}

func TestBadQuery(b *testing.T) {
	var clients []client.Client
	for i := 0; i<3; i++ {
		var influxClient client.Client = &MockClient{true, true, false, true}
		clients = append(clients, influxClient)
	}
	var database string = "testing"
	var lbi client.Client = NewLoadBalancedClient(clients,1, 10 * time.Second, )
	defer lbi.Close()
	{
		var err error
		_, err = lbi.Query(client.NewQuery(fmt.Sprintf("Select * from %s", database), "", ""))
		require.NotNil(b, err, err.Error())
	}
}

func TestBadWrite(b *testing.T) {
	var clients []client.Client
	for i := 0; i<3; i++ {
		var influxClient client.Client = &MockClient{true, false, true, true}
		clients = append(clients, influxClient)
	}
	var database string = "supertesting"
	var lbi client.Client = NewLoadBalancedClient(clients, 1, 0 * time.Second, )
	defer lbi.Close()
	var batchPoints client.BatchPoints
	{
		var err error
		batchPoints, err = client.NewBatchPoints(client.BatchPointsConfig{
			Database:  database,
			Precision: "s",
		})
		require.Nil(b, err, "New Batch Points failure ", err)
		var tags map[string]string = map[string]string {"tag": "hello",}
		var fields map[string]interface{} = map[string]interface{} {"field":"world",}
		var point *client.Point
		{
			var err error
			point, err = client.NewPoint("testing", tags, fields, time.Now())
			require.Nil(b, err, "New Point failure ", err)
		}
		batchPoints.AddPoint(point)
	}
	{
		var err = lbi.Write(batchPoints)
		require.NotNil(b, err, "write Failure ", err)
	}
}


func TestBadClose(b *testing.T) {
	var clients []client.Client
	var badClient client.Client = &MockClient{true, true, true, false}
	clients = append(clients, badClient)
	var lbi client.Client = NewLoadBalancedClient(clients, 1, 10*time.Second,)
	var err = lbi.Close()
	require.NotNil(b, err, "close should fail", err)
}