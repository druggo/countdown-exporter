package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Config struct {
	DeadlinesFile          string
	DeadlinesFileType      string
	ExporterListenPort     int
	ThresholdCheckInterval int
}

var timeFormatConstants = map[string]string{
	"ANSIC":       "Mon Jan _2 15:04:05 2006",
	"UnixDate":    "Mon Jan _2 15:04:05 MST 2006",
	"RubyDate":    "Mon Jan 02 15:04:05 -0700 2006",
	"RFC822":      "02 Jan 06 15:04 MST",
	"RFC822Z":     "02 Jan 06 15:04 -0700", // RFC822 with numeric zone
	"RFC850":      "Monday, 02-Jan-06 15:04:05 MST",
	"RFC1123":     "Mon, 02 Jan 2006 15:04:05 MST",
	"RFC1123Z":    "Mon, 02 Jan 2006 15:04:05 -0700", // RFC1123 with numeric zone
	"RFC3339":     "2006-01-02T15:04:05Z07:00",
	"RFC3339Nano": "2006-01-02T15:04:05.999999999Z07:00",
	// Handy time format.
	"Stamp":       "Jan _2 15:04:05",
	"DateTime":    "2006-01-02 15:04:05",
	"DateOnly":    "2006-01-02",
	"TimeOnly":    "15:04:05",
}

var (
	Countdowns = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "countdown_timers",
		Help: "Countdowns have exceeded threshold",
	},
		[]string{"countdown", "description", "expired", "deadline", "deadline_time_format", "threshold", "threshold_type", "threshold_tripped"})
)

var (
	DefaultDeadlineTimeFormat = "DateOnly"
	DefaultThreshold          = 7
	DefaultThresholdType      = "days"
)

type DeadlinesConfig struct {
	Deadlines []Deadlines `yaml:"deadlines,flow" json:"deadlines,flow"`
}

type Deadlines struct {
	Name               string `yaml:"name" json:"name"`
	Description        string `yaml:"description" json:"description"`
	DeadlineTime       string `yaml:"deadline-time" json:"deadline-time"`
	DeadlineTimeFormat string `yaml:"deadline-time-format" json:"deadline-time-format"`
	Threshold          int    `yaml:"threshold" json:"threshold"`
	ThresholdType      string `yaml:"threshold-type" json:"threshold-type"`
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func checkThreshold(deadlineTime, deadlineTimeFormat string, threshold int, thresholdType string) bool {
	var dt time.Time
	switch strings.ToUpper(thresholdType) {
	case "YEAR":
		fallthrough
	case "YEARS":
		fallthrough
	case "Y":
		dt = time.Now().AddDate(threshold, 0, 0)

	case "MONTH":
		fallthrough
	case "MONTHS":
		fallthrough
	case "MO":
		dt = time.Now().AddDate(0, threshold, 0)

	case "DAY":
		fallthrough
	case "DAYS":
		fallthrough
	case "D":
		dt = time.Now().AddDate(0, 0, threshold)

	case "HOUR":
		fallthrough
	case "HOURS":
		fallthrough
	case "H":
		dt = time.Now().Add(time.Hour * time.Duration(threshold))

	case "MINUTE":
		fallthrough
	case "MINUTES":
		fallthrough
	case "MIN":
		fallthrough
	case "MINS":
		fallthrough
	case "M":
		dt = time.Now().Add(time.Minute * time.Duration(threshold))

	case "SECOND":
		fallthrough
	case "SECONDS":
		fallthrough
	case "SEC":
		fallthrough
	case "SECS":
		fallthrough
	case "S":
		dt = time.Now().Add(time.Second * time.Duration(threshold))
	}

	var timeFormat string

	if _, ok := timeFormatConstants[deadlineTimeFormat]; ok {
		timeFormat = timeFormatConstants[deadlineTimeFormat]
	} else {
		timeFormat = deadlineTimeFormat
	}

	expireTime, err := time.Parse(timeFormat, deadlineTime)
	if err != nil {
		log.Printf("Error parsing deadline timestamp: %s\n", err)
	}

	return dt.After(expireTime)

}

func checkExpired(deadlineTime, deadlineTimeFormat string) bool {
	dt := time.Now()
	var expireTime time.Time
	var err error
	if deadlineFormat, ok := timeFormatConstants[deadlineTimeFormat]; ok {
		expireTime, err = time.Parse(deadlineFormat, deadlineTime)
		if err != nil {
			log.Printf("Error parsing timestamp from map: %v", err)
		}
	} else {
		expireTime, err = time.Parse(deadlineTimeFormat, deadlineTime)
		if err != nil {
			log.Printf("Error parsing timestamp: %v\n", err)
		}
	}

	return dt.After(expireTime)
}

func checkTimers(d *DeadlinesConfig) {
	for _, deadline := range d.Deadlines {
		if checkThreshold(deadline.DeadlineTime, deadline.DeadlineTimeFormat, deadline.Threshold, deadline.ThresholdType) && checkExpired(deadline.DeadlineTime, deadline.DeadlineTimeFormat) {
			Countdowns.WithLabelValues(deadline.Name, deadline.Description, "true", deadline.DeadlineTime, deadline.DeadlineTimeFormat, strconv.FormatInt(int64(deadline.Threshold), 10), deadline.ThresholdType, "true").Set(1)
		} else if checkThreshold(deadline.DeadlineTime, deadline.DeadlineTimeFormat, deadline.Threshold, deadline.ThresholdType) && !checkExpired(deadline.DeadlineTime, deadline.DeadlineTimeFormat) {
			Countdowns.WithLabelValues(deadline.Name, deadline.Description, "false", deadline.DeadlineTime, deadline.DeadlineTimeFormat, strconv.FormatInt(int64(deadline.Threshold), 10), deadline.ThresholdType, "true").Set(1)
			// below conditional shouldn't be met unless a threshold is set to a negative number
		} else if !checkThreshold(deadline.DeadlineTime, deadline.DeadlineTimeFormat, deadline.Threshold, deadline.ThresholdType) && checkExpired(deadline.DeadlineTime, deadline.DeadlineTimeFormat) {
			Countdowns.WithLabelValues(deadline.Name, deadline.Description, "true", deadline.DeadlineTime, deadline.DeadlineTimeFormat, strconv.FormatInt(int64(deadline.Threshold), 10), deadline.ThresholdType, "false").Set(1)
		} else if !checkThreshold(deadline.DeadlineTime, deadline.DeadlineTimeFormat, deadline.Threshold, deadline.ThresholdType) && !checkExpired(deadline.DeadlineTime, deadline.DeadlineTimeFormat) {
			Countdowns.WithLabelValues(deadline.Name, deadline.Description, "false", deadline.DeadlineTime, deadline.DeadlineTimeFormat, strconv.FormatInt(int64(deadline.Threshold), 10), deadline.ThresholdType, "false").Set(1)
		}
	}
}

func initialize(c *Config) {
	var err error
	// Set exporter configuration from environment variables
	c.DeadlinesFile = getEnv("COUNTDOWN_EXPTR_DEADLINES_FILE", "deadlines.yaml")
	c.DeadlinesFileType = getEnv("COUNTDOWN_EXPTR_DEADLINES_FILE_TYPE", "yaml")
	exporterListenPort := getEnv("COUNTDOWN_EXPTR_HTTP_PORT", "9208")
	c.ExporterListenPort, err = strconv.Atoi(exporterListenPort)
	if err != nil {
		log.Fatalf("Error converting COUNTDOWN_EXPTR_HTTP_PORT env var to int: %v\n", err)
	}
	thresholdCheckInterval := getEnv("COUNTDOWN_EXPTR_CHECK_INTERVAL_SECS", "60")
	c.ThresholdCheckInterval, err = strconv.Atoi(thresholdCheckInterval)
	if err != nil {
		log.Fatalf("Error converting COUNTDOWN_EXPTR_CHECK_INTERVAL_SECS env var to int: %v\n", err)
	}

	// Register prometheus metric and start http server
	prometheus.MustRegister(Countdowns)
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(fmt.Sprintf(":%d", c.ExporterListenPort), nil)
}

func readDeadlines(d *DeadlinesConfig, config *Config) {
	file, err := ioutil.ReadFile(config.DeadlinesFile)
	if err != nil {
		log.Printf("Error reading file: %s", err)
	}
	if strings.ToUpper(config.DeadlinesFileType) == "YAML" {
		err = yaml.Unmarshal(file, d)
		if err != nil {
			log.Fatalf("Error unmarshalling deadlines yaml config\n")
		}
		setDefaults(d)
	} else if strings.ToUpper(config.DeadlinesFileType) == "JSON" {
		err = json.Unmarshal(file, d)
		if err != nil {
			log.Fatalf("Error unmarshalling deadlines json config\n")
		}
		setDefaults(d)
	}
}

// TODO: get rid of this function
func setDefaults(d *DeadlinesConfig) {
	for i, deadline := range d.Deadlines {
		if deadline.DeadlineTimeFormat == "" {
			d.Deadlines[i].DeadlineTimeFormat = DefaultDeadlineTimeFormat
		}
		if deadline.Threshold == 0 {
			d.Deadlines[i].Threshold = DefaultThreshold
		}
		if deadline.ThresholdType == "" {
			d.Deadlines[i].ThresholdType = DefaultThresholdType
		}
	}
}

func listenForSignal(d *DeadlinesConfig, config *Config) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP)
	go func(d *DeadlinesConfig, config *Config) {
		for {
			_ = <-sigs
			readDeadlines(d, config)
			// Reset existing prom metrics for garbage collection
			prometheus.Unregister(Countdowns)
			Countdowns = prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Name: "countdown_timers",
				Help: "Countdowns have exceeded threshold",
			},
				[]string{"countdown", "description", "expired", "deadline", "deadline_time_format", "threshold", "threshold_type", "threshold_tripped"})
			prometheus.MustRegister(Countdowns)
			checkTimers(d)
		}
	}(d, config)
}

func main() {

	config := &Config{}
	d := &DeadlinesConfig{}

	initialize(config)
	readDeadlines(d, config)
	listenForSignal(d, config)

	for {
		checkTimers(d)
		time.Sleep(time.Duration(config.ThresholdCheckInterval) * time.Second)
	}
}
