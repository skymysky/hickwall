package collectors

import (
	"fmt"
	"github.com/GaryBoone/GoStats/stats"
	"github.com/oliveagle/hickwall/collectors/config"
	"github.com/oliveagle/hickwall/newcore"
	"github.com/tatsushid/go-fastping"
	"log"
	"math"
	"net"
	"time"
)

type ping_collector struct {
	name     string // collector name
	interval time.Duration
	enabled  bool
	prefix   string

	// ping collector specific attributes
	config  config.Config_single_pinger
	timeout time.Duration
	tags    newcore.TagSet
}

// func NewPingCollectorFactory(configs []*Config_ping) []newcore.Collector {
// }

func NewPingCollectors(name, prefix string, conf config.Config_Ping) []newcore.Collector {
	var sconf config.Config_single_pinger
	var pings []newcore.Collector

	sconf.Interval = conf.Interval
	sconf.Metric = conf.Metric
	sconf.Packets = conf.Packets
	sconf.Tags = conf.Tags
	sconf.Timeout = conf.Timeout

	for idx, target := range conf.Targets {
		sconf.Target = target
		ping := NewSinglePingCollector(fmt.Sprintf("%s_%d", name, idx), prefix, sconf)
		pings = append(pings, ping)
	}
	return pings
}

func NewSinglePingCollector(name, prefix string, conf config.Config_single_pinger) newcore.Collector {

	if conf.Target == "" {
		log.Println("CRITICAL: we cannot ping empty target.")
	}

	tags := conf.Tags.Copy()
	tags["target"] = conf.Target

	if conf.Packets <= 0 {
		conf.Packets = 1
	}

	c := &ping_collector{
		name:    name,
		enabled: true,
		config:  conf,
		prefix:  prefix,

		interval: conf.Interval.MustDuration(time.Second),
		timeout:  conf.Timeout.MustDuration(time.Millisecond * 500),
		tags:     tags,
	}
	return c
}

func (c *ping_collector) Name() string {
	return c.name
}

func (c *ping_collector) Close() error {
	return nil
}

func (c *ping_collector) ClassName() string {
	return "ping_collector"
}

func (c *ping_collector) IsEnabled() bool {
	return c.enabled
}

func (c *ping_collector) Interval() time.Duration {
	log.Println("Interval: ", c.interval)
	return c.interval
}

// func (c *ping_collector) CollectOnce_1() *newcore.CollectResult {
// 	var items newcore.MultiDataPoint

// 	for i := 0; i < 10; i++ {
// 		items = append(items, &newcore.DataPoint{
// 			Metric:    newcore.Metric(fmt.Sprintf("metric.%s", c.name)),
// 			Timestamp: time.Now(),
// 			Value:     1,
// 			Tags:      nil,
// 			Meta:      nil,
// 		})
// 	}

// 	return &newcore.CollectResult{
// 		Collected: &items,
// 		Next:      time.Now().Add(c.interval),
// 		Err:       nil,
// 	}
// }

func (c *ping_collector) CollectOnce() *newcore.CollectResult {
	log.Println("ping_collector: CollectOnce Started")
	var (
		md       newcore.MultiDataPoint
		d        stats.Stats
		p        = fastping.NewPinger()
		rtt_chan = make(chan float64)
	)

	ip, err := net.ResolveIPAddr("ip4:icmp", c.config.Target)
	if err != nil {
		log.Printf("ERROR: collector_ping: DNS resolve error: %v", err)
		return &newcore.CollectResult{
			Collected: nil,
			Next:      time.Now().Add(c.interval),
			Err:       fmt.Errorf("collector_ping: DNS resolve error: %v", err),
		}
	}

	p.MaxRTT = c.timeout
	p.AddIPAddr(ip)
	p.OnRecv = func(addr *net.IPAddr, rtt time.Duration) {
		rtt_chan <- float64(rtt.Nanoseconds() / 1000 / 1000)
	}

	go func() {
		for i := 0; i < c.config.Packets; i++ {
			err = p.Run()
			if err != nil {
				fmt.Println("run err", err)
			}
		}
		close(rtt_chan)
	}()

	for rtt := range rtt_chan {
		d.Update(rtt)
	}

	log.Println("ping c.tags: ", c.tags)
	Add(&md, c.prefix, fmt.Sprintf("%s.%s", c.config.Metric, "time_min"), d.Min(), c.tags, "", "", "")
	Add(&md, c.prefix, fmt.Sprintf("%s.%s", c.config.Metric, "time_max"), d.Max(), c.tags, "", "", "")
	Add(&md, c.prefix, fmt.Sprintf("%s.%s", c.config.Metric, "time_avg"), d.Mean(), c.tags, "", "", "")

	std := d.SampleStandardDeviation()
	if math.IsNaN(std) {
		std = 0
	}
	Add(&md, c.prefix, fmt.Sprintf("%s.%s", c.config.Metric, "time_mdev"), std, c.tags, "", "", "")
	Add(&md, c.prefix, fmt.Sprintf("%s.%s", c.config.Metric, "ip"), ip.IP.String(), c.tags, "", "", "")

	lost_pct := float64((c.config.Packets-d.Count())/c.config.Packets) * 100
	Add(&md, c.prefix, fmt.Sprintf("%s.%s", c.config.Metric, "lost_pct"), lost_pct, c.tags, "", "", "")

	// log.Println("ping_collector: CollectOnce Finished")
	return &newcore.CollectResult{
		Collected: &md,
		Next:      time.Now().Add(c.interval),
		Err:       nil,
	}

	// return &newcore.CollectResult{
	// 	Collected: nil,
	// 	Next:      time.Now().Add(c.interval),
	// 	Err:       fmt.Errorf("hahaha error"),
	// }
}
