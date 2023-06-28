package logging

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/fatih/color"
)

var (
	consoleLogger *log.Logger

	DebugSign = color.WhiteString("[DEBUG]")
	InfSign   = color.CyanString("[INFO]")
	WarnSign  = color.YellowString("[WARN]")
	ErrSign   = color.RedString("[ERROR]")
	SucSign   = color.GreenString("[SUCCESS]")

	logLevel      string
	logsDirectory string

	appLogFile   *os.File
	errorLogFile *os.File
)

func CreateLogsDirectory(dir string) error {
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(dir, os.ModePerm)
		if err != nil {
			return err
		}
	}
	logsDirectory = dir
	return nil
}

func InitLoggers(level string) error {
	logLevel = level
	consoleLogger = log.New(os.Stdout, "", 0)

	aFile, err := os.Create(fmt.Sprintf("%s/app.log", logsDirectory))
	if err != nil {
		return err
	}

	eFile, err := os.Create(fmt.Sprintf("%s/error.log", logsDirectory))
	if err != nil {
		return err
	}

	appLogFile = aFile
	errorLogFile = eFile

	return nil
}

func CloseLogFiles() error {
	if err := appLogFile.Close(); err != nil {
		return err
	}
	return errorLogFile.Close()
}

func LogDebug(message string) {
	if logLevel != "release" {
		consoleLogger.Printf("%s %s\n", DebugSign, message)
	}
	_, err := appLogFile.Write(
		[]byte(fmt.Sprintf("%s - %s %s\n", time.Now().String(), InfSign, message)),
	)
	if err != nil {
		log.Println(err)
	}
}

func LogInfo(message string) {
	consoleLogger.Printf("%s %s\n", InfSign, message)

	_, err := appLogFile.Write(
		[]byte(fmt.Sprintf("%s - %s %s\n", time.Now().String(), InfSign, message)),
	)
	if err != nil {
		log.Println(err)
	}
}

func LogWarning(message string) {
	consoleLogger.Printf("%s %s\n", WarnSign, message)

	_, err := appLogFile.Write(
		[]byte(fmt.Sprintf("%s - %s %s\n", time.Now().String(), WarnSign, message)),
	)
	if err != nil {
		log.Println(err)
	}
}

func LogError(message string) {
	consoleLogger.Printf("%s [non-critical] %s\n", ErrSign, message)

	_, err := errorLogFile.Write(
		[]byte(fmt.Sprintf("%s - %s %s\n", time.Now().String(), ErrSign, message)),
	)
	if err != nil {
		log.Println(err)
	}
}

func LogFatal(message string) {
	consoleLogger.Printf("%s [critical] %s\n", ErrSign, message)

	_, err := errorLogFile.Write(
		[]byte(fmt.Sprintf("%s - %s %s\n", time.Now().String(), ErrSign, message)),
	)
	if err != nil {
		log.Println(err)
	}

	os.Exit(1)
}

func LogSuccess(message string) {
	consoleLogger.Printf("%s %s\n", SucSign, message)

	_, err := appLogFile.Write(
		[]byte(fmt.Sprintf("%s - %s %s\n", time.Now().String(), SucSign, message)),
	)
	if err != nil {
		log.Println(err)
	}
}
