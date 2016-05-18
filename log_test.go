package applog

import (
	"fmt"
	"testing"
)

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

func TestWriteLog(t *testing.T) {
	err := LoadLogCfg("./log_aggregator/example.cfg")
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
}

func TestWrite2MQ(t *testing.T) {
	err := LoadLogCfg("./log_aggregator/example.cfg")
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
