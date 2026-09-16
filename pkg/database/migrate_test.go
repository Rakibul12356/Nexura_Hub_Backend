package database

import "testing"

func TestSplitSQLKeepsDollarBlocks(t *testing.T) {
	src := `
CREATE TABLE t (id int);
-- comment
DO $$
BEGIN
  ALTER TABLE t RENAME COLUMN a TO b;
END $$;
CREATE INDEX idx ON t(id);
`
	got := splitSQL(src)
	if len(got) != 3 {
		t.Fatalf("got %d statements, want 3: %#v", len(got), got)
	}
	if got[1][:2] != "DO" {
		t.Fatalf("second statement should be DO block, got %q", got[1])
	}
}
