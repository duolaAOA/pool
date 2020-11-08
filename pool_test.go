package pool

import (
	"log"
	"net"
	"sync"
	"testing"
	"time"
)

var (
	InitialCap = 5
	MaximumCap = 30
	network	   = "tcp"
	address	   = "127.0.0.1:7777"
	factory    = func() (net.Conn, error) {return net.Dial(network, address)}
	testPool   = &Pool{}
)

func init() {
	// used for factory function
	go simpleTCPSERVER()
	time.Sleep(time.Millisecond * 300) // wait until tcp server has been settled

	var err error
	testPool, err = newPool()
	if err != nil {
		log.Fatalln(err)
	}
}

func TestNew(t *testing.T) {
	_, err := newPool()
	if err != nil {
		t.Errorf("New error: %s", err)
	}
}

func TestPool_Get(t *testing.T) {
	_, err := testPool.Get()
	if err != nil {
		t.Errorf("Get error: %s", err)
	}

	if testPool.UsedCapacity() != (InitialCap - 1) {
		t.Errorf("Get error. Expecting %d, got %d", InitialCap - 1, testPool.UsedCapacity())
	}

	var wg sync.WaitGroup
	for i := 0; i < (InitialCap - 1); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := testPool.Get()
			if err != nil {
				t.Errorf("Get error: %s", err)
			}
		}()
	}

	wg.Wait()

	if testPool.UsedCapacity() != 0 {
		t.Errorf("Get error. Expecting %d, got %d", InitialCap - 1, testPool.UsedCapacity())
	}

	_, err = testPool.Get()
	if err != nil {
		t.Errorf("Get error: %s", err)
	}
}

func TestPool_Put(t *testing.T) {
	conn, err := testPool.Get()
	if err != nil {
		t.Errorf("Put, get error: %s", err)
	}

	testPool.Put(conn)

	if testPool.UsedCapacity() != 1 {
		t.Errorf("Put error. Expecting %d, got %d",
			1, testPool.UsedCapacity())
	}
}

func TestPool_MaximumCapacity(t *testing.T) {
	// Create new pool to test it
	p, _ := newPool()
	if p.MaximumCapacity() != MaximumCap {
		t.Errorf("MaximumCapacity error. Expecting %d, got %d",
			MaximumCap, testPool.UsedCapacity())
	}
}

func TestPool_UsedCapacity(t *testing.T) {
	// Create new pool to test it
	p, _ := newPool()
	if p.UsedCapacity() != InitialCap {
		t.Errorf("InitialCap error. Expecting %d, got %d",
			InitialCap, p.UsedCapacity())
	}
}

func TestPoolConcurrent(t *testing.T) {
	p, _ := newPool()
	pipe := make(chan net.Conn, 0)

	go func() {
		p.Close()
	}()

	for i:= 0; i < MaximumCap; i++ {
		go func() {
			conn, _ := p.Get()

			pipe <- conn
		}()

		go func() {
			conn := <- pipe
			p.Put(conn)
		}()
	}
}

func newPool() (*Pool, error)  {
	return New(InitialCap, MaximumCap, factory)
}

func simpleTCPSERVER() {
	l, err := net.Listen(network, address)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}

		buffer := make([]byte, 256)
		n, err := conn.Read(buffer)
		log.Println(n, err)
	}
}