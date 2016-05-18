# Concept
In a scalable system, there will be serveral processes co-operating to fulfill some business purpose. this project trys to provide a gerenal logging/alarm/fault mechanism for scalable systems.

Using this library:

* Applications shall be able to write log to file or consolidate all application logs to a single place, in a unified format.
* Application can easily generate alarm via calling some APIs and store it somewhere, and let the log_aggregator to collector it and generate file to integrate with **Fault Management system**.
* Application can easily do KPI statics via calling some APIs and store it somewhere, and let the log_aggregator to collector it and generate file to integrate with **Performance Management system**.

### Log sample
```
#Timestamp|Application Label|LogLevel|Log Content
20160518-150820.047|APPLICATION002|DEBUG|debug log [10234]
20160518-150820.047|APPLICATION002|INFO|info log [10238]
20160518-150820.047|APPLICATION002|ERROR|Failed to connect to db [localhost:3868]
20160518-151821.318|APPLICATION002|DEBUG|this is a debug to MQ: 10, debug string
20160518-151821.318|APPLICATION002|DEBUG|debug log [10234]
20160518-151821.318|APPLICATION002|INFO|info log [10238]
```
### KPI file sample
```
$cat DccProxy01-KPI-20160518152500.txt
#timestamp|fixed label1|fixed label2|oid|counter value
20160518152500|kpi_collector|KPI|1.3.1.2.1|201
20160518152500|kpi_collector|KPI|1.3.1.2.3|-6
20160518152500|kpi_collector|KPI|1.3.1.2.4|1
```

### WARNING file sample
```
$cat DccProxy01-WARNING-20160518162716.txt
#timestamp|Application Label|LogLevel|oid|alarm msg
20160518162713|APPLICATION001|ERROR|.1.3.1.1.1|Failed to connect to db [localhost:3868]
20160518162713|APPLICATION002|ERROR|.1.3.1.1.1|Failed to connect to db [localhost:3868]
```

# Dependencies
Depending on the  https://github.com/teepark/go-sysvipc.git

Encouting some problem when 'go test' this library on the sem.go, as only the message queue is used, I simply remove sem\*.go and shm\*.go, and simply modify the common_test.go as below.

```
$git diff common_test.go
diff --git a/common_test.go b/common_test.go
index 2ac1299..c1d5746 100644
--- a/common_test.go
+++ b/common_test.go
@@ -6,7 +6,7 @@ import (
 )
 
 func TestFtok(t *testing.T) {
-       for _, path := range []string{"common.go", "shm.go", "msg.go", "sem.go"} {
+       for _, path := range []string{"common.go", "msg.go"} {
                key, err := Ftok(path, '+')
                if err != nil {
                        t.Fatal(err)
```

# Configuration
## Config file
config file shall be in JSON format.
Application shall call the 
```
func LoadLogCfg(config_file string) error
```
api to load a specified config file.

Example Config:
```
#Refer to log_aggregator/example.cfg
{
    "mq_id" : 7888,                    //MQID for IPC
    "log_path" : "/ocg/applog",        //The path to write log file
    "alarm_kpi_path": "/ocg/applog",   //The path to write KPI and Alarm file
    "kpi_interval" : 300,              //Seconds of KPI file flush interval
    "alarm_interval" : 5,              //Seconds of WARNING file flush interval

    "AlarmOid": {                     //Alarm name string to oid mapping
        "CONN_FAIL": "1.3.1.1.2",
        "NETWORK_FAIL" : "1.3.1.1.3",
        "DB_FAIL": "1.3.1.1.1",
        "*":"1.3.1.1.9999"            //"*" is the default oid
    },

    "KpiOid": {                       //KPI name string to oid mapping
        "REQ_COUNT": "1.3.1.2.1", 
        "RES_COUNT": "1.3.1.2.3",
        "ABNORMAL_COUNT": "1.3.1.2.4"
    }
}
```

for log_aggregator, the config file fullpath shall be provided via the 'APP_LOG_CFG' env or '-c' argument

## Common Flag
DebugFlag to control if write log in Db level

StdoutFlag to control if tee log message to stdout 



# 3 Modes
## Simple Mode
you don't need to initial applog, just simply use 
Db
Info
WriteLog

no KPI and Alarm APIs are usable at this mode 

```
flowchart
Application-->Logger
Logger-->Stdout
```

```golang
//Example: log_test.go/TestSimpleLog
func TestSimpleLog(t *testing.T) {
    DebugLog(true)
    Db("this is a debug to stdout only: %d, %s", 10, "test string")
    Info("this is a info to stdout only: %d, %s", 12, "test string2")
    WriteLog(ERROR, "DB_FAIL", "Failed to connect to db [%s:%d]", "localhost", 3868)
    err := IncreaseKpi("REQ_COUNT")
    if err == nil {
        t.Fatalf("IncreaseKpi in uninitialized logger shall get error but nil")
    }
    err = DecreaseKpi("REQ_COUNT")
    if err == nil {
        t.Fatalf("DecreaseKpi in uninitialized logger shall get error but nil")
    }
}
```

## Pure SysV Message Queue Mode
you need to InitLog with filename "mq" or "MQ", in this mode, application will not write log to file but write to the SysV Message Queue, and the 'log_aggregator' shall be run to collect Log/Kpi/Alarms from the MQ, distinguish the 3 types of msg via msg type.
The log_aggregator will:
write to 'app.log' files for logs.
write to [hostname]-KPI-[YYYYMMDDHHMI].txt files for kpis via preconfigured interval.
write to [hostname]-WARNING-[YYYYMMDDHHMI].txt files for alarms via preconfigured interval

This mode provide a consolidate log file for multiple application processes.
the app.log file will switch per day, and rename the last day file as app.log.[YYYYMMDD], YYYYMMDD is last day date.

```
graph LR
Application-->Logger
Logger-->MQ_LOG
Logger-->MQ_KPI
Logger-->MQ_ALARM
MQ_LOG-->log_aggregator
MQ_KPI-->log_aggregator
MQ_ALARM-->log_aggregator
log_aggregator-->app.log
log_aggregator-->kpi.txt
log_aggregator-->alarm.txt
```

```golang
//Example: log_test.txt/TestWrite2MQ
func TestWrite2MQ(t *testing.T) {
    err := LoadLogCfg("./oid2.cfg")
    if err != nil {
        t.Fatalf("LoadLogCfg: %v", err)
    }
    err = InitLog("mq", "APPLICATION002")
    if err != nil {
        t.Fatalf("InitLog: %v", err)
    }
    DebugLog(true)
    StdoutLog(true)

    Db("this is a debug to MQ: %d, %s", 10, "debug string")
    WriteLog(DEBUG, "THEDEBUG", "%s [%d]", "debug log", 10234)
    WriteLog(INFO, "THEINFO", "%s [%d]", "info log", 10238)
    WriteLog(ERROR, "DB_FAIL", "Failed to connect to db [%s:%d]", "localhost", 3868)
    WriteKpi("REQ_COUNT", 100)
    WriteKpi("RES_COUNT", -3)
    DebugLog(false)
    Db("call db to MQ: %d, %s, but this should be suppressed via DebugLog(false)", 10, "debug string")
    Db("call db to MQ: %d, %s, but this should be suppressed via DebugLog(false)", 10, "debug string")
    Db("call db to MQ: %d, %s, but this should be suppressed via DebugLog(false)", 10, "debug string")
}
```

## Log to application file, ALARM, KPI to MQ
if InitLog with the actual log filename, the application will write log message to the specified file.

It is the user's responsibility to switch log file, you can simply rename the old log file to have a switch.

```
graph LR
Application-->Logger
Logger-->application.log
Logger-->MQ_KPI
Logger-->MQ_ALARM
MQ_KPI-->log_aggregator
MQ_ALARM-->log_aggregator
log_aggregator-->kpi.txt
log_aggregator-->alarm.txt
```

```golang
//Example: log_test.log/TestWriteLog

    err := LoadLogCfg("./oid2.cfg")
    if err != nil {
        t.Fatalf("LoadLogCfg: %v", err)
    }
    err = InitLog("log_test.log", "APPLICATION001")
    if err != nil {
        t.Fatalf("InitLog: %v", err)
    }
    DebugLog(true)
    StdoutLog(true)
    fmt.Println(DumpLog())
    WriteLog(DEBUG, "THEDEBUG", "%s [%d]", "debug log", 10234)
    WriteLog(INFO, "THEINFO", "%s [%d]", "info log", 10238)
    DebugLog(false)
    WriteLog(DEBUG, "THEDEBUG", "%s [%d]", "debug log", 10234)
    WriteLog(INFO, "THEINFO", "%s [%d]", "info log", 10238)
    WriteLog(ERROR, "DB_FAIL", "Failed to connect to db [%s:%d]", "localhost", 3868)
    err = IncreaseKpi("REQ_COUNT")
    if err != nil {
        t.Fatalf("IncreaseKpi: %v", err)
    }
    err = DecreaseKpi("ABNORMAL_COUNT")
    if err != nil {
        t.Fatalf("DecreaseKpi: %v", err)
    }
    err = WriteKpi("REQ_COUNT", 100)
    if err != nil {
        t.Fatalf("WriteKpi: %v", err)
    }
    err = WriteKpi("RES_COUNT", -3)
    if err != nil {
        t.Fatalf("WriteKpi: %v", err)
    }
    err = WriteKpi("XXXX", -3)
    if err == nil {
        t.Fatalf("WriteKpi to a undefined kpi shall get error but nil here")
    }
```
