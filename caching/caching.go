package caching

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/teris-io/shortid"
)

// CreateID creates a unique ID for operators in this system.
func CreateID() (string, error) {

	sid, err := shortid.New(1, shortid.DefaultABC, uint64(time.Now().Unix()))
	if err != nil {
		return "", err
	}

	return sid.Generate()
}

// RedisPool is an interface that allows us to swap in an mock for testing cache
// code.
type RedisPool interface {
	Get() redis.Conn
}

// ErrCacheMiss error indicates that an item is not in the cache
var ErrCacheMiss = fmt.Errorf("item is not in cache")

// NewCache returns an initialized cache ready to go.
func NewCache(redisHost, redisPort string, debug bool) (*Cache, error) {
	c := &Cache{}
	pool := c.InitPool(redisHost, redisPort)
	c.redisPool = pool
	c.debug = debug
	return c, nil
}

// Cache abstracts all of the operations of caching for the application
type Cache struct {
	// redisPool *redis.Pool
	redisPool RedisPool
	enabled   bool
	debug     bool
}

func (c *Cache) log(msg string) {
	if c.debug {
		log.Printf("Cache     : %s\n", msg)
	}
}

// InitPool starts the cache off
func (c Cache) InitPool(redisHost, redisPort string) RedisPool {
	redisAddr := fmt.Sprintf("%s:%s", redisHost, redisPort)
	msg := fmt.Sprintf("Initialized Redis at %s", redisAddr)
	c.log(msg)
	const maxConnections = 10

	pool := redis.NewPool(func() (redis.Conn, error) {
		return redis.Dial("tcp", redisAddr)
	}, maxConnections)

	return pool
}

// Clear removes all items from the cache.
func (c Cache) Clear() error {
	conn := c.redisPool.Get()
	defer conn.Close()

	if _, err := conn.Do("FLUSHALL"); err != nil {
		return err
	}
	return nil
}

// Record a hit in redis
func (c Cache) Record(instance Instance) error {

	conn := c.redisPool.Get()
	defer conn.Close()

	conn.Send("MULTI")

	if err := conn.Send("HSET", "index", instance.ID, instance.Env); err != nil {
		return err
	}

	if err := conn.Send("INCR", instance.ID); err != nil {
		return err
	}

	if _, err := conn.Do("EXEC"); err != nil {
		return err
	}

	return nil
}

// RegisterGenerator registers a load producing node.
func (c Cache) RegisterGenerator(nodeID, ip string, active bool) error {

	conn := c.redisPool.Get()
	defer conn.Close()

	node := Generator{nodeID, ip, active}

	nodestr, err := node.JSON()
	if err != nil {
		return err
	}

	if _, err := conn.Do("HSET", "loadnodes", ip, nodestr); err != nil {
		return fmt.Errorf("cannot set loadnodes in redis: %s", err)
	}

	return nil
}

// RegisterReceiver registers a receiver endpoint.
func (c Cache) RegisterReceiver(env, endpoint string) error {

	conn := c.redisPool.Get()
	defer conn.Close()

	r := Receiver{env, endpoint}

	rstr, err := r.JSON()
	if err != nil {
		return err
	}

	if _, err := conn.Do("HSET", "receivers", endpoint, rstr); err != nil {
		return err
	}

	return nil
}

// InstanceReport returns the whole collection of all of the instances
func (c Cache) InstanceReport() (InstanceReport, error) {
	index := InstanceReport{}
	keys := []interface{}{}
	intkeys := []string{}

	conn := c.redisPool.Get()
	defer conn.Close()

	s, err := redis.StringMap(conn.Do("HGETALL", "index"))
	if err == redis.ErrNil {
		return index, ErrCacheMiss
	} else if err != nil {
		return index, err
	}

	for id, env := range s {
		ins := Instance{id, env, 0}
		index[id] = ins
		keys = append(keys, id)
		intkeys = append(intkeys, id)
	}

	counts, err := redis.Strings(conn.Do("MGET", keys...))
	if err == redis.ErrNil {
		return index, ErrCacheMiss
	} else if err != nil {
		return index, err
	}

	for idx, count := range counts {

		id := intkeys[idx]

		ins, ok := index[id]
		if !ok {
			return index, fmt.Errorf("could not get instance from index")
		}

		c, err := strconv.Atoi(count)
		if err != nil {
			return index, fmt.Errorf("could not get count for instance")
		}

		ins.Count = c
		index[id] = ins

	}

	return index, nil
}

// Generators returns the whole collection of all of the load nodes
func (c Cache) Generators() (Generators, error) {
	keys := Generators{}

	conn := c.redisPool.Get()
	defer conn.Close()

	s, err := redis.StringMap(conn.Do("HGETALL", "loadnodes"))
	if err == redis.ErrNil {
		return keys, ErrCacheMiss
	} else if err != nil {
		return keys, err
	}

	for _, v := range s {
		node := Generator{}
		err := node.Load(v)
		if err != nil {
			return keys, err
		}

		keys = append(keys, node)
	}

	return keys, nil
}

// Receivers returns the whole collection of all of the receivers
func (c Cache) Receivers() (Receivers, error) {
	keys := Receivers{}

	conn := c.redisPool.Get()
	defer conn.Close()

	s, err := redis.StringMap(conn.Do("HGETALL", "receivers"))
	if err == redis.ErrNil {
		return keys, ErrCacheMiss
	} else if err != nil {
		return keys, err
	}

	for _, v := range s {
		r := Receiver{}
		err := r.Load(v)
		if err != nil {
			return keys, err
		}

		keys = append(keys, r)
	}

	return keys, nil
}

func (c Cache) calcRates(n string, cc string, count int) (string, string, error) {
	nInt, err := strconv.Atoi(n)
	if err != nil {
		return "", "", errors.New("Could not get valid value for `n`: " + n)
	}

	cInt, err := strconv.Atoi(cc)
	if err != nil {
		return "", "", fmt.Errorf("could not get valid value for env variable `TARGET_QPS`: %s", cc)
	}

	nodeN := nInt / count
	nodeC := cInt / count

	// Ensures that C never exceeds N cause if that happens Apache Bench fails.
	if nodeC > nodeN {
		nodeC = nodeN
	}
	return strconv.Itoa(nodeN), strconv.Itoa(nodeC), nil
}

// Distribute splits the load request among the active load generators
func (c Cache) Distribute(n, con, urlToHit, token string) (ABResponses, error) {
	ab := ABResponses{}

	list, err := c.Generators()

	if err != nil {
		return ab, err
	}

	listlen := len(list)

	if listlen == 0 {
		return ab, fmt.Errorf("there are no load nodes registered")
	}

	out := make(chan ABResponse)
	errs := make(chan error)

	perN, perC, err := c.calcRates(n, con, listlen)
	if err != nil {
		return ab, err
	}

	for _, v := range list {

		go func(ip, n, concur, url, token string) {
			resp, err := c.send(ip, n, concur, url, token)
			if err != nil {
				errs <- err
				return
			}
			out <- resp
		}(v.IP, perN, perC, urlToHit, token)

	}

	for i := 0; i < listlen; i++ {
		select {
		case res := <-out:
			ab = append(ab, res)
		case err := <-errs:
			return ab, err
		}
	}

	return ab, nil
}

func (c Cache) send(ip, discount, concur, url, token string) (ABResponse, error) {
	ab := ABResponse{}
	u := fmt.Sprintf("http://%s?n=%s&c=%s&url=%s&token=%s", ip, discount, concur, url, token)

	response, err := http.Get(u)
	if err != nil {
		return ab, err
	}
	defer response.Body.Close()

	resp := ABResponse{}
	resp.Load(response.Body)

	return resp, nil
}

// Generator represents a load generator
type Generator struct {
	ID     string `json:"id"`
	IP     string `json:"ip"`
	Active bool   `json:"active"`
}

// JSON Returns the given Node slice as a JSON string
func (n Generator) JSON() (string, error) {

	bytes, err := json.Marshal(n)
	if err != nil {
		return "", fmt.Errorf("could not marshal json for response: %s", err)
	}

	return string(bytes), nil
}

// Load populates a structure with data from json.
func (n *Generator) Load(j string) error {

	if err := json.Unmarshal([]byte(j), n); err != nil {
		return err
	}
	return nil
}

// Generators is a slice of strings that are the Instances
type Generators []Generator

// JSON Returns the given NodeList slice as a JSON string
func (i Generators) JSON() (string, error) {

	bytes, err := json.Marshal(i)
	if err != nil {
		return "", fmt.Errorf("could not marshal json for response: %s", err)
	}

	return string(bytes), nil
}

// Instance is a record of one instantiation of a load receiver.
type Instance struct {
	ID    string `json:"id"`
	Env   string `json:"env"`
	Count int    `json:"count"`
}

// Incr adds to the instance counter
func (i *Instance) Incr() {
	i.Count++
}

// JSON Returns the given Instance struct as a JSON string
func (i Instance) JSON() (string, error) {

	bytes, err := json.Marshal(i)
	if err != nil {
		return "", fmt.Errorf("could not marshal json for response: %s", err)
	}

	return string(bytes), nil
}

// InstanceReport refers to a collection of instances in redis
type InstanceReport map[string]Instance

// JSON Returns the given Index struct as a JSON string
func (i InstanceReport) JSON() (string, error) {

	bytes, err := json.Marshal(i)
	if err != nil {
		return "", fmt.Errorf("could not marshal json for response: %s", err)
	}

	return string(bytes), nil
}

// ABResponse is an extreme summary of the response from Apache Bench
type ABResponse struct {
	Token  string
	IP     string
	Status string
}

// JSON Returns the given ABResponse struct as a JSON string
func (a ABResponse) JSON() (string, error) {

	bytes, err := json.Marshal(a)
	if err != nil {
		return "", fmt.Errorf("could not marshal json for response: %s", err)
	}

	return string(bytes), nil
}

// Load takes the content of a http response and creates a struct of it.
func (a *ABResponse) Load(r io.Reader) error {

	bodyBytes, err := ioutil.ReadAll(r)
	if err != nil {
		log.Fatal(err)
	}

	if err := json.Unmarshal(bodyBytes, &a); err != nil {
		return fmt.Errorf("could not marshal json for response: %s", err)
	}

	return nil
}

// ABResponses is a list of ABResponses
type ABResponses []ABResponse

// JSON Returns the given ABResponse struct as a JSON string
func (a ABResponses) JSON() (string, error) {

	bytes, err := json.Marshal(a)
	if err != nil {
		return "", fmt.Errorf("could not marshal json for response: %s", err)
	}

	return string(bytes), nil
}

// Receiver is a record of the various endpoints that receive load.
type Receiver struct {
	Env      string `json:"env"`
	Endpoint string `json:"endpoint"`
}

// JSON Returns the given Recevier struct as a JSON string
func (r Receiver) JSON() (string, error) {

	bytes, err := json.Marshal(r)
	if err != nil {
		return "", fmt.Errorf("could not marshal json for response: %s", err)
	}

	return string(bytes), nil
}

// Load populates a structure with data from json.
func (r *Receiver) Load(j string) error {

	if err := json.Unmarshal([]byte(j), r); err != nil {
		return err
	}
	return nil
}

// Receivers is a slice of strings that are the Instances
type Receivers []Receiver

// JSON Returns the given ReceiverList slice as a JSON string
func (r Receivers) JSON() (string, error) {

	bytes, err := json.Marshal(r)
	if err != nil {
		return "", fmt.Errorf("could not marshal json for response: %s", err)
	}

	return string(bytes), nil
}

// URLList turns the list of endpoints as URLS
func (r Receivers) URLList() ([]*url.URL, error) {
	result := []*url.URL{}

	for _, v := range r {
		u, err := url.Parse(v.Endpoint)
		if err != nil {
			return result, err
		}
		result = append(result, u)

	}
	return result, nil
}
