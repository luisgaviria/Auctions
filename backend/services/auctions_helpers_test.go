package services

import (
	"testing"

	"backendAuction/models"
)

func TestFormattingHelpers(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "large number", got: formatWithCommas(1234567), want: "1,234,567"},
		{name: "small number", got: formatWithCommas(42), want: "42"},
		{name: "slug display", got: slugToDisplay("new-bedford-city"), want: "New Bedford City"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestBuildChecklistPersonalizesAvailableData(t *testing.T) {
	items := buildChecklist(models.AuctionJSON{Deposit: "$5,000", RegistryURL: "https://registry.example"})
	if len(items) != 13 {
		t.Fatalf("buildChecklist() returned %d items, want 13", len(items))
	}
	if items[3].Task != "Prepare certified check for deposit ($5,000)" {
		t.Errorf("deposit checklist item = %q", items[3].Task)
	}
	if items[7].Task != "Verify all existing liens at the Registry of Deeds (https://registry.example)" {
		t.Errorf("registry checklist item = %q", items[7].Task)
	}

	withoutData := buildChecklist(models.AuctionJSON{})
	if withoutData[3].Task != "Prepare certified check for deposit" || withoutData[7].Task != "Verify all existing liens at the Registry of Deeds" {
		t.Errorf("empty auction data should use generic checklist items: %#v", withoutData)
	}
}

func TestTitleCaseStr(t *testing.T) {
	if got := titleCaseStr("  NEW BEDFORD  "); got != "New Bedford" {
		t.Errorf("titleCaseStr() = %q, want %q", got, "New Bedford")
	}
}
