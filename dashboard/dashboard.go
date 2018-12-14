//
//
//
//
//
//
//
//
//
//
//
//
//
//
//

package dashboard

//go:generate go-bindata -nometadata -o assets.go -prefix assets -pkg dashboard assets/public/...

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/5sWind/bgmchain/log"
	"github.com/5sWind/bgmchain/p2p"
	"github.com/5sWind/bgmchain/rpc"
	"github.com/rcrowley/go-metrics"
	"golang.org/x/net/websocket"
)

const (
	memorySampleLimit  = 200 //
	trafficSampleLimit = 200 //
)

var nextId uint32 //

//
type Dashboard struct {
	config *Config

	listener net.Listener
	conns    map[uint32]*client //
	charts   charts             //
	lock     sync.RWMutex       //

	quit chan chan error //
	wg   sync.WaitGroup
}

//
type message struct {
	History *charts     `json:"history,omitempty"` //
	Memory  *chartEntry `json:"memory,omitempty"`  //
	Traffic *chartEntry `json:"traffic,omitempty"` //
	Log     string      `json:"log,omitempty"`     //
}

//
type client struct {
	conn   *websocket.Conn //
	msg    chan message    //
	logger log.Logger      //
}

//
type charts struct {
	Memory  []*chartEntry `json:"memorySamples,omitempty"`
	Traffic []*chartEntry `json:"trafficSamples,omitempty"`
}

//
type chartEntry struct {
	Time  time.Time `json:"time,omitempty"`
	Value float64   `json:"value,omitempty"`
}

//
func New(config *Config) (*Dashboard, error) {
	return &Dashboard{
		conns:  make(map[uint32]*client),
		config: config,
		quit:   make(chan chan error),
	}, nil
}

//
func (db *Dashboard) Protocols() []p2p.Protocol { return nil }

//
func (db *Dashboard) APIs() []rpc.API { return nil }

//
func (db *Dashboard) Start(server *p2p.Server) error {
	db.wg.Add(2)
	go db.collectData()
	go db.collectLogs() //

	http.HandleFunc("/", db.webHandler)
	http.Handle("/api", websocket.Handler(db.apiHandler))

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", db.config.Host, db.config.Port))
	if err != nil {
		return err
	}
	db.listener = listener

	go http.Serve(listener, nil)

	return nil
}

//
func (db *Dashboard) Stop() error {
//
	var errs []error
	if err := db.listener.Close(); err != nil {
		errs = append(errs, err)
	}
//
	errc := make(chan error, 1)
	for i := 0; i < 2; i++ {
		db.quit <- errc
		if err := <-errc; err != nil {
			errs = append(errs, err)
		}
	}
//
	db.lock.Lock()
	for _, c := range db.conns {
		if err := c.conn.Close(); err != nil {
			c.logger.Warn("Failed to close connection", "err", err)
		}
	}
	db.lock.Unlock()

//
	db.wg.Wait()
	log.Info("Dashboard stopped")

	var err error
	if len(errs) > 0 {
		err = fmt.Errorf("%v", errs)
	}

	return err
}

//
func (db *Dashboard) webHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("Request", "URL", r.URL)

	path := r.URL.String()
	if path == "/" {
		path = "/dashboard.html"
	}
//
	if db.config.Assets != "" {
		blob, err := ioutil.ReadFile(filepath.Join(db.config.Assets, path))
		if err != nil {
			log.Warn("Failed to read file", "path", path, "err", err)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Write(blob)
		return
	}
	blob, err := Asset(filepath.Join("public", path))
	if err != nil {
		log.Warn("Failed to load the asset", "path", path, "err", err)
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Write(blob)
}

//
func (db *Dashboard) apiHandler(conn *websocket.Conn) {
	id := atomic.AddUint32(&nextId, 1)
	client := &client{
		conn:   conn,
		msg:    make(chan message, 128),
		logger: log.New("id", id),
	}
	done := make(chan struct{}) //

//
	db.wg.Add(1)
	go func() {
		defer db.wg.Done()

		for {
			select {
			case <-done:
				return
			case msg := <-client.msg:
				if err := websocket.JSON.Send(client.conn, msg); err != nil {
					client.logger.Warn("Failed to send the message", "msg", msg, "err", err)
					client.conn.Close()
					return
				}
			}
		}
	}()
//
	client.msg <- message{
		History: &db.charts,
	}
//
	db.lock.Lock()
	db.conns[id] = client
	db.lock.Unlock()
	defer func() {
		db.lock.Lock()
		delete(db.conns, id)
		db.lock.Unlock()
	}()
	for {
		fail := []byte{}
		if _, err := conn.Read(fail); err != nil {
			close(done)
			return
		}
//
	}
}

//
func (db *Dashboard) collectData() {
	defer db.wg.Done()

	for {
		select {
		case errc := <-db.quit:
			errc <- nil
			return
		case <-time.After(db.config.Refresh):
			inboundTraffic := metrics.DefaultRegistry.Get("p2p/InboundTraffic").(metrics.Meter).Rate1()
			memoryInUse := metrics.DefaultRegistry.Get("system/memory/inuse").(metrics.Meter).Rate1()
			now := time.Now()
			memory := &chartEntry{
				Time:  now,
				Value: memoryInUse,
			}
			traffic := &chartEntry{
				Time:  now,
				Value: inboundTraffic,
			}
//
			first := 0
			if len(db.charts.Memory) == memorySampleLimit {
				first = 1
			}
			db.charts.Memory = append(db.charts.Memory[first:], memory)
			first = 0
			if len(db.charts.Traffic) == trafficSampleLimit {
				first = 1
			}
			db.charts.Traffic = append(db.charts.Traffic[first:], traffic)

			db.sendToAll(&message{
				Memory:  memory,
				Traffic: traffic,
			})
		}
	}
}

//
func (db *Dashboard) collectLogs() {
	defer db.wg.Done()

//
	for {
		select {
		case errc := <-db.quit:
			errc <- nil
			return
		case <-time.After(db.config.Refresh / 2):
			db.sendToAll(&message{
				Log: "This is a fake log.",
			})
		}
	}
}

//
func (db *Dashboard) sendToAll(msg *message) {
	db.lock.Lock()
	for _, c := range db.conns {
		select {
		case c.msg <- *msg:
		default:
			c.conn.Close()
		}
	}
	db.lock.Unlock()
}
