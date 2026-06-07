package database

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/admin8800/s-ui/config"
	"github.com/admin8800/s-ui/database/model"
	"github.com/gofrs/uuid/v5"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db *gorm.DB

func initUser() error {
	var count int64
	err := db.Model(&model.User{}).Count(&count).Error
	if err != nil {
		return err
	}
	if count == 0 {
		user := &model.User{
			Username: "admin",
			Password: "admin",
		}
		return db.Create(user).Error
	}
	return nil
}

func OpenDB(dbPath string) error {
	dir := path.Dir(dbPath)
	err := os.MkdirAll(dir, 01740)
	if err != nil {
		return err
	}

	var gormLogger logger.Interface

	if config.IsDebug() {
		gormLogger = logger.Default
	} else {
		gormLogger = logger.Discard
	}

	c := &gorm.Config{
		Logger: gormLogger,
	}
	sep := "?"
	if strings.Contains(dbPath, "?") {
		sep = "&"
	}
	dsn := dbPath + sep + "_busy_timeout=10000&_journal_mode=WAL"
	db, err = gorm.Open(sqlite.Open(dsn), c)
	if err != nil {
		return err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if config.IsDebug() {
		db = db.Debug()
	}
	return nil
}

func InitDB(dbPath string) error {
	err := OpenDB(dbPath)
	if err != nil {
		return err
	}

	// Default Outbounds
	if !db.Migrator().HasTable(&model.Outbound{}) {
		db.Migrator().CreateTable(&model.Outbound{})
		defaultOutbound := []model.Outbound{
			{Type: "direct", Tag: "direct", Options: json.RawMessage(`{}`)},
		}
		db.Create(&defaultOutbound)
	}

	err = db.AutoMigrate(
		&model.Setting{},
		&model.Tls{},
		&model.Inbound{},
		&model.Outbound{},
		&model.Service{},
		&model.Endpoint{},
		&model.User{},
		&model.Tokens{},
		&model.Stats{},
		&model.Client{},
		&model.Changes{},
		&model.Node{},
	)
	if err != nil {
		return err
	}
	// Initialize default local master node (ID=0)
	var nodeCount int64
	db.Model(&model.Node{}).Count(&nodeCount)
	if nodeCount == 0 {
		db.Exec("INSERT OR IGNORE INTO nodes (id, name, token, address, last_heartbeat, online, sync_status) VALUES (0, '本地主控', 'master-token-placeholder', '127.0.0.1', ?, 1, 'synchronized')", time.Now().Unix())
	}
	err = initUser()
	if err != nil {
		return err
	}

	// 默认 Reality 证书注入逻辑 (如果 Tls 表为空)
	var tlsCount int64
	db.Model(&model.Tls{}).Count(&tlsCount)
	if tlsCount == 0 {
		privateKey, err := wgtypes.GeneratePrivateKey()
		if err == nil {
			publicKey := privateKey.PublicKey()
			privKeyStr := base64.RawURLEncoding.EncodeToString(privateKey[:])
			pubKeyStr := base64.RawURLEncoding.EncodeToString(publicKey[:])

			shortId := "0123456789abcdef"
			tempUuid, err := uuid.NewV4()
			if err == nil {
				shortId = strings.ReplaceAll(tempUuid.String(), "-", "")[:16]
			}

			serverConfig := fmt.Sprintf(`{"enabled":true,"server_name":"yahoo.com","reality":{"enabled":true,"handshake":{"server_port":443},"private_key":"%s","short_id":["%s"]}}`, privKeyStr, shortId)
			clientConfig := fmt.Sprintf(`{"reality":{"public_key":"%s"},"utls":{"fingerprint":"chrome"}}`, pubKeyStr)

			defaultTls := model.Tls{
				Id:     1,
				Name:   "默认 Reality 证书",
				Server: json.RawMessage(serverConfig),
				Client: json.RawMessage(clientConfig),
			}
			err = db.Create(&defaultTls).Error
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func GetDB() *gorm.DB {
	return db
}

func IsNotFound(err error) bool {
	return err == gorm.ErrRecordNotFound
}

func BackupDB(dbPath string) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	bakPath := dbPath + ".bak"
	data, err := os.ReadFile(dbPath)
	if err != nil {
		return err
	}
	return os.WriteFile(bakPath, data, 0600)
}

func RollbackToBackup(dbPath string) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.Close()
	}

	bakPath := dbPath + ".bak"
	if _, err := os.Stat(bakPath); os.IsNotExist(err) {
		return OpenDB(dbPath)
	}

	data, err := os.ReadFile(bakPath)
	if err != nil {
		OpenDB(dbPath)
		return err
	}

	err = os.WriteFile(dbPath, data, 0600)
	if err != nil {
		OpenDB(dbPath)
		return err
	}

	return OpenDB(dbPath)
}
