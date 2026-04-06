package main

import (
	"fmt"
	"log"
	"os"

	"github.com/mtmtcode/rtm-housekeeper/rtm"
)

func main() {
	client := rtm.NewClient(
		os.Getenv("RTM_API_KEY"),
		os.Getenv("RTM_SHARED_SECRET"),
		os.Getenv("RTM_AUTH_TOKEN"),
	)

	lists, err := client.GetLists()
	if err != nil {
		log.Fatal(err)
	}

	var archiveListID string
	for _, l := range lists {
		if l.Name == "archive" && l.Archived == "1" {
			archiveListID = l.ID
			break
		}
	}

	if archiveListID == "" {
		fmt.Println("archived list named 'archive' not found")
		os.Exit(0)
	}

	timeline, err := client.CreateTimeline()
	if err != nil {
		log.Fatal(err)
	}

	if err := client.UnarchiveList(timeline, archiveListID); err != nil {
		log.Fatal(err)
	}

	fmt.Println("unarchived 'archive' list successfully")
}
