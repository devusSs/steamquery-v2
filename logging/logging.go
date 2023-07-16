package logging

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/fatih/color"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

var (
	consoleLogger *log.Logger

	DebugSign = color.WhiteString("[DEBUG]")
	InfSign   = color.CyanString("[INFO]")
	WarnSign  = color.YellowString("[WARN]")
	ErrSign   = color.RedString("[ERROR]")
	SucSign   = color.GreenString("[SUCCESS]")

	DebugSignNoColour = "[DEBUG]"
	InfSignNoColour   = "[INFO]"
	WarnSignNoColour  = "[WARN]"
	ErrSignNoColour   = "[ERROR]"
	SucSignNoColour   = "[SUCCESS]"

	logLevel      string
	logsDirectory string

	appLogger   *lumberjack.Logger
	errorLogger *lumberjack.Logger
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

	appLogger = &lumberjack.Logger{
		Filename:   fmt.Sprintf("%s/app.log", logsDirectory),
		MaxSize:    50,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   true,
	}

	errorLogger = &lumberjack.Logger{
		Filename:   fmt.Sprintf("%s/error.log", logsDirectory),
		MaxSize:    50,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   true,
	}

	return nil
}

func CloseLogFiles() error {
	if err := appLogger.Close(); err != nil {
		return err
	}
	return errorLogger.Close()
}

func LogDebug(message string) {
	if logLevel != "release" {
		consoleLogger.Printf("%s %s\n", DebugSign, message)
	}
}

func LogInfo(message string) {
	consoleLogger.Printf("%s %s\n", InfSign, message)

	_, err := appLogger.Write(
		[]byte(fmt.Sprintf("%s - %s %s\n", time.Now().String(), InfSignNoColour, message)),
	)
	if err != nil {
		log.Println(err)
	}
}

func LogWarning(message string) {
	consoleLogger.Printf("%s %s\n", WarnSign, message)

	_, err := appLogger.Write(
		[]byte(fmt.Sprintf("%s - %s %s\n", time.Now().String(), WarnSignNoColour, message)),
	)
	if err != nil {
		log.Println(err)
	}
}

func LogError(message string) {
	consoleLogger.Printf("%s [non-critical] %s\n", ErrSign, message)

	_, err := errorLogger.Write(
		[]byte(fmt.Sprintf("%s - %s %s\n", time.Now().String(), ErrSignNoColour, message)),
	)
	if err != nil {
		log.Println(err)
	}
}

func LogFatal(message string) {
	consoleLogger.Printf("%s [critical] %s\n", ErrSign, message)

	_, err := errorLogger.Write(
		[]byte(fmt.Sprintf("%s - %s %s\n", time.Now().String(), ErrSignNoColour, message)),
	)
	if err != nil {
		log.Println(err)
	}

	os.Exit(1)
}

func LogSuccess(message string) {
	consoleLogger.Printf("%s %s\n", SucSign, message)

	_, err := appLogger.Write(
		[]byte(fmt.Sprintf("%s - %s %s\n", time.Now().String(), SucSignNoColour, message)),
	)
	if err != nil {
		log.Println(err)
	}
}
