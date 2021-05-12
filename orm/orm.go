package orm

import (
	"time"

	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// Config database
type Config struct {
	DSN         string        // database adress
	Active      int           // pool
	Idle        int           // pool
	IdleTimeout time.Duration // conn max life time
}

// NewMySQL returns a new MySQL connection
func NewMySQL(c *Config) (db *gorm.DB) {
	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN:                       c.DSN, // DSN data source name
		DefaultStringSize:         256,   // string 类型字段的默认长度
		DisableDatetimePrecision:  true,  // 禁用 datetime 精度，MySQL 5.6 之前的数据库不支持
		DontSupportRenameIndex:    true,  // 重命名索引时采用删除并新建的方式，MySQL 5.7 之前的数据库和 MariaDB 不支持重命名索引
		DontSupportRenameColumn:   true,  // 用 `change` 重命名列，MySQL 8 之前的数据库和 MariaDB 不支持重命名列
		SkipInitializeWithVersion: false, // 根据当前 MySQL 版本自动配置
	}), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
		PrepareStmt:    true,
	})
	if err != nil {
		log.Printf("open database error:(%v)", err)
		panic(err)
	}
	db.Use(&OpentracingPlugin{})
	conn, _ := db.DB()
	conn.SetMaxIdleConns(c.Idle)
	conn.SetMaxOpenConns(c.Active)
	// conn.SetConnMaxLifetime(time.Duration(c.IdleTimeout) / time.Second)
	return
}
