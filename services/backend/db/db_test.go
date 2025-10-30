package db_test

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/thakurnishu/MinimalDo/db"
)

// #############################################################################
// # Unit Test
// #############################################################################

// TestInitDB unit tests the private initDB function using sqlmock.
func TestInitDB(t *testing.T) {
	t.Run("success - initializes database schema", func(t *testing.T) {
		dbsql, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer dbsql.Close()

		// We expect a single Exec call.
		// We use a regex to match the query since it's a large multi-line string.
		// This regex checks for "CREATE TABLE IF NOT EXISTS todos"
		// and allows for leading whitespace/newlines.
		mock.ExpectExec("(?s)CREATE TABLE IF NOT EXISTS todos").
			WillReturnResult(sqlmock.NewResult(0, 0)) // Success, no rows affected

		err = db.InitDB(dbsql)
		assert.NoError(t, err)

		// Ensure all mock expectations were met
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - database exec fails", func(t *testing.T) {
		dbsql, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer dbsql.Close()

		expectedErr := errors.New("db error")

		// Expect the query and return an error
		mock.ExpectExec("(?s)CREATE TABLE IF NOT EXISTS todos").
			WillReturnError(expectedErr)

		err = db.InitDB(dbsql)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)

		// Ensure all mock expectations were met
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
