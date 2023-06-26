package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/devusSs/steamquery-v2/config"
	"github.com/devusSs/steamquery-v2/logging"
	"github.com/devusSs/steamquery-v2/query"
	"github.com/devusSs/steamquery-v2/system"
	"github.com/devusSs/steamquery-v2/tables"
	"github.com/devusSs/steamquery-v2/updater"
)

func main() {
	startTime := time.Now().Local()

	cfgPathFlag := flag.String("c", "./files/config.json", "sets the config path")
	gCloudPathFlag := flag.String("g", "./files/gcloud.json", "sets the gcloud config path")
	debugFlag := flag.Bool("d", false, "sets the app to debugging mode")
	versionFlag := flag.Bool("v", false, "prints version and build info and exits")
	logDirFlag := flag.String("l", "./logs", "sets the logging directory")
	disableUpdatesFlag := flag.Bool("du", false, "disables update check on startup")
	analysisModeFlag := flag.Bool("a", false, "runs the app in analysis mode and exits")
	flag.Parse()

	if *analysisModeFlag {
		if err := system.RunAnalysisMode(*logDirFlag, *cfgPathFlag, *gCloudPathFlag); err != nil {
			log.Fatalf("Error running analysis mode: %s\n", err.Error())
		}
		return
	}

	if !*disableUpdatesFlag {
		if err := updater.CheckForUpdatesAndApply(); err != nil {
			log.Fatalf("Error checking for updates: %s\n", err.Error())
		}
	}

	system.InitClearFunc()

	if err := logging.CreateLogsDirectory(*logDirFlag); err != nil {
		log.Fatalf("Error creating logs directory: %s\n", err.Error())
	}

	if *debugFlag {
		if err := logging.InitLoggers("dev"); err != nil {
			log.Fatalf("Error initiating loggers: %s\n", err.Error())
		}
	} else {
		if err := logging.InitLoggers("release"); err != nil {
			log.Fatalf("Error initiating loggers: %s\n", err.Error())
		}
	}

	if *versionFlag {
		updater.PrintBuildInfo()
		return
	}

	if *debugFlag {
		logging.LogWarning("Running debug mode, app might be unstable")

		updater.PrintBuildInfo()
	}

	if *disableUpdatesFlag {
		logging.LogWarning("Skipped update check because of -du flag")
	}

	cfg, err := config.LoadConfig(*cfgPathFlag)
	if err != nil {
		logging.LogFatal(err.Error())
	}

	if err := cfg.CheckConfig(); err != nil {
		logging.LogFatal(err.Error())
	}

	if err := system.CheckForGCloudConfigFile(*gCloudPathFlag); err != nil {
		logging.LogFatal(err.Error())
	}

	svc, err := tables.NewSpreadsheetService(*gCloudPathFlag, cfg.SpreadSheetID)
	if err != nil {
		logging.LogFatal(err.Error())
	}

	if err := svc.TestConnection(); err != nil {
		logging.LogFatal(err.Error())
	}

	query.InitQuery(
		svc,
		cfg.ItemList,
		cfg.PriceColumn,
		cfg.PriceTotalColumn,
		cfg.AmountColumn,
		cfg.OrgCells,
		cfg.SteamAPIKey,
	)

	if err := query.RunQuery(); err != nil {
		if strings.Contains(err.Error(), "last run has been less than 3 minutes ago") {
			logging.LogFatal(err.Error())
		}

		if strings.Contains(err.Error(), "last error has been less than 3 minutes ago") {
			logging.LogFatal(err.Error())
		}

		if err := query.WriteErrorCell(fmt.Errorf("%s (TS: %s)", err.Error(), time.Now().Local().Format("2006-01-02 15:04:05 CEST"))); err != nil {
			logging.LogFatal(err.Error())
		}
	} else {
		if err := query.WriteNoErrorCell(); err != nil {
			logging.LogFatal(err.Error())
		}
	}

	logging.LogSuccess("Done, exiting app now")

	logging.LogInfo(
		fmt.Sprintf("Programm execution took %.2f second(s)", time.Since(startTime).Seconds()),
	)

	// App exit
	if err := logging.CloseLogFiles(); err != nil {
		log.Fatalf("Error closing log files: %s\n", err.Error())
	}
}
