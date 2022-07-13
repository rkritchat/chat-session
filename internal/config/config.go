package config

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/caarlos0/env/v6"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"time"
)

type Cfg struct {
	DB  *sql.DB
	Env Env
}

type Env struct {
	Port                string `env:"PORT"`
	MySqlUser           string `env:"MYSQL_USER"`
	MySqlPwd            string `env:"MYSQL_PWD"`
	MySqlUrl            string `env:"MYSQL_URL"`
	MysqlDbName         string `env:"MYSQL_DB_NAME"`
	MySqlMaxOpenCon     int    `env:"MYSQL_MAX_OPEN_CON"`
	MySqlMaxIdleCon     int    `env:"MYSQL_MAX_IDLE_CON"`
	MySqlConMaxLifetime int    `env:"MYSQL_CON_MAX_LIFETIME"`
}

func InitConfig() Cfg {
	localEnv := initEnv()
	initDB(localEnv)
	initLogs()
	return Cfg{
		DB:  initDBCon(localEnv),
		Env: localEnv,
	}
}

func initEnv() Env {
	//read .env file
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	//parse environment variable to struct
	var localEnv Env
	err = env.Parse(&localEnv)
	if err != nil {
		panic(err)
	}

	return localEnv
}

func initDB(env Env) {
	url := fmt.Sprintf("%v:%v@tcp(%v)/", env.MySqlUser, env.MySqlPwd, env.MySqlUrl)
	conn, err := sql.Open("mysql", url)
	if err != nil {
		panic(err)
	}
	defer func() {
		err = conn.Close()
		fmt.Println(err)
	}()

	//health check
	err = conn.Ping()
	if err != nil {
		panic(err)
	}

	query := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %v CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", env.MysqlDbName)
	_, err = conn.Exec(query)
	if err != nil {
		panic(err)
	}
}

func initDBCon(env Env) *sql.DB {
	url := fmt.Sprintf("%v:%v@tcp(%v)/%v?parseTime=true&loc=Local", env.MySqlUser, env.MySqlPwd, env.MySqlUrl, env.MysqlDbName)
	fmt.Printf("start connect: %v\n", url)
	conn, err := sql.Open("mysql", url)
	if err != nil {
		panic(err)
	}
	conn.SetMaxOpenConns(env.MySqlMaxOpenCon)
	conn.SetMaxIdleConns(env.MySqlMaxIdleCon)
	conn.SetConnMaxLifetime(time.Duration(env.MySqlConMaxLifetime) * time.Millisecond)

	//check db health
	err = conn.Ping()
	if err != nil {
		panic(err)
	}

	return conn
}

func initLogs() {
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoder := zapcore.NewJSONEncoder(config)

	//create directory if not exist
	_, err := os.Stat("./logs")
	if err != nil {
		err = os.Mkdir("./logs", os.ModePerm)
		if err != nil {
			panic(err)
		}
	}

	//create logs file
	logFile, err := os.OpenFile("./logs/app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}

	context.Background()
	core := zapcore.NewTee(
		zapcore.NewCore(fileEncoder, zapcore.AddSync(logFile), zapcore.DebugLevel),
		zapcore.NewCore(fileEncoder, os.Stdout, zapcore.DebugLevel),
	)
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	zap.ReplaceGlobals(logger)
}

func (cfg *Cfg) Free() {
	if cfg.DB != nil {
		_ = cfg.DB.Close()
	}
}
