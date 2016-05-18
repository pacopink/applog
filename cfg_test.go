package applog

import (
	"fmt"
	"testing"
)

func TestSavaAlarm(t *testing.T) {
	am := &LogCfg{
		AlarmOid: make(map[string]string),
	}

	am.AlarmOid["DB_FAIL"] = "1.3.1.1.1"
	am.AlarmOid["CONN_FAIL"] = "1.3.1.1.2"
	am.AlarmOid["*"] = "1.3.1.1.99999999"

	err := am.Save("./oid.cfg")
	if err != nil {
		t.Fatalf("failed to save: %v", err)
	}
	fmt.Println(am)
}

func TestLoadAlarm(t *testing.T) {
	var am LogCfg
	err := am.Load("./oid2.cfg")
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}
	fmt.Println(am)

	fmt.Println(am.GetKpiOid("REQ_COUNT"))
	fmt.Println(am.GetKpiOid("RES_COUNT"))
	fmt.Println(am.GetKpiOid("ERR_COUNT"))
	fmt.Println(am.GetAlarmOid("NETWORK_FAIL"))
	fmt.Println(am.GetAlarmOid("XXXXXXXXXXXX"))
}

func TestGenAlarmFilename(t *testing.T) {
	fmt.Println(GenerateFileName("WARNING"))
	fmt.Println(GenerateFileName("KPI"))
}
