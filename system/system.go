package system

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/nightlyone/lockfile"

	"github.com/devusSs/steamquery-v2/config"
	"github.com/devusSs/steamquery-v2/logging"
)

var Clear map[string]func()

func InitClearFunc() {
	Clear = make(map[string]func())
	Clear["linux"] = func() {
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		err := cmd.Run()
		if err != nil {
			return
		}
	}
	Clear["darwin"] = func() {
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		err := cmd.Run()
		if err != nil {
			return
		}
	}
	Clear["windows"] = func() {
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		err := cmd.Run()
		if err != nil {
			return
		}
	}
}

func CheckAlreadyRunning(watchdog bool) (bool, error) {
	// Ignore screen sessions.
	if os.Getenv("STY") != "" {
		return false, nil
	}

	executable, err := os.Executable()
	if err != nil {
		return false, err
	}

	lockFile := filepath.Join(os.TempDir(), filepath.Base(executable)+".lock")

	lock, err := lockfile.New(lockFile)
	if err != nil {
		return false, err
	}

	if err := lock.TryLock(); err != nil {
		return true, nil
	}
	defer func() {
		if err := lock.Unlock(); err != nil {
			log.Fatal("Error unlocking lock file:", err)
		}
	}()

	return false, nil
}

func GetUserAgentHeaderFromOS() string {
	switch strings.ToLower(runtime.GOOS) {
	case "windows":
		return "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36"
	case "darwin":
		return "Mozilla/5.0 (Macintosh; Intel Mac OS X 13_4_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36"
	case "linux":
		return "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36"
	default:
		return "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36"
	}
}

func ListenForCTRLC() {
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	<-done
	fmt.Println("")
}

func CheckForGCloudConfigFile(path string) error {
	f, err := os.Open(path)
	f.Close()
	return err
}

func RunAnalysisMode(logsDir, cfg, gCloud string) error {
	if err := readAndCheckErrorFile(logsDir); err != nil {
		if strings.Contains(err.Error(), "no such file or directory") {
			fmt.Printf("%s No error.log file so far\n", logging.SucSign)
		} else {
			return err
		}
	}

	if err := testDNS(); err != nil {
		return err
	}

	if err := getSupportedOS(); err != nil {
		return err
	}

	if err := checkConfigFileExist(cfg); err != nil {
		return err
	}

	if err := loadAndCheckConfig(cfg); err != nil {
		return err
	}

	if err := checkGCoudConfigFileExist(gCloud); err != nil {
		return err
	}

	fmt.Printf("%s No errors occured so far\n", logging.InfSign)
	fmt.Printf("%s This might indicate a problem outside of this enviroment\n", logging.InfSign)
	fmt.Printf(
		"%s Please make sure:\n\ta) Your GCloud config is not malformed\n\tb) Your internet connection is working properly\n\tc) You are running the latest version of this app\n\td) Your Google sheet is formatted properly (check README.md)\n",
		logging.InfSign,
	)

	return nil
}

func readAndCheckErrorFile(logsDir string) error {
	fmt.Printf("%s Reading error file\n", logging.InfSign)

	f, err := os.Open(fmt.Sprintf("%s/error.log", logsDir))
	if err != nil {
		return err
	}
	defer f.Close()

	errorLines := 0

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()

		if line != "" {
			fmt.Printf(
				"%s Found following error: %s\n",
				logging.ErrSign,
				strings.Split(line, "-")[1],
			)
			errorLines++
		}
	}

	if errorLines == 0 {
		fmt.Printf("%s No errors found in %s\n", logging.SucSign, f.Name())
	}

	return scanner.Err()
}

func testDNS() error {
	fmt.Printf(
		"%s Testing DNS lookup for steamcommunity.com and docs.google.com\n",
		logging.InfSign,
	)

	ips, err := net.LookupHost("steamcommunity.com")
	if err != nil {
		return err
	}

	if len(ips) == 0 {
		return errors.New("no ip address found for steamcommunity.com")
	}

	ips, err = net.LookupHost("docs.google.com")
	if err != nil {
		return err
	}

	if len(ips) == 0 {
		return errors.New("no ip address found for docs.google.com")
	}

	fmt.Printf("%s DNS lookup works\n", logging.SucSign)

	return nil
}

func getSupportedOS() error {
	fmt.Printf("%s Getting supported OS list\n", logging.InfSign)

	currentHost := runtime.GOOS
	allowedHost := []string{"windows", "darwin", "linux"}

	foundOS := false

	for _, allowed := range allowedHost {
		if strings.ToLower(currentHost) == allowed {
			foundOS = true
			break
		}
	}

	if !foundOS {
		return fmt.Errorf("unsupported OS: %s (want: %v)", runtime.GOOS, allowedHost)
	}

	fmt.Printf("%s Current OS (%s) is supported\n", logging.SucSign, runtime.GOOS)

	return nil
}

func checkConfigFileExist(configFile string) error {
	fmt.Printf("%s Checking if specified config file (%s) exists\n", logging.InfSign, configFile)

	f, err := os.Open(configFile)
	if err != nil {
		return err
	}

	if err := f.Close(); err != nil {
		return err
	}

	fmt.Printf("%s Config file exists\n", logging.SucSign)

	return nil
}

func checkGCoudConfigFileExist(gCloud string) error {
	fmt.Printf("%s Checking if specified gcloud config file (%s) exists\n", logging.InfSign, gCloud)

	f, err := os.Open(gCloud)
	if err != nil {
		return err
	}

	if err := f.Close(); err != nil {
		return err
	}

	fmt.Printf("%s Gcloud config file exists\n", logging.SucSign)

	return nil
}

func loadAndCheckConfig(cfg string) error {
	fmt.Printf("%s Attempting to load config\n", logging.InfSign)

	c, err := config.LoadConfig(cfg)
	if err != nil {
		return err
	}

	fmt.Printf("%s Successfully loaded config\n", logging.SucSign)

	fmt.Printf("%s Checking config\n", logging.InfSign)

	if err := c.CheckConfig(false); err != nil {
		return err
	}

	fmt.Printf("%s Successfully checked config\n", logging.SucSign)

	return nil
}
