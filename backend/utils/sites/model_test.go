package sites

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestAuctionPrint(t *testing.T) {
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldStdout := os.Stdout
	os.Stdout = write
	t.Cleanup(func() { os.Stdout = oldStdout })

	(&Auction{
		Date:    "Jan 2, 2026",
		Time:    "10:00 AM",
		Street:  "12 Main St",
		City:    "Boston",
		Deposit: "$5,000",
		Status:  "Active",
	}).Print()

	if err := write.Close(); err != nil {
		t.Fatal(err)
	}
	output, err := io.ReadAll(read)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"10:00 AM", "12 Main St", "Boston", "$5,000", "Active"} {
		if !strings.Contains(string(output), want) {
			t.Errorf("Print() output %q does not contain %q", output, want)
		}
	}
}
