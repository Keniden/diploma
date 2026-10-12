package directions_test

import (
	"testing"

	"github.com/google/uuid"

	"last-mile-task/internal/documents/domain/directions"
)

func pkg(id, from, to uuid.UUID, action directions.PackageAction) directions.Package {
	return directions.Package{ID: id, From: from, To: to, Action: action}
}

func TestGroupByDirection_ForwardAndReturn(t *testing.T) {
	src := uuid.New()
	dest := uuid.New()
	intermediate := uuid.New()

	pkgs := []directions.Package{
		pkg(uuid.New(), src, dest, directions.PackageActionLoad),
		pkg(uuid.New(), src, dest, directions.PackageActionLoad),
		pkg(uuid.New(), intermediate, src, directions.PackageActionLoad),
	}

	groups := directions.GroupByDirection(pkgs)

	if len(groups) != 2 {
		t.Fatalf("expected 2 direction groups, got %d", len(groups))
	}

	forward := groups[directions.DirectionKey{From: src, To: dest}]
	if len(forward) != 2 {
		t.Fatalf("expected 2 forward packages, got %d", len(forward))
	}
	if !directions.HasAction(forward, directions.PackageActionLoad) {
		t.Fatal("forward group must contain LOAD action")
	}

	returnGroup := groups[directions.DirectionKey{From: intermediate, To: src}]
	if len(returnGroup) != 1 {
		t.Fatalf("expected 1 return package, got %d", len(returnGroup))
	}
}

func TestGroupByDirection_Empty(t *testing.T) {
	groups := directions.GroupByDirection(nil)
	if len(groups) != 0 {
		t.Fatalf("expected no groups for empty input, got %d", len(groups))
	}
}

func TestHasActionAndIsEmpty(t *testing.T) {
	if directions.IsEmpty(nil) != true {
		t.Fatal("expected IsEmpty true for nil")
	}
	if directions.IsEmpty([]directions.Package{{ID: uuid.New()}}) {
		t.Fatal("expected IsEmpty false for non-empty")
	}

	if directions.HasAction(nil, directions.PackageActionLoad) {
		t.Fatal("expected HasAction false for nil")
	}
	if !directions.HasAction([]directions.Package{{Action: directions.PackageActionUnload}}, directions.PackageActionUnload) {
		t.Fatal("expected HasAction true matching action")
	}
}