package caching

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/gomodule/redigo/redis"
)

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

// RegisterNode registers a load producing node.
func (c Cache) RegisterNode(ip string) error {

	conn := c.redisPool.Get()
	defer conn.Close()

	if _, err := conn.Do("HSET", "loadnodes", ip, ip); err != nil {
		return err
	}

	return nil
}

// Index returns the whole collection of all of the instances
func (c Cache) Index() (Index, error) {
	index := Index{}
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

	c.log("Successfully retrieved index from cache")

	return index, nil
}

// ListNodes returns the whole collection of all of the load nodes
func (c Cache) ListNodes() ([]string, error) {
	keys := []string{}

	conn := c.redisPool.Get()
	defer conn.Close()

	s, err := redis.StringMap(conn.Do("HGETALL", "loadnodes"))
	if err == redis.ErrNil {
		return keys, ErrCacheMiss
	} else if err != nil {
		return keys, err
	}

	for i := range s {
		keys = append(keys, i)
	}

	c.log("Successfully retrieved node list from cache")

	return keys, nil
}

// Instance refers to one record in redis
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

// Index refers to a collection of instances in redis
type Index map[string]Instance

// JSON Returns the given Index struct as a JSON string
func (i Index) JSON() (string, error) {

	bytes, err := json.Marshal(i)
	if err != nil {
		return "", fmt.Errorf("could not marshal json for response: %s", err)
	}

	return string(bytes), nil
}
