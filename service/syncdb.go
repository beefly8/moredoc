package service

import (
	"database/sql"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"moredoc/conf"
	"moredoc/model"

	"github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
)

func SyncDB(cfg *conf.Config, logger *zap.Logger) {
	err := checkAndCreateDatabase(cfg, logger)
	if err != nil {
		logger.Fatal("checkAndCreateDatabase", zap.Error(err))
		return
	}

	lg := logger.Named("syncdb")
	lg.Info("start syncdb")
	dbModel, err := model.NewDBModel(&cfg.Database, logger)
	if err != nil {
		lg.Fatal("NewDBModel", zap.Error(err))
		return
	}
	defer dbModel.CloseDB()

	err = dbModel.SyncDB()
	if err != nil {
		lg.Fatal("SyncDB", zap.Error(err))
		return
	}
	lg.Info("syncdb success")
}

func checkAndCreateDatabase(cfgI *conf.Config, loggger *zap.Logger) (err error) {
	if cfgI.Database.Driver == "mysql" {
		cfg, err := mysql.ParseDSN(cfgI.Database.DSN)
		if err != nil {
			loggger.Error("ParseDSN", zap.Error(err))
			return err
		}
		dbName := cfg.DBName
		if dbName == "" {
			loggger.Error("ParseDSN", zap.String("database", "数据库名称不能为空"))
			return err
		}
		conn := fmt.Sprintf("%s:%s@tcp(%s)/", cfg.User, cfg.Passwd, cfg.Addr)
		db, err := sql.Open("mysql", conn)
		if err != nil {
			loggger.Error("sql.Open", zap.Error(err))
			return err
		}

		createDB := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARSET utf8mb4 COLLATE utf8mb4_unicode_ci", cfg.DBName)
		_, err = db.Exec(createDB)
		if err != nil {
			loggger.Error("db.Exec", zap.Error(err))
		}
		return err
	} else if cfgI.Database.Driver == "postgresql" {

		var dbNameToCreate = "moredoc"

		db, err := gorm.Open(postgres.Open(cfgI.Database.DSN), &gorm.Config{
			NamingStrategy: schema.NamingStrategy{
				TablePrefix:   cfgI.Database.Prefix, // 表名前缀，`User`表为`t_users`
				SingularTable: true,                 // 使用单数表名，启用该选项后，`User` 表将是`user`
			},
			Logger: logger.Default,
		})
		checkDBExistQuery := fmt.Sprintf("SELECT 1 FROM pg_database WHERE datname='%s'", dbNameToCreate)

		// 执行检查数据库是否存在的操作
		var exists bool
		db.Exec(checkDBExistQuery).Scan(&exists)

		if exists {
			fmt.Printf("Database %s already exists!\n", dbNameToCreate)
		} else {
			// 创建新数据库 SQL 语句
			createDBQuery := fmt.Sprintf("CREATE DATABASE %s WITH ENCODING 'UTF8' TEMPLATE template0 LC_COLLATE 'C' LC_CTYPE 'C';", dbNameToCreate)

			// 执行创建数据库操作
			_ = db.Exec(createDBQuery)

			fmt.Printf("Database %s created successfully!\n", dbNameToCreate)
		}

		if err != nil {
			loggger.Error("db.Exec", zap.Error(err))
			return err

		}
	}
	return err
}
