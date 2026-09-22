package ormx

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type kingbaseDialector struct {
	*postgres.Dialector
}

type kingbaseMigrator struct {
	postgres.Migrator
}

func newKingbaseDialector(dsn string) gorm.Dialector {
	return &kingbaseDialector{
		Dialector: postgres.Open(dsn).(*postgres.Dialector),
	}
}

func (d kingbaseDialector) Migrator(db *gorm.DB) gorm.Migrator {
	return kingbaseMigrator{
		Migrator: d.Dialector.Migrator(db).(postgres.Migrator),
	}
}

func (m kingbaseMigrator) ColumnTypes(value interface{}) ([]gorm.ColumnType, error) {
	return m.Migrator.Migrator.ColumnTypes(value)
}
