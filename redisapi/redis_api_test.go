package redisapi

import (
	"fmt"
	"github.com/absolute8511/ZanRedisDB/node"
	"github.com/absolute8511/ZanRedisDB/store"
	"github.com/coreos/etcd/raft/raftpb"
	"github.com/siddontang/goredis"
	"io/ioutil"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"
)

var testOnce sync.Once
var kvs *node.KVNode
var redisport int
var OK = "OK"

func startTestServer(t *testing.T) (*node.KVNode, int, string) {
	confChangeC := make(chan raftpb.ConfChange)
	tmpDir, err := ioutil.TempDir("", fmt.Sprintf("rocksdb-test-%d", time.Now().UnixNano()))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("dir:%v\n", tmpDir)
	raftAddr := "127.0.0.1:12345"
	redisport := 22345
	clusterNodes := make(map[int]string)
	clusterNodes[1] = raftAddr
	kvOpts := &store.KVOptions{
		DataDir: tmpDir,
		EngType: "rocksdb",
	}
	kvs, nodeStopC := node.NewKVNode(kvOpts, 1000, 1, raftAddr,
		clusterNodes, false, confChangeC)
	go ServeRedisAPI(redisport, nodeStopC)
	time.Sleep(time.Millisecond * 10)
	return kvs, redisport, tmpDir
}

func getTestConn(t *testing.T) *goredis.PoolConn {
	testOnce.Do(func() {
		kvs, redisport, _ = startTestServer(t)
	},
	)
	c := goredis.NewClient("127.0.0.1:"+strconv.Itoa(redisport), "")
	c.SetMaxIdleConns(4)
	conn, err := c.Get()
	if err != nil {
		t.Fatal(err)
	}
	return conn
}

func TestKV(t *testing.T) {
	c := getTestConn(t)
	defer c.Close()

	key1 := "test:a"
	key2 := "test:b"
	if ok, err := goredis.String(c.Do("set", key1, "1234")); err != nil {
		t.Fatal(err)
	} else if ok != OK {
		t.Fatal(ok)
	}

	if n, err := goredis.Int(c.Do("setnx", key1, "123")); err != nil {
		t.Fatal(err)
	} else if n != 0 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("setnx", key2, "123")); err != nil {
		t.Fatal(err)
	} else if n != 1 {
		t.Fatal(n)
	}

	if v, err := goredis.String(c.Do("get", key1)); err != nil {
		t.Fatal(err)
	} else if v != "1234" {
		t.Fatal(v)
	}

	//if v, err := goredis.String(c.Do("getset", "a", "123")); err != nil {
	//	t.Fatal(err)
	//} else if v != "1234" {
	//	t.Fatal(v)
	//}

	//if v, err := goredis.String(c.Do("get", "a")); err != nil {
	//	t.Fatal(err)
	//} else if v != "123" {
	//	t.Fatal(v)
	//}

	if n, err := goredis.Int(c.Do("exists", key1)); err != nil {
		t.Fatal(err)
	} else if n != 1 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("exists", "test:empty_key_test")); err != nil {
		t.Fatal(err)
	} else if n != 0 {
		t.Fatal(n)
	}

	if _, err := goredis.Int(c.Do("del", key1, key2)); err != nil {
		t.Fatal(err)
	}

	if n, err := goredis.Int(c.Do("exists", key1)); err != nil {
		t.Fatal(err)
	} else if n != 0 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("exists", key2)); err != nil {
		t.Fatal(err)
	} else if n != 0 {
		t.Fatal(n)
	}
}

func TestKVM(t *testing.T) {
	c := getTestConn(t)
	defer c.Close()

	key1 := "test:a"
	key2 := "test:b"
	key3 := "test:c"
	if ok, err := goredis.String(c.Do("mset", key1, "1", key2, "2")); err != nil {
		t.Fatal(err)
	} else if ok != OK {
		t.Fatal(ok)
	}

	if v, err := goredis.MultiBulk(c.Do("mget", key1, key2, key3)); err != nil {
		t.Fatal(err)
	} else if len(v) != 3 {
		t.Fatal(len(v))
	} else {
		if vv, ok := v[0].([]byte); !ok || string(vv) != "1" {
			t.Fatal("not 1")
		}

		if vv, ok := v[1].([]byte); !ok || string(vv) != "2" {
			t.Fatal("not 2")
		}

		if v[2] != nil {
			t.Errorf("must nil: %v", v[2])
		}
	}
}

func TestKVIncrDecr(t *testing.T) {
	c := getTestConn(t)
	defer c.Close()

	key := "test:kv_n"
	if n, err := goredis.Int64(c.Do("incr", key)); err != nil {
		t.Fatal(err)
	} else if n != 1 {
		t.Fatal(n)
	}

	if n, err := goredis.Int64(c.Do("incr", key)); err != nil {
		t.Fatal(err)
	} else if n != 2 {
		t.Fatal(n)
	}

	//if n, err := goredis.Int64(c.Do("incrby", "n", 10)); err != nil {
	//	t.Fatal(err)
	//} else if n != 11 {
	//	t.Fatal(n)
	//}

	//if n, err := goredis.Int64(c.Do("decrby", "n", 10)); err != nil {
	//	t.Fatal(err)
	//} else if n != 1 {
	//	t.Fatal(n)
	//}
}

func TestKVErrorParams(t *testing.T) {
	c := getTestConn(t)
	defer c.Close()

	key1 := "test:a"
	key2 := "test:b"
	key3 := "test:c"
	if _, err := c.Do("get", key1, key2, key3); err == nil {
		t.Fatalf("invalid err %v", err)
	}

	if _, err := c.Do("set", key1, key2, key3); err == nil {
		t.Fatalf("invalid err %v", err)
	}

	if _, err := c.Do("getset", key1, key2, key3); err == nil {
		t.Fatalf("invalid err %v", err)
	}

	if _, err := c.Do("setnx", key1, key2, key3); err == nil {
		t.Fatalf("invalid err %v", err)
	}

	if _, err := c.Do("exists", key1, key2); err == nil {
		t.Fatalf("invalid err %v", err)
	}

	if _, err := c.Do("incr", key1, key2); err == nil {
		t.Fatalf("invalid err %v", err)
	}

	if _, err := c.Do("incrby", key1); err == nil {
		t.Fatalf("invalid err %v", err)
	}

	if _, err := c.Do("decrby", key1); err == nil {
		t.Fatalf("invalid err %v", err)
	}

	if _, err := c.Do("del"); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("mset"); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("mset", key1, key2, key3); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("mget"); err == nil {
		t.Fatal("invalid err of %v", err)
	}

}

func TestHash(t *testing.T) {
	c := getTestConn(t)
	defer c.Close()

	key := []byte("test:a")
	//if n, err := goredis.Int(c.Do("hkeyexists", key)); err != nil {
	//	t.Fatal(err)
	//} else if n != 0 {
	//	t.Fatal(n)
	//}

	if n, err := goredis.Int(c.Do("hset", key, 1, 0)); err != nil {
		t.Fatal(err)
	} else if n != 1 {
		t.Fatal(n)
	}
	//if n, err := goredis.Int(c.Do("hkeyexists", key)); err != nil {
	//	t.Fatal(err)
	//} else if n != 1 {
	//	t.Fatal(n)
	//}

	if n, err := goredis.Int(c.Do("hexists", key, 1)); err != nil {
		t.Fatal(err)
	} else if n != 1 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("hexists", key, -1)); err != nil {
		t.Fatal(err)
	} else if n != 0 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("hget", key, 1)); err != nil {
		t.Fatal(err)
	} else if n != 0 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("hset", key, 1, 1)); err != nil {
		t.Fatal(err)
	} else if n != 0 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("hget", key, 1)); err != nil {
		t.Fatal(err)
	} else if n != 1 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("hlen", key)); err != nil {
		t.Fatal(err)
	} else if n != 1 {
		t.Fatal(n)
	}
}

func testHashArray(ay []interface{}, checkValues ...int) error {
	if len(ay) != len(checkValues) {
		return fmt.Errorf("invalid return number %d != %d", len(ay), len(checkValues))
	}

	for i := 0; i < len(ay); i++ {
		if ay[i] == nil && checkValues[i] != 0 {
			return fmt.Errorf("must nil")
		} else if ay[i] != nil {
			v, ok := ay[i].([]byte)
			if !ok {
				return fmt.Errorf("invalid return data %d %v :%T", i, ay[i], ay[i])
			}

			d, _ := strconv.Atoi(string(v))

			if d != checkValues[i] {
				return fmt.Errorf("invalid data %d %s != %d", i, v, checkValues[i])
			}
		}
	}
	return nil
}

func TestHashM(t *testing.T) {
	c := getTestConn(t)
	defer c.Close()

	key := []byte("test:msetb")
	if ok, err := goredis.String(c.Do("hmset", key, 1, 1, 2, 2, 3, 3)); err != nil {
		t.Fatal(err)
	} else if ok != OK {
		t.Fatal(ok)
	}

	if n, err := goredis.Int(c.Do("hlen", key)); err != nil {
		t.Fatal(err)
	} else if n != 3 {
		t.Fatal(n)
	}

	if v, err := goredis.MultiBulk(c.Do("hmget", key, 1, 2, 3, 4)); err != nil {
		t.Fatal(err)
	} else {
		if err := testHashArray(v, 1, 2, 3, 0); err != nil {
			t.Fatal(err)
		}
	}

	if n, err := goredis.Int(c.Do("hdel", key, 1, 2, 3, 4)); err != nil {
		t.Fatal(err)
	} else if n != 3 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("hlen", key)); err != nil {
		t.Fatal(err)
	} else if n != 0 {
		t.Fatal(n)
	}

	if v, err := goredis.MultiBulk(c.Do("hmget", key, 1, 2, 3, 4)); err != nil {
		t.Fatal(err)
	} else {
		if err := testHashArray(v, 0, 0, 0, 0); err != nil {
			t.Fatal(err)
		}
	}

	if n, err := goredis.Int(c.Do("hlen", key)); err != nil {
		t.Fatal(err)
	} else if n != 0 {
		t.Fatal(n)
	}
}

func TestHashIncr(t *testing.T) {
	c := getTestConn(t)
	defer c.Close()

	key := []byte("test:c")
	if n, err := goredis.Int(c.Do("hincrby", key, 1, 1)); err != nil {
		t.Fatal(err)
	} else if n != 1 {
		t.Fatal(err)
	}

	if n, err := goredis.Int(c.Do("hlen", key)); err != nil {
		t.Fatal(err)
	} else if n != 1 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("hincrby", key, 1, 10)); err != nil {
		t.Fatal(err)
	} else if n != 11 {
		t.Fatal(err)
	}

	if n, err := goredis.Int(c.Do("hlen", key)); err != nil {
		t.Fatal(err)
	} else if n != 1 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("hincrby", key, 1, -11)); err != nil {
		t.Fatal(err)
	} else if n != 0 {
		t.Fatal(err)
	}
}

func TestHashGetAll(t *testing.T) {
	c := getTestConn(t)
	defer c.Close()

	key := []byte("test:hgetalld")

	if ok, err := goredis.String(c.Do("hmset", key, 1, 1, 2, 2, 3, 3)); err != nil {
		t.Fatal(err)
	} else if ok != OK {
		t.Fatal(ok)
	}

	if v, err := goredis.MultiBulk(c.Do("hgetall", key)); err != nil {
		t.Fatal(err)
	} else {
		if err := testHashArray(v, 1, 1, 2, 2, 3, 3); err != nil {
			t.Fatal(err)
		}
	}

	if v, err := goredis.MultiBulk(c.Do("hkeys", key)); err != nil {
		t.Fatal(err)
	} else {
		if err := testHashArray(v, 1, 2, 3); err != nil {
			t.Fatal(err)
		}
	}

	//if v, err := goredis.MultiBulk(c.Do("hvals", key)); err != nil {
	//	t.Fatal(err)
	//} else {
	//	if err := testHashArray(v, 1, 2, 3); err != nil {
	//		t.Fatal(err)
	//	}
	//}

	if n, err := goredis.Int(c.Do("hclear", key)); err != nil {
		t.Fatal(err)
	} else if n != 3 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("hlen", key)); err != nil {
		t.Fatal(err)
	} else if n != 0 {
		t.Fatal(n)
	}
}

func TestHashErrorParams(t *testing.T) {
	c := getTestConn(t)
	defer c.Close()

	key := "test:hash_err_param"
	if _, err := c.Do("hset", key); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("hget", key); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("hexists", key); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("hdel", key); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("hlen", key, "a"); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("hincrby", key); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("hmset", key); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("hmset", key, "f1", "v1", "f2"); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("hmget", key); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("hgetall"); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("hkeys"); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("hvals"); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("hclear"); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("hclear", key, "a"); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("hmclear"); err == nil {
		t.Fatal("invalid err of %v", err)
	}
}

func testListIndex(t *testing.T, key []byte, index int64, v int) error {
	c := getTestConn(t)
	defer c.Close()

	n, err := goredis.Int(c.Do("lindex", key, index))
	if err == goredis.ErrNil && v != 0 {
		return fmt.Errorf("must nil")
	} else if err != nil && err != goredis.ErrNil {
		return err
	} else if n != v {
		return fmt.Errorf("index err number %d != %d", n, v)
	}

	return nil
}

func testListRange(t *testing.T, key []byte, start int64, stop int64, checkValues ...int) error {
	c := getTestConn(t)
	defer c.Close()

	vs, err := goredis.MultiBulk(c.Do("lrange", key, start, stop))
	if err != nil {
		return err
	}

	if len(vs) != len(checkValues) {
		return fmt.Errorf("invalid return number %d != %d", len(vs), len(checkValues))
	}

	var n int
	for i, v := range vs {
		if d, ok := v.([]byte); ok {
			n, err = strconv.Atoi(string(d))
			if err != nil {
				return err
			} else if n != checkValues[i] {
				return fmt.Errorf("invalid data %d: %d != %d", i, n, checkValues[i])
			}
		} else {
			return fmt.Errorf("invalid data %v %T", v, v)
		}
	}

	return nil
}

func TestList(t *testing.T) {
	c := getTestConn(t)
	defer c.Close()

	key := []byte("test:a")
	//if n, err := goredis.Int(c.Do("lkeyexists", key)); err != nil {
	//	t.Fatal(err)
	//} else if n != 0 {
	//	t.Fatal(n)
	//}

	if n, err := goredis.Int(c.Do("lpush", key, 1)); err != nil {
		t.Fatal(err)
	} else if n != 1 {
		t.Fatal(n)
	}

	//if n, err := goredis.Int(c.Do("lkeyexists", key)); err != nil {
	//	t.Fatal(err)
	//} else if n != 1 {
	//	t.Fatal(1)
	//}

	if n, err := goredis.Int(c.Do("rpush", key, 2)); err != nil {
		t.Fatal(err)
	} else if n != 2 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("rpush", key, 3)); err != nil {
		t.Fatal(err)
	} else if n != 3 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("llen", key)); err != nil {
		t.Fatal(err)
	} else if n != 3 {
		t.Fatal(n)
	}

	//for ledis-cli a 1 2 3
	// 127.0.0.1:6379> lrange a 0 0
	// 1) "1"
	if err := testListRange(t, key, 0, 0, 1); err != nil {
		t.Fatal(err)
	}

	// 127.0.0.1:6379> lrange a 0 1
	// 1) "1"
	// 2) "2"

	if err := testListRange(t, key, 0, 1, 1, 2); err != nil {
		t.Fatal(err)
	}

	// 127.0.0.1:6379> lrange a 0 5
	// 1) "1"
	// 2) "2"
	// 3) "3"
	if err := testListRange(t, key, 0, 5, 1, 2, 3); err != nil {
		t.Fatal(err)
	}

	// 127.0.0.1:6379> lrange a -1 5
	// 1) "3"
	if err := testListRange(t, key, -1, 5, 3); err != nil {
		t.Fatal(err)
	}

	// 127.0.0.1:6379> lrange a -5 -1
	// 1) "1"
	// 2) "2"
	// 3) "3"
	if err := testListRange(t, key, -5, -1, 1, 2, 3); err != nil {
		t.Fatal(err)
	}

	// 127.0.0.1:6379> lrange a -2 -1
	// 1) "2"
	// 2) "3"
	if err := testListRange(t, key, -2, -1, 2, 3); err != nil {
		t.Fatal(err)
	}

	// 127.0.0.1:6379> lrange a -1 -2
	// (empty list or set)
	if err := testListRange(t, key, -1, -2); err != nil {
		t.Fatal(err)
	}

	// 127.0.0.1:6379> lrange a -1 2
	// 1) "3"
	if err := testListRange(t, key, -1, 2, 3); err != nil {
		t.Fatal(err)
	}

	// 127.0.0.1:6379> lrange a -5 5
	// 1) "1"
	// 2) "2"
	// 3) "3"
	if err := testListRange(t, key, -5, 5, 1, 2, 3); err != nil {
		t.Fatal(err)
	}

	// 127.0.0.1:6379> lrange a -1 0
	// (empty list or set)
	if err := testListRange(t, key, -1, 0); err != nil {
		t.Fatal(err)
	}

	if err := testListRange(t, []byte("empty list"), 0, 100); err != nil {
		t.Fatal(err)
	}

	// 127.0.0.1:6379> lrange a -1 -1
	// 1) "3"
	if err := testListRange(t, key, -1, -1, 3); err != nil {
		t.Fatal(err)
	}

	if err := testListIndex(t, key, -1, 3); err != nil {
		t.Fatal(err)
	}

	if err := testListIndex(t, key, 0, 1); err != nil {
		t.Fatal(err)
	}

	if err := testListIndex(t, key, 1, 2); err != nil {
		t.Fatal(err)
	}

	if err := testListIndex(t, key, 2, 3); err != nil {
		t.Fatal(err)
	}

	if err := testListIndex(t, key, 5, 0); err != nil {
		t.Fatal(err)
	}

	if err := testListIndex(t, key, -1, 3); err != nil {
		t.Fatal(err)
	}

	if err := testListIndex(t, key, -2, 2); err != nil {
		t.Fatal(err)
	}

	if err := testListIndex(t, key, -3, 1); err != nil {
		t.Fatal(err)
	}
}

func TestListMPush(t *testing.T) {
	c := getTestConn(t)
	defer c.Close()

	key := []byte("test:b")
	if n, err := goredis.Int(c.Do("rpush", key, 1, 2, 3)); err != nil {
		t.Fatal(err)
	} else if n != 3 {
		t.Fatal(n)
	}

	if err := testListRange(t, key, 0, 3, 1, 2, 3); err != nil {
		t.Fatal(err)
	}

	if n, err := goredis.Int(c.Do("lpush", key, 1, 2, 3)); err != nil {
		t.Fatal(err)
	} else if n != 6 {
		t.Fatal(n)
	}

	if err := testListRange(t, key, 0, 6, 3, 2, 1, 1, 2, 3); err != nil {
		t.Fatal(err)
	}
}

func TestPop(t *testing.T) {
	c := getTestConn(t)
	defer c.Close()

	key := []byte("test:c")
	if n, err := goredis.Int(c.Do("rpush", key, 1, 2, 3, 4, 5, 6)); err != nil {
		t.Fatal(err)
	} else if n != 6 {
		t.Fatal(n)
	}

	if v, err := goredis.Int(c.Do("lpop", key)); err != nil {
		t.Fatal(err)
	} else if v != 1 {
		t.Fatal(v)
	}

	if v, err := goredis.Int(c.Do("rpop", key)); err != nil {
		t.Fatal(err)
	} else if v != 6 {
		t.Fatal(v)
	}

	if n, err := goredis.Int(c.Do("lpush", key, 1)); err != nil {
		t.Fatal(err)
	} else if n != 5 {
		t.Fatal(n)
	}

	if err := testListRange(t, key, 0, 5, 1, 2, 3, 4, 5); err != nil {
		t.Fatal(err)
	}

	for i := 1; i <= 5; i++ {
		if v, err := goredis.Int(c.Do("lpop", key)); err != nil {
			t.Fatal(err)
		} else if v != i {
			t.Fatal(v)
		}
	}

	if n, err := goredis.Int(c.Do("llen", key)); err != nil {
		t.Fatal(err)
	} else if n != 0 {
		t.Fatal(n)
	}

	c.Do("rpush", key, 1, 2, 3, 4, 5)

	if n, err := goredis.Int(c.Do("lclear", key)); err != nil {
		t.Fatal(err)
	} else if n != 5 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("llen", key)); err != nil {
		t.Fatal(err)
	} else if n != 0 {
		t.Fatal(n)
	}

}

func disableTestTrim(t *testing.T) {
	c := getTestConn(t)
	defer c.Close()

	key := []byte("test:d")
	if n, err := goredis.Int(c.Do("rpush", key, 1, 2, 3, 4, 5, 6)); err != nil {
		t.Fatal(err)
	} else if n != 6 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("ltrim_front", key, 2)); err != nil {
		t.Fatal(err)
	} else if n != 2 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("llen", key)); err != nil {
		t.Fatal(err)
	} else if n != 4 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("ltrim_back", key, 2)); err != nil {
		t.Fatal(err)
	} else if n != 2 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("llen", key)); err != nil {
		t.Fatal(err)
	} else if n != 2 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("ltrim_front", key, 5)); err != nil {
		t.Fatal(err)
	} else if n != 2 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("llen", key)); err != nil {
		t.Fatal(err)
	} else if n != 0 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("rpush", key, 1, 2)); err != nil {
		t.Fatal(err)
	} else if n != 2 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("ltrim_front", key, 2)); err != nil {
		t.Fatal(err)
	} else if n != 2 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("llen", key)); err != nil {
		t.Fatal(err)
	} else if n != 0 {
		t.Fatal(n)
	}
}

func TestListErrorParams(t *testing.T) {
	c := getTestConn(t)
	defer c.Close()

	key := "test:list_err_param"
	if _, err := c.Do("lpush", key); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("rpush", key); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("lpop", key, "a"); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("rpop", key, "a"); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("llen", key, "a"); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("lindex", key); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("lrange", key); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("lclear"); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("lmclear"); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("ltrim_front", key, "-1"); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("ltrim_back", key, "a"); err == nil {
		t.Fatal("invalid err of %v", err)
	}
}

func disableTestSet(t *testing.T) {
	c := getTestConn(t)
	defer c.Close()

	key1 := "test:testdb_cmd_set_1"
	key2 := "test:testdb_cmd_set_2"

	//if n, err := goredis.Int(c.Do("skeyexists", key1)); err != nil {
	//	t.Fatal(err)
	//} else if n != 0 {
	//	t.Fatal(n)
	//}

	if n, err := goredis.Int(c.Do("sadd", key1, 0, 1)); err != nil {
		t.Fatal(err)
	} else if n != 2 {
		t.Fatal(n)
	}

	//if n, err := goredis.Int(c.Do("skeyexists", key1)); err != nil {
	//	t.Fatal(err)
	//} else if n != 1 {
	//	t.Fatal(n)
	//}

	if n, err := goredis.Int(c.Do("sadd", key2, 0, 1, 2, 3)); err != nil {
		t.Fatal(err)
	} else if n != 4 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("srem", key1, 0, 1)); err != nil {
		t.Fatal(err)
	} else if n != 2 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("sismember", key2, 0)); err != nil {
		t.Fatal(err)
	} else if n != 1 {
		t.Fatal(n)
	}

	if n, err := goredis.MultiBulk(c.Do("smembers", key2)); err != nil {
		t.Fatal(err)
	} else if len(n) != 4 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("sclear", key2)); err != nil {
		t.Fatal(err)
	} else if n != 4 {
		t.Fatal(n)
	}

	c.Do("sadd", key1, 0)
	c.Do("sadd", key2, 1)
	if n, err := goredis.Int(c.Do("smclear", key1, key2)); err != nil {
		t.Fatal(err)
	} else if n != 2 {
		t.Fatal(n)
	}
}

func TestSetErrorParams(t *testing.T) {
	c := getTestConn(t)
	defer c.Close()

	key := "test:set_error_param"
	if _, err := c.Do("sadd", key); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("scard"); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("scard", key, key); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("sismember", key); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("sismember", key, "m1", "m2"); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("smembers"); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("smembers", key, key); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("srem"); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("srem", key); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("sclear"); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("sclear", key, key); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("smclear"); err == nil {
		t.Fatal("invalid err of %v", err)
	}

}

func TestZSet(t *testing.T) {
	c := getTestConn(t)
	defer c.Close()

	key := []byte("test:myzset")

	//if n, err := goredis.Int(c.Do("zkeyexists", key)); err != nil {
	//	t.Fatal(err)
	//} else if n != 0 {
	//	t.Fatal(n)
	//}

	if n, err := goredis.Int(c.Do("zadd", key, 3, "a", 4, "b")); err != nil {
		t.Fatal(err)
	} else if n != 2 {
		t.Fatal(n)
	}

	//if n, err := goredis.Int(c.Do("zkeyexists", key)); err != nil {
	//	t.Fatal(err)
	//} else if n != 1 {
	//	t.Fatal(n)
	//}

	if n, err := goredis.Int(c.Do("zcard", key)); err != nil {
		t.Fatal(n)
	} else if n != 2 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("zadd", key, 1, "a", 2, "b")); err != nil {
		t.Fatal(err)
	} else if n != 0 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("zcard", key)); err != nil {
		t.Fatal(n)
	} else if n != 2 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("zadd", key, 3, "c", 4, "d")); err != nil {
		t.Fatal(err)
	} else if n != 2 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("zcard", key)); err != nil {
		t.Fatal(err)
	} else if n != 4 {
		t.Fatal(n)
	}

	if s, err := goredis.Int(c.Do("zscore", key, "c")); err != nil {
		t.Fatal(err)
	} else if s != 3 {
		t.Fatal(s)
	}

	if n, err := goredis.Int(c.Do("zrem", key, "d", "e")); err != nil {
		t.Fatal(err)
	} else if n != 1 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("zcard", key)); err != nil {
		t.Fatal(err)
	} else if n != 3 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("zincrby", key, 4, "c")); err != nil {
		t.Fatal(err)
	} else if n != 7 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("zincrby", key, -4, "c")); err != nil {
		t.Fatal(err)
	} else if n != 3 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("zincrby", key, 4, "d")); err != nil {
		t.Fatal(err)
	} else if n != 4 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("zcard", key)); err != nil {
		t.Fatal(err)
	} else if n != 4 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("zrem", key, "a", "b", "c", "d")); err != nil {
		t.Fatal(err)
	} else if n != 4 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("zcard", key)); err != nil {
		t.Fatal(err)
	} else if n != 0 {
		t.Fatal(n)
	}

}

func TestZSetCount(t *testing.T) {
	c := getTestConn(t)
	defer c.Close()

	key := []byte("test:myzset")
	if _, err := goredis.Int(c.Do("zadd", key, 1, "a", 2, "b", 3, "c", 4, "d")); err != nil {
		t.Fatal(err)
	}

	if n, err := goredis.Int(c.Do("zcount", key, 2, 4)); err != nil {
		t.Fatal(err)
	} else if n != 3 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("zcount", key, 4, 4)); err != nil {
		t.Fatal(err)
	} else if n != 1 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("zcount", key, 4, 3)); err != nil {
		t.Fatal(err)
	} else if n != 0 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("zcount", key, "(2", 4)); err != nil {
		t.Fatal(err)
	} else if n != 2 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("zcount", key, "2", "(4")); err != nil {
		t.Fatal(err)
	} else if n != 2 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("zcount", key, "(2", "(4")); err != nil {
		t.Fatal(err)
	} else if n != 1 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("zcount", key, "-inf", "+inf")); err != nil {
		t.Fatal(err)
	} else if n != 4 {
		t.Fatal(n)
	}

	c.Do("zadd", key, 3, "e")

	if n, err := goredis.Int(c.Do("zcount", key, "(2", "(4")); err != nil {
		t.Fatal(err)
	} else if n != 2 {
		t.Fatal(n)
	}

	c.Do("zrem", key, "a", "b", "c", "d", "e")
}

func TestZSetRank(t *testing.T) {
	c := getTestConn(t)
	defer c.Close()

	key := []byte("test:myzset")
	if _, err := goredis.Int(c.Do("zadd", key, 1, "a", 2, "b", 3, "c", 4, "d")); err != nil {
		t.Fatal(err)
	}

	if n, err := goredis.Int(c.Do("zrank", key, "c")); err != nil {
		t.Fatal(err)
	} else if n != 2 {
		t.Fatal(n)
	}

	if _, err := goredis.Int(c.Do("zrank", key, "e")); err != goredis.ErrNil {
		t.Fatal(err)
	}

	if n, err := goredis.Int(c.Do("zrevrank", key, "c")); err != nil {
		t.Fatalf("cmd error: %v", err)
	} else if n != 1 {
		t.Fatal(n)
	}

	if _, err := goredis.Int(c.Do("zrevrank", key, "e")); err != goredis.ErrNil {
		t.Fatal(err)
	}
}

func testZSetRange(ay []interface{}, checkValues ...interface{}) error {
	if len(ay) != len(checkValues) {
		return fmt.Errorf("invalid return number %d != %d", len(ay), len(checkValues))
	}

	for i := 0; i < len(ay); i++ {
		v, ok := ay[i].([]byte)
		if !ok {
			return fmt.Errorf("invalid data %d %v %T", i, ay[i], ay[i])
		}

		switch cv := checkValues[i].(type) {
		case string:
			if string(v) != cv {
				return fmt.Errorf("not equal %s != %s", v, checkValues[i])
			}
		default:
			if s, _ := strconv.Atoi(string(v)); s != checkValues[i] {
				return fmt.Errorf("not equal %s != %v", v, checkValues[i])
			}
		}

	}

	return nil
}

func TestZSetRangeScore(t *testing.T) {
	c := getTestConn(t)
	defer c.Close()

	key := []byte("test:myzset_range")
	if _, err := goredis.Int(c.Do("zadd", key, 1, "a", 2, "b", 3, "c", 4, "d")); err != nil {
		t.Fatal(err)
	}

	if v, err := goredis.MultiBulk(c.Do("zrangebyscore", key, 1, 4, "withscores")); err != nil {
		t.Fatal(err)
	} else {
		if err := testZSetRange(v, "a", 1, "b", 2, "c", 3, "d", 4); err != nil {
			t.Fatal(err)
		}
	}

	if v, err := goredis.MultiBulk(c.Do("zrangebyscore", key, 1, 4, "withscores", "limit", 1, 2)); err != nil {
		t.Fatal(err)
	} else {
		if err := testZSetRange(v, "b", 2, "c", 3); err != nil {
			t.Fatal(err)
		}
	}

	if v, err := goredis.MultiBulk(c.Do("zrangebyscore", key, "-inf", "+inf", "withscores")); err != nil {
		t.Fatal(err)
	} else {
		if err := testZSetRange(v, "a", 1, "b", 2, "c", 3, "d", 4); err != nil {
			t.Fatal(err)
		}
	}

	if v, err := goredis.MultiBulk(c.Do("zrangebyscore", key, "(1", "(4")); err != nil {
		t.Fatal(err)
	} else {
		if err := testZSetRange(v, "b", "c"); err != nil {
			t.Fatal(err)
		}
	}

	if v, err := goredis.MultiBulk(c.Do("zrevrangebyscore", key, 4, 1, "withscores")); err != nil {
		t.Fatal(err)
	} else {
		if err := testZSetRange(v, "d", 4, "c", 3, "b", 2, "a", 1); err != nil {
			t.Fatalf("%v, %v", err, v)
		}
	}

	if v, err := goredis.MultiBulk(c.Do("zrevrangebyscore", key, 4, 1, "withscores", "limit", 1, 2)); err != nil {
		t.Fatal(err)
	} else {
		if err := testZSetRange(v, "c", 3, "b", 2); err != nil {
			t.Fatal(err)
		}
	}

	if v, err := goredis.MultiBulk(c.Do("zrevrangebyscore", key, "+inf", "-inf", "withscores")); err != nil {
		t.Fatal(err)
	} else {
		if err := testZSetRange(v, "d", 4, "c", 3, "b", 2, "a", 1); err != nil {
			t.Fatal(err)
		}
	}

	if v, err := goredis.MultiBulk(c.Do("zrevrangebyscore", key, "(4", "(1")); err != nil {
		t.Fatal(err)
	} else {
		if err := testZSetRange(v, "c", "b"); err != nil {
			t.Fatal(err)
		}
	}

	if n, err := goredis.Int(c.Do("zremrangebyscore", key, 2, 3)); err != nil {
		t.Fatal(err)
	} else if n != 2 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("zcard", key)); err != nil {
		t.Fatal(err)
	} else if n != 2 {
		t.Fatal(n)
	}

	if v, err := goredis.MultiBulk(c.Do("zrangebyscore", key, 1, 4)); err != nil {
		t.Fatal(err)
	} else {
		if err := testZSetRange(v, "a", "d"); err != nil {
			t.Fatal(err)
		}
	}
}

func TestZSetRange(t *testing.T) {
	c := getTestConn(t)
	defer c.Close()

	key := []byte("test:myzset_range_rank")
	if _, err := goredis.Int(c.Do("zadd", key, 1, "a", 2, "b", 3, "c", 4, "d")); err != nil {
		t.Fatal(err)
	}

	if v, err := goredis.MultiBulk(c.Do("zrange", key, 0, 3, "withscores")); err != nil {
		t.Fatal(err)
	} else {
		if err := testZSetRange(v, "a", 1, "b", 2, "c", 3, "d", 4); err != nil {
			t.Fatal(err)
		}
	}

	if v, err := goredis.MultiBulk(c.Do("zrange", key, 1, 4, "withscores")); err != nil {
		t.Fatal(err)
	} else {
		if err := testZSetRange(v, "b", 2, "c", 3, "d", 4); err != nil {
			t.Fatal(err)
		}
	}

	if v, err := goredis.MultiBulk(c.Do("zrange", key, -2, -1, "withscores")); err != nil {
		t.Fatal(err)
	} else {
		if err := testZSetRange(v, "c", 3, "d", 4); err != nil {
			t.Fatal(err)
		}
	}

	if v, err := goredis.MultiBulk(c.Do("zrange", key, 0, -1, "withscores")); err != nil {
		t.Fatal(err)
	} else {
		if err := testZSetRange(v, "a", 1, "b", 2, "c", 3, "d", 4); err != nil {
			t.Fatal(err)
		}
	}

	if v, err := goredis.MultiBulk(c.Do("zrange", key, -1, -2, "withscores")); err != nil {
		t.Fatal(err)
	} else if len(v) != 0 {
		t.Fatal(len(v))
	}

	if v, err := goredis.MultiBulk(c.Do("zrevrange", key, 0, 4, "withscores")); err != nil {
		t.Fatal(err)
	} else {
		if err := testZSetRange(v, "d", 4, "c", 3, "b", 2, "a", 1); err != nil {
			t.Fatal(err)
		}
	}

	if v, err := goredis.MultiBulk(c.Do("zrevrange", key, 0, -1, "withscores")); err != nil {
		t.Fatal(err)
	} else {
		if err := testZSetRange(v, "d", 4, "c", 3, "b", 2, "a", 1); err != nil {
			t.Fatal(err)
		}
	}

	if v, err := goredis.MultiBulk(c.Do("zrevrange", key, 2, 3, "withscores")); err != nil {
		t.Fatal(err)
	} else {
		if err := testZSetRange(v, "b", 2, "a", 1); err != nil {
			t.Fatal(err)
		}
	}

	if v, err := goredis.MultiBulk(c.Do("zrevrange", key, -2, -1, "withscores")); err != nil {
		t.Fatal(err)
	} else {
		if err := testZSetRange(v, "b", 2, "a", 1); err != nil {
			t.Fatal(err)
		}
	}

	if n, err := goredis.Int(c.Do("zremrangebyrank", key, 2, 3)); err != nil {
		t.Fatal(err)
	} else if n != 2 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("zcard", key)); err != nil {
		t.Fatal(err)
	} else if n != 2 {
		t.Fatal(n)
	}

	if v, err := goredis.MultiBulk(c.Do("zrange", key, 0, 4)); err != nil {
		t.Fatal(err)
	} else {
		if err := testZSetRange(v, "a", "b"); err != nil {
			t.Fatal(err)
		}
	}

	if n, err := goredis.Int(c.Do("zclear", key)); err != nil {
		t.Fatal(err)
	} else if n != 2 {
		t.Fatal(n)
	}

	if n, err := goredis.Int(c.Do("zcard", key)); err != nil {
		t.Fatal(err)
	} else if n != 0 {
		t.Fatal(n)
	}

}

func TestZsetErrorParams(t *testing.T) {
	c := getTestConn(t)
	defer c.Close()

	key := "test:zset_error_param"
	//zadd
	if _, err := c.Do("zadd", key); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("zadd", key, "a", "b", "c"); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("zadd", key, "-a", "a"); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("zadd", key, "0.1", "a"); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	//zcard
	if _, err := c.Do("zcard"); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	//zscore
	if _, err := c.Do("zscore", key); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	//zrem
	if _, err := c.Do("zrem", key); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	//zincrby
	if _, err := c.Do("zincrby", key); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("zincrby", key, 0.1, "a"); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	//zcount
	if _, err := c.Do("zcount", key); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("zcount", key, "-inf", "=inf"); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("zcount", key, 0.1, 0.1); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	//zrank
	if _, err := c.Do("zrank", key); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	//zrevzrank
	if _, err := c.Do("zrevrank", key); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	//zremrangebyrank
	if _, err := c.Do("zremrangebyrank", key); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("zremrangebyrank", key, 0.1, 0.1); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	//zremrangebyscore
	if _, err := c.Do("zremrangebyscore", key); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("zremrangebyscore", key, "-inf", "a"); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("zremrangebyscore", key, 0, "a"); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	//zrange
	if _, err := c.Do("zrange", key); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("zrange", key, 0, 1, "withscore"); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("zrange", key, 0, 1, "withscores", "a"); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	//zrevrange, almost same as zrange
	if _, err := c.Do("zrevrange", key); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	//zrangebyscore

	if _, err := c.Do("zrangebyscore", key); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("zrangebyscore", key, 0, 1, "withscore"); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("zrangebyscore", key, 0, 1, "withscores", "limit"); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("zrangebyscore", key, 0, 1, "withscores", "limi", 1, 1); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("zrangebyscore", key, 0, 1, "withscores", "limit", "a", 1); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	if _, err := c.Do("zrangebyscore", key, 0, 1, "withscores", "limit", 1, "a"); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	//zrevrangebyscore, almost same as zrangebyscore
	if _, err := c.Do("zrevrangebyscore", key); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	//zclear
	if _, err := c.Do("zclear"); err == nil {
		t.Fatal("invalid err of %v", err)
	}

	//zmclear
	if _, err := c.Do("zmclear"); err == nil {
		t.Fatal("invalid err of %v", err)
	}
}

func TestZSetLex(t *testing.T) {
	c := getTestConn(t)
	defer c.Close()

	key := []byte("test:myzlexset")
	if _, err := c.Do("zadd", key,
		0, "a", 0, "b", 0, "c", 0, "d", 0, "e", 0, "f", 0, "g"); err != nil {
		t.Fatal(err)
	}

	if ay, err := goredis.Strings(c.Do("zrangebylex", key, "-", "[c")); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(ay, []string{"a", "b", "c"}) {
		t.Fatal("must equal")
	}

	if ay, err := goredis.Strings(c.Do("zrangebylex", key, "-", "(c")); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(ay, []string{"a", "b"}) {
		t.Fatal("must equal")
	}

	if ay, err := goredis.Strings(c.Do("zrangebylex", key, "[aaa", "(g")); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(ay, []string{"b", "c", "d", "e", "f"}) {
		t.Fatal("must equal")
	}

	if n, err := goredis.Int64(c.Do("zlexcount", key, "-", "(c")); err != nil {
		t.Fatal(err)
	} else if n != 2 {
		t.Fatal(n)
	}

	if n, err := goredis.Int64(c.Do("zremrangebylex", key, "[aaa", "(g")); err != nil {
		t.Fatal(err)
	} else if n != 5 {
		t.Fatal(n)
	}

	if n, err := goredis.Int64(c.Do("zlexcount", key, "-", "+")); err != nil {
		t.Fatal(err)
	} else if n != 2 {
		t.Fatal(n)
	}
}