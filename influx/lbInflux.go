// Package influx provides a static client-side load balancer for influx
package influx

import (
	"github.com/influxdata/influxdb/client/v2"
	"github.com/go-kit/kit/sd/lb"
	"time"
	"github.com/go-kit/kit/sd"
	"context"
	"github.com/go-kit/kit/circuitbreaker"
	"github.com/sony/gobreaker"
	"github.com/go-kit/kit/ratelimit"
	jujuratelimit "github.com/juju/ratelimit"
	"github.com/pkg/errors"
	"github.com/go-kit/kit/endpoint"
)


// NamedClient extends the functionality of the regular influx client by adding a name to it,
// allowing easy identification of a faulty client when things go wrong
type NamedClient interface {
	GetName() string
	client.Client
}

type namedClient struct {
	name string
	client.Client
}

// Gets the name of a named client
func (nc namedClient) GetName() string {
	return nc.name
}

// Names an existing influx client
func NewNamedClient(name string, client client.Client) NamedClient {
	return &namedClient{
		name:name,
		Client: client,
	}
}

type loadBalancedClient struct {
	balancerEndpoint endpoint.Endpoint
	max              int
	timeout          time.Duration
}

type op int

const (
	ping  op = iota
	query
	write
	clean
)



type request struct {
	o op
	req interface{}
}

func (l *loadBalancedClient)Ping(timeout time.Duration) (time.Duration, string, error) {
	var response interface{}
	var err error
	{
		response, err = l.balancerEndpoint(context.Background(), request{
			ping,
			timeout,
		})
		if err != nil {
			return 0,"",errors.Wrap(err, "ping failed :")
		}
	}
	var pingResponse pingResp = response.(pingResp)
	return pingResponse.t, pingResponse.s, nil
}

type pingResp struct {
	t time.Duration
	s string
}

func (l *loadBalancedClient)Write(bp client.BatchPoints) error {
	var err error
	_, err = l.balancerEndpoint(context.Background(), request{
		write,
		bp,
	})
	return err
}

func (l *loadBalancedClient)Query(q client.Query) (*client.Response, error) {
	var response interface{}
	var err error
	response, err = l.balancerEndpoint(context.Background(), request{
		query,
		q,
	})
	if err != nil {
		return nil, err
	}
	return response.(*client.Response), nil
}

// Closes all current connections on all balanced clients
func (l *loadBalancedClient)Close() error {
	var err error
	_, err = l.balancerEndpoint(context.Background(), request{
		clean,
		nil,
	})
	return err
}

// An error aggregation is required for close operations, which ripple through all balanced clients.
// It simply aggregates all the errors produced by all clients by name. (Experimental! Could be subject to change)
type ErrorAggregation struct {
	AggrError map[string]error
}

// Returns a string representation of all errors each on a new line
func (aggr ErrorAggregation) Error() (err string) {
	for _,e := range aggr.AggrError {
		err += e.Error() + "\n"
	}
	return err
}

// Returns a new aggregator with the added (name, error)
func (aggr ErrorAggregation) Add(name string, err error) ErrorAggregation {
	var agError map[string]error = aggr.AggrError
	agError[name] = err
	return ErrorAggregation{
		AggrError:agError,
	}
}

// NewLoadBalancerClient creates a round-robin balancer of the clients passed in parameter.
// The balancer behaves identically to a regular influx client.
// `timeout` specifies the timeout before giving up on a client and retrying,
// `max` is the max number of retries before failing,
// `qpsLimit` is the hard limit on the number of queries per second the load balancer will accept,
// `qpsQueued` is the number of queries per second before the load balancer starts queuing the queries.
// The load balancer implements the breaker pattern (Not flexible yet. TODO)
func NewLoadBalancedClient(niClients []NamedClient, timeout time.Duration, max int, qpsLimit int, qpsQueued int) (client.Client) {
	var endpoints sd.FixedSubscriber
	for _, niClient := range niClients {
		var influxEndpoint endpoint.Endpoint
		{
			influxEndpoint = func(niClient NamedClient) endpoint.Endpoint {
				return func(ctx context.Context, i interface{}) (interface{}, error) {
					var iReq request = i.(request)
					switch iReq.o {
					case ping:
						var t time.Duration
						var s string
						{
							var err error
							t, s, err = niClient.Ping(iReq.req.(time.Duration))
							if err != nil {
								return nil, errors.Wrap(err, "ping failed on client "+niClient.GetName())
							}
						}
						return pingResp{
							t: t,
							s: s,
						}, nil
					case write:
						var err error
						err = niClient.Write(iReq.req.(client.BatchPoints))
						if err != nil {
							return nil, errors.Wrap(err, "write failed on client "+niClient.GetName())
						}
						return nil, nil
					case query:
						var r *client.Response
						{
							var err error
							r, err = niClient.Query(iReq.req.(client.Query))
							if err != nil {
								return nil, errors.Wrap(err, "query failed on client "+niClient.GetName())
							}
						}
						return r, nil
					default: //close
						var err error
						err = niClient.Close()
						if err != nil {
							return niClient.GetName(), errors.Wrap(err, "close failed on client "+niClient.GetName())
						}
						return nil, nil
					}
				}
			}(niClient)
		}
		endpoints = append(endpoints, influxEndpoint)
	}

	var balancerEndpoint endpoint.Endpoint
	var loadBalancer lb.Balancer
	{
		loadBalancer = lb.NewRoundRobin(endpoints)
		balancerEndpoint = func(ctx context.Context, req interface{}) (interface{}, error) {
			iReq := req.(request)
			if iReq.o == clean {
				var aggrErrors ErrorAggregation
				var aggrErrors_witness bool
				var err error
				{
					aggrErrors = ErrorAggregation{make(map[string]error)}
					aggrErrors_witness = true
					for _, e := range endpoints {
						var name interface{}
						{
							name, err = e(ctx, req)
							if err != nil {
								if aggrErrors_witness {
									aggrErrors_witness = false
								}
								aggrErrors = aggrErrors.Add(name.(string), err)
							}
						}
					}
					if !aggrErrors_witness {
						err = aggrErrors
					}
				}
				return nil,err
			}
			return lb.Retry(max, timeout, loadBalancer)(ctx, req)
		}
		balancerEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{}))(balancerEndpoint)
		balancerEndpoint = ratelimit.NewTokenBucketThrottler(jujuratelimit.NewBucketWithRate(float64(qpsQueued), int64(qpsQueued)), time.Sleep)(balancerEndpoint)
		balancerEndpoint = ratelimit.NewTokenBucketLimiter(jujuratelimit.NewBucketWithRate(float64(qpsLimit), int64(qpsLimit)))(balancerEndpoint)
	}

	return &loadBalancedClient{
		balancerEndpoint: balancerEndpoint,
	}
}