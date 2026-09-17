package repository

import (
	"errors"
	"testing"
)

func TestRebind(t *testing.T) {
	pg := &DB{Dialect: Postgres}
	if got := pg.Rebind(`SELECT ? , ? FROM t WHERE a = ?`); got != `SELECT $1 , $2 FROM t WHERE a = $3` {
		t.Fatalf("postgres rebind: %q", got)
	}
	my := &DB{Dialect: MySQL}
	if got := my.Rebind(`? ?`); got != `? ?` {
		t.Fatalf("mysql rebind dokunmamalı: %q", got)
	}
}

func TestMysqlDSN(t *testing.T) {
	cases := map[string]string{
		"u:p@127.0.0.1:3306/db":                   "u:p@tcp(127.0.0.1:3306)/db?parseTime=true&multiStatements=true",
		"u:p@127.0.0.1:3306/db?charset=utf8mb4":   "u:p@tcp(127.0.0.1:3306)/db?charset=utf8mb4&parseTime=true&multiStatements=true",
		"u:p@tcp(h:3306)/db?parseTime=true":       "u:p@tcp(h:3306)/db?parseTime=true&multiStatements=true",
		"u:p@unix(/var/lib/mysql/mysql.sock)/db":  "u:p@unix(/var/lib/mysql/mysql.sock)/db?parseTime=true&multiStatements=true",
	}
	for in, want := range cases {
		if got := mysqlDSN(in); got != want {
			t.Errorf("%s:\n  got  %s\n  want %s", in, got, want)
		}
	}
}

func TestIsUniqueViolation(t *testing.T) {
	for _, msg := range []string{
		`ERROR: duplicate key value violates unique constraint "uq_pkey" (SQLSTATE 23505)`,
		`Error 1062 (23000): Duplicate entry 'a' for key 'PRIMARY'`,
		`constraint failed: UNIQUE constraint failed: uq.k (1555)`,
	} {
		if !IsUniqueViolation(errors.New(msg)) {
			t.Errorf("tekil ihlal tanınmadı: %s", msg)
		}
	}
	if IsUniqueViolation(errors.New("connection refused")) || IsUniqueViolation(nil) {
		t.Error("yanlış pozitif")
	}
}
