package bootstrap

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/OpenListTeam/OpenList/v4/cmd/flags"
	"github.com/OpenListTeam/OpenList/v4/drivers/base"
	"github.com/OpenListTeam/OpenList/v4/internal/conf"
	"github.com/OpenListTeam/OpenList/v4/internal/net"
	"github.com/OpenListTeam/OpenList/v4/pkg/utils"
	"github.com/caarlos0/env/v9"
	log "github.com/sirupsen/logrus"
)

// Program working directory
func PWD() string {
	if flags.ForceBinDir {
		ex, err := os.Executable()
		if err != nil {
			log.Fatal(err)
		}
		pwd := filepath.Dir(ex)
		return pwd
	}
	d, err := os.Getwd()
	if err != nil {
		d = "."
	}
	return d
}

func InitConfig() {
	pwd := PWD()
	dataDir := flags.DataDir
	if !filepath.IsAbs(dataDir) {
		flags.DataDir = filepath.Join(pwd, flags.DataDir)
	}
	configPath := filepath.Join(flags.DataDir, "config.json")
	log.Infof("reading config file: %s", configPath)
	if !utils.Exists(configPath) {
		log.Infof("config file not exists, creating default config file")
		_, err := utils.CreateNestedFile(configPath)
		if err != nil {
			log.Fatalf("failed to create config file: %+v", err)
		}
		conf.Conf = conf.DefaultConfig(dataDir)
		LastLaunchedVersion = conf.Version
		conf.Conf.LastLaunchedVersion = conf.Version
		if !utils.WriteJsonToFile(configPath, conf.Conf) {
			log.Fatalf("failed to create default config file")
		}
	} else {
		configBytes, err := os.ReadFile(configPath)
		if err != nil {
			log.Fatalf("reading config file error: %+v", err)
		}
		conf.Conf = conf.DefaultConfig(dataDir)
		err = utils.Json.Unmarshal(configBytes, conf.Conf)
		if err != nil {
			log.Fatalf("load config error: %+v", err)
		}
		LastLaunchedVersion = conf.Conf.LastLaunchedVersion
		if strings.HasPrefix(conf.Version, "v") || LastLaunchedVersion == "" {
			conf.Conf.LastLaunchedVersion = conf.Version
		}
		// update config.json struct
		confBody, err := utils.Json.MarshalIndent(conf.Conf, "", "  ")
		if err != nil {
			log.Fatalf("marshal config error: %+v", err)
		}
		err = os.WriteFile(configPath, confBody, 0o777)
		if err != nil {
			log.Fatalf("update config struct error: %+v", err)
		}
	}
	if conf.Conf.MaxConcurrency > 0 {
		net.DefaultConcurrencyLimit = &net.ConcurrencyLimit{Limit: conf.Conf.MaxConcurrency}
	}
	if !conf.Conf.Force {
		confFromEnv()
	}
	// convert abs path
	convertAbsPath := func(path *string) {
		if !filepath.IsAbs(*path) {
			*path = filepath.Join(pwd, *path)
		}
	}
	convertAbsPath(&conf.Conf.TempDir)
	convertAbsPath(&conf.Conf.BleveDir)
	convertAbsPath(&conf.Conf.Log.Name)
	convertAbsPath(&conf.Conf.Database.DBFile)
	if conf.Conf.DistDir != "" {
		convertAbsPath(&conf.Conf.DistDir)
	}
	err := os.MkdirAll(conf.Conf.TempDir, 0o777)
	if err != nil {
		log.Fatalf("create temp dir error: %+v", err)
	}
	log.Debugf("config: %+v", conf.Conf)
	base.InitClient()
	initURL()
}

func confFromEnv() {
	prefix := "OPENLIST_"
	if flags.NoPrefix {
		prefix = ""
	}
	log.Infof("load config from env with prefix: %s", prefix)
	if err := env.ParseWithOptions(conf.Conf, env.Options{
		Prefix: prefix,
	}); err != nil {
		log.Fatalf("load config from env error: %+v", err)
	}
}

func initURL() {
	if !strings.Contains(conf.Conf.SiteURL, "://") {
		conf.Conf.SiteURL = utils.FixAndCleanPath(conf.Conf.SiteURL)
	}
	u, err := url.Parse(conf.Conf.SiteURL)
	if err != nil {
		utils.Log.Fatalf("can't parse site_url: %+v", err)
	}
	conf.URL = u
}

func CleanTempDir() {
	files, err := os.ReadDir(conf.Conf.TempDir)
	if err != nil {
		log.Errorln("failed list temp file: ", err)
	}
	for _, file := range files {
		if err := os.RemoveAll(filepath.Join(conf.Conf.TempDir, file.Name())); err != nil {
			log.Errorln("failed delete temp file: ", err)
		}
	}
}
