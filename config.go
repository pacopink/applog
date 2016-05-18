package applog

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

const (
	KPI_MSG_TYPE   = int64(10)
	ALARM_MSG_TYPE = int64(11)
	LOG_MSG_TYPE   = int64(12)
	DEFAULT_MQID   = int64(7888)
	NO_ALARM       = ""
)

type LogCfg struct {
	MQID          int64             `json:"mq_id"`
	LogPath       string            `json:"log_path"`
	AlarmKpiPath  string            `json:"alarm_kpi_path"`
	KpiInterval   int64             `json:"kpi_interval"`
	AlarmInterval int64             `json:"alarm_interval"`
	AlarmOid      map[string]string `jason:"alarm_oid"`
	KpiOid        map[string]string `jason:"kpi_oid"`
	mutex         sync.Mutex
}

func (self *LogCfg) Dump() string {
	return fmt.Sprintf("mq_id[%d]\nlog_path[%s]\nalarm_kpi_path[%s]\nkpi_interval[%d]\nalarm_interval[%d]\nalarm_oid[%v]\nkpi_oid[%v]\n", self.MQID, self.LogPath, self.AlarmKpiPath, self.KpiInterval, self.AlarmInterval, self.AlarmOid, self.KpiOid)
}

func GenerateFileName(pattern string) (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", errors.New(fmt.Sprintf("GenerateFileName failed: %v", err))
	}
	return fmt.Sprintf("%s-%s-%s.txt", hostname, pattern, time.Now().Format("20060102150405")), nil
}

func ValidateFile(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return errors.New(fmt.Sprintf("ValidateFile [%s] %v", path, err))
	}
	f.Close()
	return nil
}

func ValidateDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return errors.New(fmt.Sprintf("ValidateDir: [%s] failed] :%v", dir, err))
	}
	defer d.Close()
	s, err := d.Stat()
	if err != nil {
		return errors.New(fmt.Sprintf("ValidateDir: [%s] failed] :%v", dir, err))
	}
	if !s.IsDir() {
		return errors.New(fmt.Sprintf("ValidateDir: [%s] failed], it is not a dir ", dir))
		errors.New("The path " + dir + " is not a dir")
	}

	//try to create test file under the dir
	test_file := fmt.Sprintf("%s/.%d.test", dir, os.Getpid())
	f, err := os.Create(test_file)
	if err != nil {
		return errors.New(fmt.Sprintf("ValidateDir: [%s] failed], cannot write to the dir :%v ", dir, err))
	}
	defer func() {
		f.Close()
		os.Remove(test_file)
	}()
	return nil
}

func (self *LogCfg) Load(file string) error {
	self.mutex.Lock()
	defer self.mutex.Unlock()

	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	s, err := f.Stat()
	if err != nil {
		return err
	}

	b := make([]byte, s.Size())
	_, err = f.Read(b)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, self)
	if err != nil {
		return err
	}
	err = ValidateDir(self.LogPath)
	if err != nil {
		return err
	}
	err = ValidateDir(self.AlarmKpiPath)
	if err != nil {
		return err
	}
	if self.MQID <= 0 {
		self.MQID = DEFAULT_MQID
	}
	if self.KpiInterval <= 60 {
		self.KpiInterval = 5 * 60 //write kpi stat file per 5 minutes by default
	}
	if self.AlarmInterval <= 1 && self.AlarmInterval >= 60 {
		self.AlarmInterval = 5 //write self file per 5 second by default
	}
	return nil
}

func (self *LogCfg) Save(file string) error {
	self.mutex.Lock()
	defer self.mutex.Unlock()

	b, err := json.MarshalIndent(self, "", "    ")
	if err != nil {
		return err
	}

	f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	f.Write(b)
	return nil
}

func (self *LogCfg) GetAlarmOid(alarm_name string) (oid string, err error) {
	if alarm_name == NO_ALARM {
		return NO_ALARM, nil
	}
	self.mutex.Lock()
	defer self.mutex.Unlock()
	oid, present := self.AlarmOid[alarm_name]
	if present {
		return oid, nil
	}
	//try to find default oid if no matched
	oid, present = self.AlarmOid["*"]
	if present {
		return oid, nil
	} else {
		return "", errors.New(fmt.Sprintf("not found oid via self [%s]", alarm_name))
	}
}

func (self *LogCfg) GetKpiOid(kpi_name string) (oid string, err error) {
	self.mutex.Lock()
	defer self.mutex.Unlock()

	oid, present := self.KpiOid[kpi_name]
	if present {
		return oid, nil
	} else {
		return "", errors.New(fmt.Sprintf("KPI name [%s] not found oid", kpi_name))
	}
}
