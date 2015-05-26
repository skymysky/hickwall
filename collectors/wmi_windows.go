package collectors

import (
	"fmt"
	"github.com/mattn/go-ole"
	"github.com/mattn/go-ole/oleutil"
	"github.com/oliveagle/hickwall/collectors/config"
	"github.com/oliveagle/hickwall/newcore"
	"github.com/oliveagle/hickwall/utils"
	"log"
	"regexp"
	"strings"
	"time"
)

var (
	_ = fmt.Sprint("")
)

var (
	win_wmi_pat_format, _ = regexp.Compile("\\/format:\\w+(.xsl)?")
	win_wmi_pat_get, _    = regexp.Compile("\\bget\\b")
	win_wmi_pat_field, _  = regexp.Compile(`\{\{\.\w+((_?)+\w+)+\}\}`)
)

func oleCoUninitialize() {

}

type win_wmi_collector struct {
	name     string // collector name
	interval time.Duration
	enabled  bool

	// win_wmi_collector specific attributes
	service *ole.IDispatch
	config  config.Config_win_wmi
}

func NewWinWmiCollector(name string, opts config.Config_win_wmi) newcore.Collector {

	// // use COINIT_MULTITHREADED model
	// ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED)

	c := &win_wmi_collector{
		name:     name,
		enabled:  true,
		interval: opts.Interval.MustDuration(time.Second * 15),

		config: opts,
	}

	err := c.connect()
	if err != nil {
		log.Println("CRITICAL: win_wmi_collector cannot connect: ", err)
	}

	return c
}

func (c *win_wmi_collector) connect() (err error) {
	// use COINIT_MULTITHREADED model
	ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED)

	unknown, err := oleutil.CreateObject("WbemScripting.SWbemLocator")
	if err != nil {
		log.Println("oleutil.CreateObject Failed: ", err)
		return err
	}
	defer unknown.Release()

	wmi, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		log.Println("QueryInterface Failed: ", err)
		return err
	}
	defer wmi.Release()

	serviceRaw, err := oleutil.CallMethod(wmi, "ConnectServer")
	if err != nil {
		log.Println("Connect to Server Failed", err)
		return err
	}

	c.service = serviceRaw.ToIDispatch()
	return nil
}

func (c *win_wmi_collector) Name() string {
	return c.name
}

func (c *win_wmi_collector) Close() error {
	if c.service != nil {
		c.service.Release()
	}

	//TODO: is there any problem to close multiple times?
	ole.CoUninitialize()
	return nil
}

func (c *win_wmi_collector) ClassName() string {
	return "win_wmi_collector"
}

func (c *win_wmi_collector) IsEnabled() bool {
	return c.enabled
}

func (c *win_wmi_collector) Interval() time.Duration {
	return c.interval
}

func (c *win_wmi_collector) query(query string, fields []string) ([]map[string]string, error) {
	if c.service != nil {
		resultRaw, err := oleutil.CallMethod(c.service, "ExecQuery", query)
		if err != nil {
			log.Println("ExecQuery Failed: ", err)
			return nil, fmt.Errorf("ExecQuery Failed: %v", err)
		}
		result := resultRaw.ToIDispatch()
		defer result.Release()

		countVar, err := oleutil.GetProperty(result, "Count")
		if err != nil {
			log.Println("Get result count Failed: ", err)
			return nil, fmt.Errorf("Get result count Failed: %v", err)
		}
		count := int(countVar.Val)

		resultMap := []map[string]string{}

		for i := 0; i < count; i++ {
			itemMap := make(map[string]string)

			itemRaw, err := oleutil.CallMethod(result, "ItemIndex", i)
			if err != nil {
				return nil, fmt.Errorf("ItemIndex Failed: %v", err)
			}

			item := itemRaw.ToIDispatch()
			defer item.Release()

			for _, field := range fields {
				asString, err := oleutil.GetProperty(item, field)

				if err == nil {
					itemMap[field] = fmt.Sprintf("%v", asString.Value())
				} else {
					fmt.Println(err)
				}
			}

			resultMap = append(resultMap, itemMap)
		}

		// log.Println("resultMap: ", resultMap)

		return resultMap, nil
	} else {
		log.Println("win_wmi_collector c.service is nil")
		return nil, fmt.Errorf("win_wmi_collector c.service is nil")
	}
}

func (c *win_wmi_collector) c_win_wmi_parse_metric_key(metric string, data map[string]string) (string, error) {
	if strings.Contains(metric, "{{") {
		return utils.ExecuteTemplate(metric, data, newcore.NormalizeMetricKey)
	} else {
		return metric, nil
	}

}

func (c *win_wmi_collector) c_win_wmi_parse_tags(tags map[string]string, data map[string]string) (map[string]string, error) {
	res := map[string]string{}

	for key, tag := range tags {
		if strings.Contains(tag, "{{") {
			tag_value, err := utils.ExecuteTemplate(tag, data, newcore.NormalizeTag)
			if err != nil {
				return res, err
			}
			res[key] = tag_value
		} else {
			res[key] = tag
		}
	}
	return res, nil
}

func (c *win_wmi_collector) get_fields_of_query(query config.Config_win_wmi_query) []string {
	fields := map[string]bool{}
	for _, item := range query.Metrics {
		if len(item.Value_from) > 0 {
			fields[item.Value_from] = true
		}

		for _, f := range win_wmi_pat_field.FindAllString(string(item.Metric), -1) {
			key := f[3 : len(f)-2]
			if len(key) > 0 {
				fields[key] = true
			}

		}

		for _, value := range item.Tags {
			// fmt.Println("item.Tags.value: ", value)
			for _, f := range win_wmi_pat_field.FindAllString(value, -1) {
				key := f[3 : len(f)-2]
				if len(key) > 0 {
					fields[key] = true
				}
			}
		}
	}

	results := []string{}
	for key, _ := range fields {
		results = append(results, key)
	}

	// fmt.Println("results: ", results)

	return results
}

func (c *win_wmi_collector) CollectOnce() (res *newcore.CollectResult) {
	var items newcore.MultiDataPoint

	for _, query := range c.config.Queries {

		fields := c.get_fields_of_query(query)

		results, err := c.query(query.Query, fields)

		// results, err := c.query(
		// 	"select Name, FileSystem, FreeSpace, Size from Win32_LogicalDisk where MediaType=11 or mediatype=12",
		// 	[]string{"Name", "FileSystem", "FreeSpace", "Size"},
		// )

		if err != nil {
			continue
		}

		if len(results) > 0 {
			for _, record := range results {
				for _, item := range query.Metrics {

					metric, err := c.c_win_wmi_parse_metric_key(string(item.Metric), record)
					if err != nil {
						fmt.Println(err)
						continue
					}

					tags, err := c.c_win_wmi_parse_tags(item.Tags, record)
					if err != nil {
						fmt.Println(err)
						continue
					}

					tags = newcore.AddTags.Copy().Merge(query.Tags).Merge(tags)

					if value, ok := record[item.Value_from]; ok == true {

						Add(&items, metric, value, tags, "", "", "")

					} else if item.Default != "" {

						Add(&items, metric, item.Default, tags, "", "", "")

					}
				}
			}
		} else {
			for _, item := range query.Metrics {
				if item.Default != "" {
					// no templating support if no data got
					if strings.Contains(string(item.Metric), "{{") {
						continue
					}
					for _, value := range item.Tags {
						if strings.Contains(value, "{{") {
							continue
						}
					}

					tags := newcore.AddTags.Copy().Merge(query.Tags).Merge(item.Tags)

					Add(&items, item.Metric.Clean(), item.Default, tags, "", "", "")
				}
			}
		}
	} // for queries

	return &newcore.CollectResult{
		Collected: &items,
		Next:      time.Now().Add(c.interval),
		Err:       nil,
	}
}
