package main

import (
	log "applog"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

func LogRoutine(wg *sync.WaitGroup, filename string) {
	defer wg.Done()
	last_day := time.Now().Format("20060102")
	var f *os.File
	rec_per_cycle := 50
	for {
		if f == nil {
			f, _ = os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		}
		//main loop to write log from MQ to file
		err := log.ProcLogRecCycle(f, rec_per_cycle)
		if err != nil {
			fmt.Println("LogRoutine err:", err)
			time.Sleep(1000 * time.Millisecond)
		}
		//check if day pass then trigger a file swtich
		now_day := time.Now().Format("20060102")
		if last_day != now_day {
			if f != nil {
				f.Close()
			}
			os.Rename(filename, filename+"."+last_day)
			last_day = now_day
		}
	}
}

func KpiRoutine(wg *sync.WaitGroup, kc *log.KpiFile) {
	defer wg.Done()
	for {
		kc.Process()
		kc.Flush()
		time.Sleep(100 * time.Millisecond)

	}
}

func AlarmRoutine(wg *sync.WaitGroup, af *log.AlarmFile) {
	defer wg.Done()
	for {
		af.Process()
		af.Flush()
		time.Sleep(500 * time.Millisecond)
	}
}

func main() {
	pcfg := flag.String("c", "", "the alarm & kpi config file in json format")
	global_logfile := flag.String("g_log", "app.log", "the global log filename")
	debug := flag.Bool("d", false, "if turn on debug log")
	stdout := flag.Bool("p", false, "if print log to stdout")
	flag.Parse()
	cfg := os.Getenv("APP_LOG_CFG")
	if len(*pcfg) > 0 { //the argument shall override the env
		cfg = *pcfg
	}
	if len(cfg) <= 0 {
		fmt.Println("Cannot find the log config file. It shall be set in the APP_LOG_CFG env or -c argument")
		os.Exit(1)
	}
	err := log.LoadLogCfg(cfg)
	if err != nil {
		fmt.Println("Load LogCfg", cfg, " failed", err)
		os.Exit(1)
	}
	log.DebugLog(*debug)
	log.StdoutLog(*stdout)
	err = log.InitLog("alarm_kpi_aggregator.log", "AGGREGATOR")
	if err != nil {
		fmt.Println("fail to InitLog:", log.Config().Dump())
		os.Exit(1)
	}

	fullpath := filepath.Join(log.Config().LogPath, *global_logfile)
	err = log.ValidateFile(fullpath)
	if err != nil {
		log.WriteLog(log.ERROR, "APP_START", "Failed to write global log file [%s]: %v. Exit", fullpath, err)
		os.Exit(1)
	}

	kpiCounter, err := log.NewKpiFile()
	if err != nil {
		log.WriteLog(log.ERROR, "APP_START", "Failed to NewKpiFile: %v. Exit", err)
		os.Exit(1)
	}
	alarmFile := &log.AlarmFile{}

	wg := &sync.WaitGroup{}
	wg.Add(3)
	log.WriteLog(log.INFO, "", "Launch LogRoutine")
	go LogRoutine(wg, fullpath)
	log.WriteLog(log.INFO, "", "Launch AlarmRoutine")
	go AlarmRoutine(wg, alarmFile)
	log.WriteLog(log.INFO, "", "Launch KpiRoutine")
	go KpiRoutine(wg, kpiCounter)
	log.WriteLog(log.CLEAN, "APP_START", "application startup normally")
	wg.Wait()
	log.WriteLog(log.INFO, "", "Exit ...")
}
