package course

import (
	"strings"
	"testing"
)

func TestLearningPointsExprDoesNotCastArrayToJSONB(t *testing.T) {
	if strings.Contains(learningPointsExpr, "::jsonb") {
		t.Fatal("learning_points is text[]; ::jsonb 500s with SQLSTATE 42846")
	}
	if !strings.Contains(learningPointsExpr, "learning_points") {
		t.Fatal("expected learning_points column in select expression")
	}
}
