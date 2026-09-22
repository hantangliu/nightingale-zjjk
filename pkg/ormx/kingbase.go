package ormx

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/migrator"
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
	columnTypes := make([]gorm.ColumnType, 0)
	err := m.RunWithValue(value, func(stmt *gorm.Statement) error {
		rows, err := m.DB.Session(&gorm.Session{}).Table(stmt.Table).Limit(1).Rows()
		if err != nil {
			return err
		}
		defer rows.Close()

		rawColumnTypes, err := rows.ColumnTypes()
		if err != nil {
			return err
		}
		for _, columnType := range rawColumnTypes {
			columnTypes = append(columnTypes, &migrator.ColumnType{SQLColumnType: columnType})
		}
		return nil
	})
	return columnTypes, err
}
