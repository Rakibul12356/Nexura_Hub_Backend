package database

import (
	"database/sql"
	"log"
	"strings"
)

// RepairCompat upgrades leftover columns from the older Nexura schema so
// current handlers can scan UUIDs and text[] without 500s.
func RepairCompat(db *sql.DB) {
	if db == nil {
		return
	}
	repairCategoriesToUUID(db)
	repairLearningPointsArray(db)
}

func columnUDT(db *sql.DB, table, column string) string {
	var udt string
	err := db.QueryRow(`
		SELECT udt_name FROM information_schema.columns
		WHERE table_schema='public' AND table_name=$1 AND column_name=$2
	`, table, column).Scan(&udt)
	if err != nil {
		return ""
	}
	return strings.ToLower(udt)
}

func execLog(db *sql.DB, label, q string) {
	if _, err := db.Exec(q); err != nil {
		log.Printf("[WARNING] Compat %s: %v", label, err)
	}
}

func repairCategoriesToUUID(db *sql.DB) {
	idType := columnUDT(db, "categories", "id")
	if idType == "" || idType == "uuid" {
		return
	}
	log.Printf("[Database] Converting categories.id (%s) and courses.category_id to UUID...", idType)

	rows, err := db.Query(`
		SELECT rel.relname, con.conname
		FROM pg_constraint con
		JOIN pg_class rel ON rel.oid = con.conrelid
		JOIN pg_namespace nsp ON nsp.oid = rel.relnamespace
		WHERE nsp.nspname='public' AND con.contype='f'
		  AND (con.confrelid = 'public.categories'::regclass
		       OR (con.conrelid = 'public.courses'::regclass AND con.confrelid = 'public.categories'::regclass))
	`)
	if err == nil {
		var drops [][2]string
		for rows.Next() {
			var table, name string
			_ = rows.Scan(&table, &name)
			if table != "" && name != "" {
				drops = append(drops, [2]string{table, name})
			}
		}
		_ = rows.Close()
		for _, d := range drops {
			execLog(db, "drop fk "+d[1], `ALTER TABLE `+pqQuoteIdent(d[0])+` DROP CONSTRAINT IF EXISTS `+pqQuoteIdent(d[1]))
		}
	}

	execLog(db, "add categories.id_uuid", `ALTER TABLE categories ADD COLUMN IF NOT EXISTS id_uuid UUID`)
	execLog(db, "fill categories.id_uuid", `UPDATE categories SET id_uuid = uuid_generate_v4() WHERE id_uuid IS NULL`)
	execLog(db, "categories.id_uuid not null", `ALTER TABLE categories ALTER COLUMN id_uuid SET NOT NULL`)
	execLog(db, "add courses.category_uuid", `ALTER TABLE courses ADD COLUMN IF NOT EXISTS category_uuid UUID`)
	execLog(db, "map courses.category_uuid", `
		UPDATE courses c SET category_uuid = cat.id_uuid
		FROM categories cat
		WHERE c.category_id IS NOT NULL AND c.category_id = cat.id`)

	execLog(db, "drop categories pkey", `ALTER TABLE categories DROP CONSTRAINT IF EXISTS categories_pkey`)
	execLog(db, "drop categories.id", `ALTER TABLE categories DROP COLUMN id`)
	execLog(db, "rename categories.id_uuid", `ALTER TABLE categories RENAME COLUMN id_uuid TO id`)
	execLog(db, "categories pkey", `ALTER TABLE categories ADD PRIMARY KEY (id)`)

	catIDType := columnUDT(db, "courses", "category_id")
	if catIDType != "uuid" && catIDType != "" {
		execLog(db, "drop courses.category_id", `ALTER TABLE courses DROP COLUMN category_id`)
		execLog(db, "rename courses.category_uuid", `ALTER TABLE courses RENAME COLUMN category_uuid TO category_id`)
	}
	execLog(db, "courses.category_id fk", `ALTER TABLE courses ADD CONSTRAINT courses_category_id_fkey FOREIGN KEY (category_id) REFERENCES categories(id)`)
	log.Println("[Database] categories.id is now UUID.")
}

func repairLearningPointsArray(db *sql.DB) {
	udt := columnUDT(db, "courses", "learning_points")
	if udt == "" || udt == "_text" || udt == "text" {
		return
	}
	if udt != "jsonb" && udt != "json" {
		return
	}
	log.Println("[Database] Converting courses.learning_points jsonb → text[]...")
	execLog(db, "add learning_points_arr", `ALTER TABLE courses ADD COLUMN IF NOT EXISTS learning_points_arr TEXT[] DEFAULT '{}'`)
	execLog(db, "copy learning_points_arr", `
		UPDATE courses SET learning_points_arr = ARRAY(
			SELECT jsonb_array_elements_text(learning_points)
		)
		WHERE learning_points IS NOT NULL AND jsonb_typeof(learning_points) = 'array'`)
	execLog(db, "drop learning_points jsonb", `ALTER TABLE courses DROP COLUMN IF EXISTS learning_points`)
	execLog(db, "rename learning_points_arr", `ALTER TABLE courses RENAME COLUMN learning_points_arr TO learning_points`)
	execLog(db, "learning_points default", `ALTER TABLE courses ALTER COLUMN learning_points SET DEFAULT '{}'`)
}

func pqQuoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
