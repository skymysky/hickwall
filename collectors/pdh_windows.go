package collectors

import (
	"fmt"
	// "github.com/oliveagle/hickwall/config"
	"github.com/oliveagle/hickwall/lib/pdh"
	"github.com/oliveagle/hickwall/newcore"
	"log"
	"time"
)

var (
	_ = fmt.Sprint("")
)

type config_win_pdh_query struct {
	Query  string         `json:"query"`
	Metric newcore.Metric `json:"metric"`
	Tags   newcore.TagSet `json:"tags"`
}

type config_win_pdh_collector struct {
	Interval newcore.Interval        `json:"interval"`
	Tags     newcore.TagSet          `json:"tags"`
	Queries  []*config_win_pdh_query `json:"queries"`
}

type win_pdh_collector struct {
	name     string // collector name
	interval time.Duration
	enabled  bool

	// win_pdh_collector specific attributes
	config      config_win_pdh_collector
	hPdh        *pdh.PdhCollector
	map_queries map[string]*config_win_pdh_query
}

func NewWinPdhCollector(name string, opts config_win_pdh_collector) newcore.Collector {

	c := &win_pdh_collector{
		name:        name,
		enabled:     true,
		interval:    opts.Interval.MustDuration(time.Second),
		config:      opts,
		hPdh:        pdh.NewPdhCollector(),
		map_queries: make(map[string]*config_win_pdh_query),
	}

	for _, q := range opts.Queries {
		c.hPdh.AddEnglishCounter(q.Query)
		if q.Tags == nil {
			q.Tags = newcore.AddTags.Copy()
		}
		q.Tags["query"] = q.Query
		c.map_queries[q.Query] = q
	}

	return c
}

func (c *win_pdh_collector) Name() string {
	return c.name
}

func (c *win_pdh_collector) Close() error {
	c.hPdh.Close()
	return nil
}

func (c *win_pdh_collector) ClassName() string {
	return "win_pdh_collector"
}

func (c *win_pdh_collector) IsEnabled() bool {
	return c.enabled
}

func (c *win_pdh_collector) Interval() time.Duration {
	return c.interval
}

func (c *win_pdh_collector) CollectOnce() *newcore.CollectResult {
	var items newcore.MultiDataPoint
	// log.Println("win_pdh_collector_CollectOnce Started")

	if c.hPdh != nil {
		for _, pd := range c.hPdh.CollectData() {
			// log.Println("pd : ", pd)
			if pd.Err == nil {
				query, ok := c.map_queries[pd.Query]
				if ok == true && query != nil {
					Add(&items, query.Metric.Clean(), pd.Value, query.Tags, "", "", "")
				}
			} else {
				log.Println("win_pdh_collector ERROR: ", pd.Err)
			}
		}
	} else {
		log.Println("win_pdh_collector ERROR: c.hPdh is nil")
	}

	// log.Println("win_pdh_collector_CollectOnce Finished")
	return &newcore.CollectResult{
		Collected: &items,
		Next:      time.Now().Add(c.interval),
		Err:       nil,
	}
}
