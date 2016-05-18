package applog

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sysvipc"
	"time"
)

type KpiFile struct {
	last_flush int64
	counters   map[string]int64
}

type AlarmFile struct {
	last_flush int64
}

/*Reset and Initial counters map with configured kpi oid as keys and 0 as value*/
func (self *KpiFile) reset() error {
	if g_log_cfg == nil {
		return errors.New("failed to build valid kpi oid map from empty oid map")
	}
	self.counters = make(map[string]int64)
	if g_log_cfg.KpiOid == nil {
		return nil
	}
	for _, v := range g_log_cfg.KpiOid {
		self.counters[v] = 0
	}
	return nil
}

/*generate a new KpiFile*/
func NewKpiFile() (*KpiFile, error) {
	kc := &KpiFile{}
	err := kc.reset()
	if err != nil {
		return nil, err
	}
	return kc, nil
}

/*process kpi record in mq, add up to KpiFile*/
func (self *KpiFile) Process() error {
	var str string
	var i int64
	for {
		b, _, err := getKpiRec()
		if err != nil {
			break
		}
		line := string(b)
		//Db("KpiFile::Process getKpiRec [%s][%d][%v]", string(b), l, b)
		sv := strings.Split(line, "|")
		if sv != nil && len(sv) == 2 {
			str = sv[0]
			fmt.Sscanf(sv[1], "%d", &i)
			c, present := self.counters[str]
			if present {
				self.counters[str] = c + i
			} else {
				WriteLog(WARN, NO_ALARM, "ProcessKpi encouter unknown oid [%s] [%d]", str, i)
			}
		} else {
			WriteLog(WARN, NO_ALARM, "ProcessKpi encouter invalid record [%s]", line)
		}
	}
	return nil
}

/* flush KpiFile to file */
func (self *KpiFile) Flush() error {
	/*20160514041503|kpi_collector|KPI|1.3.6.1.4.1.193.176.10.2.1.0|360000*/
	now := time.Now().Unix()
	if now-self.last_flush >= g_log_cfg.KpiInterval && now%g_log_cfg.KpiInterval < 2 {
		self.last_flush = now

		filename, err := GenerateFileName("KPI")
		keys := make([]string, len(self.counters))
		i := 0
		for k, _ := range self.counters {
			keys[i] = k
			i++
		}
		sort.Strings(keys) //sort the oids to get ordered output
		tmp := filepath.Join(g_log_cfg.AlarmKpiPath, ".kpi.tmp")
		target := filepath.Join(g_log_cfg.AlarmKpiPath, filename)
		f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return errors.New(fmt.Sprintf("KpiFile::Flush falied:", err))
		}
		defer func() {
			f.Close()
			Info("Flush KPI file [%s]", target)
			os.Rename(tmp, target)
		}()

		ts := time.Now().Format("20060102150405")
		for _, k := range keys {
			v, present := self.counters[k]
			if present {
				f.Write([]byte(fmt.Sprintf("%s|kpi_collector|KPI|%s|%d\n", ts, k, v)))
			}
		}
		self.reset()
	}
	return nil
}

/* Process alarm record from MQ, write to tmp file */
func (self *AlarmFile) Process() error {
	tmp := filepath.Join(g_log_cfg.AlarmKpiPath, ".alarm.tmp")
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return errors.New(fmt.Sprintf("ProcessAlarm falied to open tmp file for writing:", err))
	}
	defer f.Close()
	for {
		b, _, err := getAlarmRec()
		if err != nil {
			break
		}
		//Db("AlarmFile::Process getAlarmRec [%s][%d][%v]", string(b[0:l]), l, b)
		f.Write(b)
		f.Write([]byte("\n"))
	}
	return nil
}

/* Commit tmp alarm file to alarm interface file */
func (self *AlarmFile) Flush() error {
	now := time.Now().Unix()
	if now-self.last_flush >= g_log_cfg.AlarmInterval && now%g_log_cfg.AlarmInterval < 3 {
		self.last_flush = now

		filename, err := GenerateFileName("WARNING")
		if err != nil {
			return errors.New(fmt.Sprintf("FlushAlarmFile falied:", err))
		}
		target := filepath.Join(g_log_cfg.AlarmKpiPath, filename)

		tmp := filepath.Join(g_log_cfg.AlarmKpiPath, ".alarm.tmp")
		f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return errors.New(fmt.Sprintf("FlushAlarmFile falied:", err))
		}
		not_empty := false
		defer func() {
			f.Close()
			if not_empty {
				Info("Write WARNING file [%s]", target)
				os.Rename(tmp, target)
			}
		}()

		info, err := f.Stat()
		if err != nil {
			return errors.New(fmt.Sprintf("FlushAlarmFile falied:", err))
		}
		if info.Size() > 0 {
			not_empty = true
		}
	}
	return nil
}

func ProcLogRecCycle(w io.Writer, rec_per_cycle int) error {
	for i := 0; i < rec_per_cycle; i++ {
		b, _, err := getLogRec(false) //wait for msg
		if err != nil {
			return err
		}
		w.Write(b)
	}
	return nil
}

/*Get a Kpi Record from MQ
return with the bytes, length, error*/
func getKpiRec() ([]byte, int64, error) {
	if g_log_cfg == nil {
		return nil, 0, errors.New("getKpiRec failed, MQ not initialized")
	}
	return g_mq.Receive(1024, KPI_MSG_TYPE, &sysvipc.MQRecvFlags{true, true})
}

/*Get a Alarm Record from MQ
return with the bytes, length, error*/
func getAlarmRec() ([]byte, int64, error) {
	if g_log_cfg == nil {
		return nil, 0, errors.New("getAlarmRec failed, MQ not initialized")
	}
	return g_mq.Receive(1024, ALARM_MSG_TYPE, &sysvipc.MQRecvFlags{true, true})
}

/*Get a Log Record from MQ
return with the bytes, length, error*/
func getLogRec(nowait bool) ([]byte, int64, error) {
	if g_log_cfg == nil {
		return nil, 0, errors.New("getAlarmRec failed, MQ not initialized")
	}
	return g_mq.Receive(1024, LOG_MSG_TYPE, &sysvipc.MQRecvFlags{nowait, true})
}
