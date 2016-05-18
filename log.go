package applog

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"sysvipc"
	"time"
)

type LOG_LEVEL uint8

const (
	DEBUG     = LOG_LEVEL(0)
	INFO      = LOG_LEVEL(1)
	CLEAN     = LOG_LEVEL(2)
	EVENT     = LOG_LEVEL(3)
	WARN      = LOG_LEVEL(4)
	ERROR     = LOG_LEVEL(5)
	FATAL     = LOG_LEVEL(6)
	MAX_LEVEL = LOG_LEVEL(7)
)

func Level2Str(l LOG_LEVEL) string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case CLEAN:
		return "CLEAN"
	case EVENT:
		return "EVENT"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNDEFINE"
	}
}

type Logger struct {
	LogPath     string
	LogFilename string
	LogFullpath string
	AppName     string
	mutex       sync.Mutex
}

var g_logger Logger
var g_debug_flag = false
var g_stdout_flag = false
var g_log_cfg *LogCfg
var g_mq sysvipc.MessageQueue
var g_log_available = false

func Config() *LogCfg {
	return g_log_cfg
}

func Log() *Logger {
	return &g_logger
}

func DumpLog() string {
	return fmt.Sprintf("LogPath: %s\nLogFilename: %s\nLogFullpath: %s\nAppName: %s\n", g_logger.LogPath, g_logger.LogFilename, g_logger.LogFullpath, g_logger.AppName)
}

func InitLog(filename string, app_name string) error {
	g_logger.mutex.Lock()
	defer g_logger.mutex.Unlock()

	if len(g_log_cfg.LogPath) < 1 || len(filename) < 1 || len(app_name) < 1 {
		return errors.New(fmt.Sprintf("InitLog: invalid arguments path[%s] filename[%s] app_name[%s]", g_log_cfg.LogPath, filename, app_name))
	}
	/* if filename == "mq"||"MQ“ do not write to file */
	if filename != "mq" || filename != "MQ" {
		err := ValidateFile(g_log_cfg.LogPath + "/" + filename)
		if err != nil {
			return errors.New(fmt.Sprintf("InitLog [%s/%s] failed, %v", g_log_cfg.LogPath, filename, err))
		}
		g_logger.LogPath = g_log_cfg.LogPath
	} else {
		g_logger.LogPath = "mq"
	}
	g_logger.LogFilename = filename
	g_logger.LogFullpath = g_log_cfg.LogPath + "/" + filename
	g_logger.AppName = app_name
	g_log_available = true
	return nil
}

func LoadLogCfg(config_file string) error {
	var cfg LogCfg
	err := cfg.Load(config_file)
	if err != nil {
		return err
	}
	mq, err := sysvipc.GetMsgQueue(cfg.MQID, &sysvipc.MQFlags{true, false, 0660})
	if err != nil {
		return err
	}
	g_log_cfg = &cfg
	g_mq = mq

	return nil
}

func DebugLog(d bool) {
	g_debug_flag = d
}

func StdoutLog(s bool) {
	g_stdout_flag = s
}

func IncreaseKpi(kpi_name string) error {
	return WriteKpi(kpi_name, 1)
}

func DecreaseKpi(kpi_name string) error {
	return WriteKpi(kpi_name, 1)
}

func WriteKpi(kpi_name string, delta int64) error {
	g_logger.mutex.Lock()
	defer g_logger.mutex.Unlock()
	if g_log_cfg == nil {
		return errors.New("WriteKpi failed, mq not initialized")
	}
	oid, err := g_log_cfg.GetKpiOid(kpi_name)
	if err != nil {
		return errors.New("WriteKpi failed, invalid kpi_name " + kpi_name)
	}
	kpi_line := fmt.Sprintf("%s|%d", oid, delta)
	//fmt.Printf("WriteKpi [%s][%v]\n", kpi_line, []byte(kpi_line))
	g_mq.Send(KPI_MSG_TYPE, []byte(kpi_line), &sysvipc.MQSendFlags{true})
	return nil
}

func Db(format string, v ...interface{}) {
	WriteLog(DEBUG, "", format, v...)
}

func Info(format string, v ...interface{}) {
	WriteLog(INFO, "", format, v...)
}

func WriteLog(level LOG_LEVEL, alarm_name string, format string, v ...interface{}) {
	g_logger.mutex.Lock()
	defer g_logger.mutex.Unlock()

	ts := time.Now().Format("20060102-150405.000")
	if !g_debug_flag && level == DEBUG {
		return
	}
	line := ts + "|" + g_logger.AppName + "|" + Level2Str(level) + "|" + fmt.Sprintf(format, v...)
	if g_stdout_flag || !g_log_available { //if no log file available print to stdout
		fmt.Println(line)
	}
	if !g_log_available { //simple mode, only write to stdout
		return
	}

	if g_logger.LogFilename == "mq" {
		err := g_mq.Send(LOG_MSG_TYPE, []byte(line+"\n"), &sysvipc.MQSendFlags{true})
		if err != nil {
			fmt.Printf("log failed to write to mq: %s\n", line)
			return
		}
	} else {
		f, err := os.OpenFile(g_logger.LogFullpath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			//fmt.Printf("WriteLog [%s] failed\n", g_logger.LogFullpath)
			fmt.Println(err)
			return
		}
		defer f.Close()
		f.Write([]byte(line + "\n"))
	}

	////////////// write alarm string to mq ///////////////
	/*20160514030208|DC:AOC_001:C001|ERROR|.1.3.6.1.4.1.193.176.3.4.2|Cannot connect to backup DCC server. ip address: , port: 0. Invalid ip address or port. The current instance of dcc_client is DC:AOC_001:C001*/
	if g_log_cfg != nil && level > INFO && level < MAX_LEVEL {
		oid, err := g_log_cfg.GetAlarmOid(alarm_name)
		if err != nil || oid == NO_ALARM {
			return
		}
		alarm_line := fmt.Sprintf("%s|%s|%s|.%s|%s", time.Now().Format("20060102150405"), g_logger.AppName, Level2Str(level), oid, fmt.Sprintf(format, v...))
		//fmt.Printf("WriteAlarm [%s][%v]\n", alarm_line, []byte(alarm_line))
		g_mq.Send(ALARM_MSG_TYPE, []byte(alarm_line), &sysvipc.MQSendFlags{true})
	}
}
