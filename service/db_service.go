package service

import (
	"github.com/go-pg/pg/v9"
)

const (
	pgAddr     = "localhost:5432"
	pgUser     = "postgres"
	pgPassword = "postgres"
	pgDbName   = "chatserverdb"
)

type DBService struct {
	DB *pg.DB
}

func (service *DBService) Connect() error {
	service.DB = pg.Connect(&pg.Options{
		Addr:     pgAddr,
		User:     pgUser,
		Password: pgPassword,
		Database: pgDbName,
	})
	return nil
}

func (service *DBService) CloseConnection() {
	_ = service.DB.Close()
}
