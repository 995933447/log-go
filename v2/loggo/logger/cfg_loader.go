package logger

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
)

type LogConf struct {
	File       FileLogConf
	AlertLevel string
}

type FileLogConf struct {
	FilePrefix                  string
	MaxFileSizeBytes            int64  // 文件最大容量,字节为单位
	DefaultLogDir               string // 默认日志目录
	ExceptionLogDir             string // 异常日志目录
	BillLogDir                  string // bill日志目录
	StatLogDir                  string // stat日志目录
	Level                       string // 日志最小级别
	DebugMsgMaxLen              int32  // debug日志消息最大长度,-1或者0代表不限制
	InfoMsgMaxLen               int32  // info日志消息最大长度,-1或者0代表不限制
	LogDebugBeforeFileSizeBytes int64  // 文件允许写入debug日志的大小阀值,-1代表不限制
	LogInfoBeforeFileSizeBytes  int64  // 文件允许写入info日志的大小阀值,-1代表不限制
	FileMaxRemainDays           int    // 文件最大保留天数
	MaxRemainFileNum            int    // 保留文件数量
	CompressFrequentHours       int    // 压缩频率小时数
	CompressAfterReachBytes     int64  // 压缩最小文件大小
}

func (f *FileLogConf) GetLevel() Level {
	return TransStrToLevel(f.Level)
}

const defaultReloadCfgFileIntervalSec = 10

func NewConfLoader(cfgFile string, reloadCfgFileIntervalSec uint32, defaultLogCfg *LogConf) (*ConfLoader, error) {
	var loader ConfLoader
	if reloadCfgFileIntervalSec <= 0 {
		reloadCfgFileIntervalSec = defaultReloadCfgFileIntervalSec
	}

	loader.reloadCfgFileIntervalSec = reloadCfgFileIntervalSec
	loader.cfgFile = cfgFile

	loader.opLogCfgMu.Lock()
	loader.cfg = defaultLogCfg
	loader.defaultLogCfg = defaultLogCfg
	loader.opLogCfgMu.Unlock()

	if err := loader.loadFile(); err != nil {
		return nil, err
	}

	loader.init()

	return &loader, nil
}

type ConfLoader struct {
	cfgFile                  string
	cfg                      *LogConf
	defaultLogCfg            *LogConf
	opLogCfgMu               sync.RWMutex
	reloadCfgFileIntervalSec uint32
}

func (c *ConfLoader) GetConf() *LogConf {
	c.opLogCfgMu.RLock()
	defer c.opLogCfgMu.RUnlock()
	return c.cfg
}

func (c *ConfLoader) init() {
	go func() {
		for {
			if err := c.loadFile(); err != nil {
				fmt.Println(err)
			}
			time.Sleep(time.Duration(c.reloadCfgFileIntervalSec) * time.Second)
		}
	}()
}

func (c *ConfLoader) loadFile() error {
	if c.cfgFile == "" {
		c.opLogCfgMu.Lock()
		defer c.opLogCfgMu.Unlock()
		c.cfg = c.defaultLogCfg
		return nil
	}

	if _, err := os.Stat(c.cfgFile); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		c.opLogCfgMu.Lock()
		defer c.opLogCfgMu.Unlock()
		c.cfg = c.defaultLogCfg
		return nil
	}

	var cfg LogConf
	if _, err := toml.DecodeFile(c.cfgFile, &cfg); err != nil {
		return err
	}
	c.opLogCfgMu.Lock()
	defer c.opLogCfgMu.Unlock()
	if cfg.File.Level == "" {
		cfg.File.Level = c.defaultLogCfg.File.Level
	}
	if cfg.File.DefaultLogDir == "" {
		cfg.File.DefaultLogDir = c.defaultLogCfg.File.DefaultLogDir
	}
	if cfg.File.BillLogDir == "" {
		cfg.File.BillLogDir = c.defaultLogCfg.File.BillLogDir
	}
	if cfg.File.StatLogDir == "" {
		cfg.File.StatLogDir = c.defaultLogCfg.File.StatLogDir
	}
	c.cfg = &cfg

	return nil
}

func (c *ConfLoader) SetDefaultLogConf(cfg *LogConf) {
	if cfg == nil {
		return
	}
	c.opLogCfgMu.Lock()
	defer c.opLogCfgMu.Unlock()
	if cfg.File.Level == "" {
		cfg.File.Level = c.defaultLogCfg.File.Level
	}
	if cfg.File.DefaultLogDir == "" {
		cfg.File.DefaultLogDir = c.defaultLogCfg.File.DefaultLogDir
	}
	if cfg.File.BillLogDir == "" {
		cfg.File.BillLogDir = c.defaultLogCfg.File.BillLogDir
	}
	if cfg.File.StatLogDir == "" {
		cfg.File.StatLogDir = c.defaultLogCfg.File.StatLogDir
	}
	if cfg.File.ExceptionLogDir == "" {
		cfg.File.ExceptionLogDir = c.defaultLogCfg.File.ExceptionLogDir
	}
	c.defaultLogCfg = cfg
}
