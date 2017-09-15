package influx

import (
	"github.com/influxdata/influxdb/client/v2"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/sd/lb"
	"github.com/go-kit/kit/circuitbreaker"
	"github.com/sony/gobreaker"
	"time"
	"github.com/go-kit/kit/ratelimit"
	jujuratelimit "github.com/juju/ratelimit"
	"context"
	"github.com/pkg/errors"
	"github.com/go-kit/kit/sd"
	"log"
)

type loadBalancedClient struct {
	balancerEndpoint endpoint.Endpoint
	max              int
	timeout          time.Duration
}

/*
Middleware that applies to the whole client
e.g. throttler, limiter, breaker...
 */
type Middleware = endpoint.Middleware

/*
Middleware acting differently for each function
e.g. logging.
 */
type InfluxMiddleware func(client client.Client) client.Client

type influxLoggingMiddleware struct {
	logger log.Logger
	next client.Client
}

type influxEndpoints struct {
	influxEndpoint endpoint.Endpoint
}

func NewTokenBucketThrottler(rate int64, capacity int64) Middleware{
	return ratelimit.NewTokenBucketThrottler(jujuratelimit.NewBucketWithRate(float64(rate), int64(capacity)), time.Sleep)
}

func NewTokenBucketLimiter(rate int64, capacity int64) Middleware{
	return ratelimit.NewTokenBucketLimiter(jujuratelimit.NewBucketWithRate(float64(rate), int64(capacity)))
}

func NewCircuitBreaker() Middleware {
	return circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{}))
}

// NewLoadBalancerClient creates a round-robin balancer of the clients passed in parameter.
// The balancer behaves identically to a regular influx client.
// `timeout` specifies the timeout before giving up on a client and retrying,
// `max` is the max number of retries before failing,
// `qpsLimit` is the hard limit on the number of queries per second the load balancer will accept,
// `qpsQueued` is the number of queries per second before the load balancer starts queuing the queries.
// The load balancer implements the breaker pattern (Not flexible yet. TODO)
func NewLoadBalancedClient(niClients []client.Client,
	max int, timeout time.Duration, //Load Balancer
		mids ...Middleware) (client.Client) {
	var endpoints sd.FixedEndpointer
	for _, niClient := range niClients {
		var influxEndpoint endpoint.Endpoint
		{
			influxEndpoint = func(niClient client.Client) endpoint.Endpoint {
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
								return nil, err
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
							return nil, err
						}
						return nil, nil
					case query:
						var r *client.Response
						{
							var err error
							r, err = niClient.Query(iReq.req.(client.Query))
							if err != nil {
								return nil, err
							}
						}
						return r, nil
					default: //close
						var err error
						err = niClient.Close()
						return nil, err
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
					aggrErrors = ErrorAggregation{}
					aggrErrors_witness = true
					for _, e := range endpoints {
						_, err = e(ctx, req)
						if err != nil {
							aggrErrors_witness = false
							aggrErrors = aggrErrors.Add(err)
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
		for _, m := range mids {
			balancerEndpoint = m(balancerEndpoint)
		}
	}

	return &influxEndpoints{
		influxEndpoint: balancerEndpoint,
	}
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

func (l *influxEndpoints)Ping(timeout time.Duration) (time.Duration, string, error) {
	var response interface{}
	var err error
	{
		response, err = l.influxEndpoint(context.Background(), request{
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

func (l *influxEndpoints)Write(bp client.BatchPoints) error {
	var err error
	_, err = l.influxEndpoint(context.Background(), request{
		write,
		bp,
	})
	return err
}

func (l *influxEndpoints)Query(q client.Query) (*client.Response, error) {
	var response interface{}
	var err error
	response, err = l.influxEndpoint(context.Background(), request{
		query,
		q,
	})
	if err != nil {
		return nil, err
	}
	return response.(*client.Response), nil
}

// Closes all current connections on all balanced clients
func (l *influxEndpoints)Close() error {
	var err error
	_, err = l.influxEndpoint(context.Background(), request{
		clean,
		nil,
	})
	return err
}

// An error aggregation is required for close operations, which ripple through all balanced clients.
// It simply aggregates all the errors produced by all clients by name. (Experimental! Could be subject to change)
type ErrorAggregation struct {
	errors []error
}

// Returns a string representation of all errors each on a new line
func (aggr ErrorAggregation) Error() (err string) {
	for _,e := range aggr.errors {
		err += e.Error() + "\n"
	}
	return err
}

// Returns a new aggregator with the added (name, error)
func (aggr ErrorAggregation) Add(err error) ErrorAggregation {
	aggr.errors = append(aggr.errors, err)
	return aggr
}