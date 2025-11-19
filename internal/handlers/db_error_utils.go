package handlers

import (
	"errors"

	"github.com/go-sql-driver/mysql"
)

// isForeignKeyConstraintError checks if the error corresponds to a MySQL/MariaDB
// foreign key constraint failure. This helps translate DB failures into clear
// client-facing validation responses instead of generic 500 errors.
func isForeignKeyConstraintError(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1452
}
