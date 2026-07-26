// Package database
package database

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

type Database struct {
	*gorm.DB
}

func NewDatabase(db *gorm.DB) *Database {
	return &Database{DB: db}
}

func (d *Database) beginTxn(ctx context.Context) (*Database, error) {
	txn := d.DB.WithContext(ctx).Begin()
	if txn.Error != nil {
		return nil, fmt.Errorf("begin transaction failed: %w", txn.Error)
	}
	return &Database{DB: txn}, nil
}

func (d *Database) commitTxn() error {
	txn := d.DB.Commit()
	if txn.Error != nil {
		return fmt.Errorf("failed to commit transaction: %w", txn.Error)
	}
	return nil
}

func (d *Database) rollBackTxn() error {
	if err := d.DB.Rollback().Error; err != nil {
		return fmt.Errorf("rollback transaction: %w", err)
	}
	return nil
}

func (d *Database) WithTxn(ctx context.Context, fn func(tx *Database) error) error {
	tx, err := d.beginTxn(ctx)
	if err != nil {
		return err
	}
	err = fn(tx)
	if err != nil {
		if rbErr := tx.rollBackTxn(); rbErr != nil {
			return fmt.Errorf("%w (rollback error: %v)", err, rbErr)
		}
		return err
	}
	return tx.commitTxn()
}
