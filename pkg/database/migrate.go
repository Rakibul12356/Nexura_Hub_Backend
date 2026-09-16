package database

import (
	"database/sql"
	"log"
	"os"
	"strings"
	"unicode"
)

// ApplySchema runs the init migration statement-by-statement so one
// compatibility error (e.g. missing column on an older database) does
// not abort later CREATE TABLE statements.
func ApplySchema(db *sql.DB, path string) {
	sqlBytes, err := os.ReadFile(path)
	if err != nil {
		log.Printf("[WARNING] Auto-migration: could not read %s: %v", path, err)
		return
	}
	stmts := splitSQL(string(sqlBytes))
	var failed int
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			failed++
			log.Printf("[WARNING] Auto-migration: %v", err)
		}
	}
	if failed == 0 {
		log.Println("[Database] Schema migration applied.")
	} else {
		log.Printf("[Database] Schema migration finished with %d warning(s).", failed)
	}
	RepairCompat(db)
}

func splitSQL(src string) []string {
	var stmts []string
	var b strings.Builder
	inDollar := false
	dollarTag := ""
	i := 0
	for i < len(src) {
		if !inDollar && i+1 < len(src) && src[i] == '-' && src[i+1] == '-' {
			for i < len(src) && src[i] != '\n' {
				i++
			}
			continue
		}
		if src[i] == '$' {
			j := i + 1
			for j < len(src) && isDollarTagChar(src[j]) {
				j++
			}
			if j < len(src) && src[j] == '$' {
				tag := src[i : j+1]
				if !inDollar {
					inDollar = true
					dollarTag = tag
				} else if tag == dollarTag {
					inDollar = false
					dollarTag = ""
				}
				b.WriteString(tag)
				i = j + 1
				continue
			}
		}
		if !inDollar && src[i] == ';' {
			if s := strings.TrimSpace(b.String()); s != "" {
				stmts = append(stmts, s)
			}
			b.Reset()
			i++
			continue
		}
		b.WriteByte(src[i])
		i++
	}
	if s := strings.TrimSpace(b.String()); s != "" {
		stmts = append(stmts, s)
	}
	return stmts
}

func isDollarTagChar(r byte) bool {
	return r == '_' || unicode.IsLetter(rune(r)) || unicode.IsDigit(rune(r))
}
