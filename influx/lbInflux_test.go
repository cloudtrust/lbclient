package influx

import (
	"fmt"
	"testing"
	"time"
	"strconv"
	"github.com/influxdata/influxdb/client/v2"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/assert"
	"sync"
)

func TestPing(b *testing.T) {
	var clients []NamedClient
	for i := 0; i<3; i++ {
		var name string = "client "+strconv.Itoa(i)
		var addr string = "http://localhost:8086"
		var influxClient client.Client
		{
			var err error
			influxClient, err = client.NewHTTPClient(client.HTTPConfig{
				Addr:     addr,
				Username: "",
				Password: ""},
			)
			require.Nil(b, err, "Couldn't create client" + name)
		}
		var namedClient NamedClient = NewNamedClient(name, influxClient)
		clients = append(clients, namedClient)
	}
	var lbi client.Client = NewLoadBalancedClient(clients, 10 * time.Second, 1, 3, 1)
	defer lbi.Close()
	{
		var err error
		_, _, err = lbi.Ping(0)
		assert.Nil(b, err, "Error while pinging!")
	}
}

func TestLimiter(b *testing.T) {
	var clients []NamedClient
	for i := 0; i<3; i++ {
		var name string = "client "+strconv.Itoa(i)
		var addr string = "http://localhost:8086"
		var influxClient client.Client
		{
			var err error
			influxClient, err = client.NewHTTPClient(client.HTTPConfig{
				Addr:     addr,
				Username: "",
				Password: ""},
			)
			require.Nil(b, err, "Couldn't create client" + name)
		}
		var namedClient NamedClient = NewNamedClient(name, influxClient)
		clients = append(clients, namedClient)
	}
	var qpsLimit int = 3
	var lbi client.Client = NewLoadBalancedClient(clients, 10 * time.Second, 1, qpsLimit, 1)
	defer lbi.Close()
	var wg sync.WaitGroup
	wg.Add(5)
	var numtest int = 5
	var errnum int = 0
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
	assert.Equal(b, errnum, numtest - qpsLimit, "Wrong number of errors happened!")
}

func TestQueryWrite(b *testing.T) {
	var clients []NamedClient
	for i := 0; i<3; i++ {
		var name string = "client "+strconv.Itoa(i)
		var addr string = "http://localhost:8086"
		var influxClient client.Client
		{
			var err error
			influxClient, err = client.NewHTTPClient(client.HTTPConfig{
				Addr:     addr,
				Username: "",
				Password: ""},
			)
			require.Nil(b, err, "Couldn't create client" + name)
		}
		var namedClient NamedClient = NewNamedClient(name, influxClient)
		clients = append(clients, namedClient)
	}
	var database string = "supertesting"
	var lbi client.Client = NewLoadBalancedClient(clients, 10 * time.Second, 1, 3, 1)
	defer lbi.Close()
	//Create DB
	{
		var err error
		_, err = lbi.Query(client.NewQuery(fmt.Sprintf("CREATE DATABASE %s", database), "", ""))
		require.Nil(b, err, "query failure ", err)
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
	//Remove DB
	{
		var err error
		_, err = lbi.Query(client.NewQuery(fmt.Sprintf("DROP DATABASE %s", database), "", ""))
		require.Nil(b, err, "query failure ", err)
	}
}

func TestBadPing(b *testing.T) {
	var clients []NamedClient
	for i := 0; i<3; i++ {
		var name string = "client "+strconv.Itoa(i)
		var addr string = "http://localhost:22"
		var influxClient client.Client
		{
			var err error
			influxClient, err = client.NewHTTPClient(client.HTTPConfig{
				Addr:     addr,
				Username: "",
				Password: ""},
			)
			require.Nil(b, err, "Couldn't create client" + name)
		}
		var namedClient NamedClient = NewNamedClient(name, influxClient)
		clients = append(clients, namedClient)
	}
	var lbi client.Client = NewLoadBalancedClient(clients, 10 * time.Second, 1, 3, 1)
	defer lbi.Close()
	{
		var err error
		_, _, err = lbi.Ping(1)
		require.NotNil(b, err, err.Error())
	}
}

func TestBadQuery(b *testing.T) {
	var clients []NamedClient
	for i := 0; i<3; i++ {
		var name string = "client "+strconv.Itoa(i)
		var addr string = "http://localhost:22"
		var influxClient client.Client
		{
			var err error
			influxClient, err = client.NewHTTPClient(client.HTTPConfig{
				Addr:     addr,
				Username: "",
				Password: ""},
			)
			require.Nil(b, err, "Couldn't create client" + name)
		}
		var namedClient NamedClient = NewNamedClient(name, influxClient)
		clients = append(clients, namedClient)
	}
	var database string = "testing"
	var lbi client.Client = NewLoadBalancedClient(clients, 10 * time.Second, 1, 3, 1)
	defer lbi.Close()
	{
		var err error
		_, err = lbi.Query(client.NewQuery(fmt.Sprintf("Select * from %s", database), "", ""))
		require.NotNil(b, err, err.Error())
	}
}

func TestBadWrite(b *testing.T) {
	var clients []NamedClient
	for i := 0; i<3; i++ {
		var name string = "client "+strconv.Itoa(i)
		var addr string = "http://localhost:22"
		var influxClient client.Client
		{
			var err error
			influxClient, err = client.NewHTTPClient(client.HTTPConfig{
				Addr:     addr,
				Username: "",
				Password: ""},
			)
			require.Nil(b, err, "Couldn't create client" + name)
		}
		var namedClient NamedClient = NewNamedClient(name, influxClient)
		clients = append(clients, namedClient)
	}
	var database string = "supertesting"
	var lbi client.Client = NewLoadBalancedClient(clients, 10 * time.Second, 1, 3, 1)
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



type badCloseClient struct {
	name string
	succeed bool
}

func (b badCloseClient)GetName() string {
	return b.name
}

func (b badCloseClient)Ping(timeout time.Duration) (time.Duration, string, error){
	return time.Second,"",nil
}

func (b badCloseClient)Write(bp client.BatchPoints) error{
	return nil
}

func (b badCloseClient)Query(q client.Query) (*client.Response, error){
	return nil,nil
}

func (b badCloseClient)Close() error{
	if b.succeed {
		return nil
	}
	return errors.New("Completely Failed!")
}

func TestBadClose(b *testing.T) {
	var clients []NamedClient
	var badClient NamedClient = badCloseClient{"client0", false}
	clients = append(clients, badClient)
	var lbi client.Client = NewLoadBalancedClient(clients, 10*time.Second, 1, 3, 1)
	var err = lbi.Close()
	assert.NotNil(b, err, "close should fail", err)
	assert.Equal(b, err.Error(), "close failed on client client0: Completely Failed!\n")
}