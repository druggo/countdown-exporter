# countdown-exporter
A countdown (timer) exporter for Prometheus


### Concepts
* Deadline - countdown timer _expiration_ time.
* Threshold - configured prior to deadline (can be used as an countdown expiry warning)


### Deadlines/Threshold Configuration
The exporter expects either a YAML or JSON file with a single key, `deadlines`, whose value is an array of objects containing these fields:
* `name` - timer name
  * type: string
* `description` - timer description
  * type: string
* `deadline-time` - timer expiry time
  * type: string
* `deadline-time-format` - [golang time format string](https://golang.org/pkg/time/#Time.Format) (Example golang time format [constants](https://golang.org/pkg/time/#pkg-constants))
  * type: string
  * default: RFC3339
* `threshold` - threshold quantity
  * type: int
  * default: days
* `threshold-type` - threshold type from one of: `years`, `months`, `days`, `hours`, `minutes`, `seconds`
  * type: string
  * default: RFC3339

### Exported Prometheus Metric Reference:
* Name: `countdown_timers`
* Labels:
  * `countdown` - countdown timer name 
  * `description` - timer description
  * `expired` - deadline expired status (true or false)
  * `deadline` - deadline/expiry timestamp
  * `deadline_time_format` - deadline timestamp format
  * `threshold` - threshold quantity
  * `threshold_type` - threshold type (one of: `years`, `months`, `days`, `hours`, `minutes`, `seconds`)
  * `threshold_tripped` - threshold exceeded status (true or false)


### Exporter Configuration
The exporter can be configured with these environment variables:
* `COUNTDOWN_EXPTR_DEADLINES_FILE`
  * default: deadlines.yaml
* `COUNTDOWN_EXPTR_DEADLINES_FILE_TYPE`
  * default: yaml
* `COUNTDOWN_EXPTR_HTTP_PORT` 9208
  * default: 9208
* `COUNTDOWN_EXPTR_CHECK_INTERVAL_SECS`
  * default: 60
  
### Example Deadlines Config File
[deadlines.yaml](https://github.com/tmegow/countdown-exporter/blob/master/deadlines.yaml)
