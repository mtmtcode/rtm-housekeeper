package housekeeper

import (
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/mtmtcode/rtm-housekeeper/rtm"
)

const (
	archiveListName = "archive"
	nextListName    = "next"
)

// Housekeeper performs maintenance tasks on RTM.
type Housekeeper struct {
	client         *rtm.Client
	dryRun         bool
	somedayLists   []string
	staleDays      int
	inboxLists     []string
	inboxStaleDays int
	nextLimit      int
}

// New creates a new Housekeeper.
func New(client *rtm.Client, dryRun bool, somedayLists []string, staleDays int, inboxLists []string, inboxStaleDays int, nextLimit int) *Housekeeper {
	return &Housekeeper{
		client:         client,
		dryRun:         dryRun,
		somedayLists:   somedayLists,
		staleDays:      staleDays,
		inboxLists:     inboxLists,
		inboxStaleDays: inboxStaleDays,
		nextLimit:      nextLimit,
	}
}

// Run executes all housekeeper tasks.
func (h *Housekeeper) Run() error {
	timeline := ""
	if !h.dryRun {
		var err error
		timeline, err = h.client.CreateTimeline()
		if err != nil {
			return fmt.Errorf("create timeline: %w", err)
		}
	}

	for _, list := range h.inboxLists {
		if err := h.demoteInbox(timeline, list); err != nil {
			return fmt.Errorf("demote inbox (%s): %w", list, err)
		}
	}

	for _, list := range h.somedayLists {
		if err := h.archiveSomeday(timeline, list); err != nil {
			return fmt.Errorf("archive someday (%s): %w", list, err)
		}
	}

	if err := h.capNext(timeline); err != nil {
		return fmt.Errorf("cap next: %w", err)
	}

	if err := h.archiveTagged(timeline); err != nil {
		return fmt.Errorf("archive tagged: %w", err)
	}

	if err := h.deleteTrash(timeline); err != nil {
		return fmt.Errorf("delete trash: %w", err)
	}

	return nil
}

// archiveSomeday moves stale tasks in the given list to the archive list.
func (h *Housekeeper) archiveSomeday(timeline, listName string) error {
	cutoff := time.Now().AddDate(0, 0, -h.staleDays).Format("2006-01-02")
	filter := fmt.Sprintf(`list:"%s" AND status:incomplete AND updatedBefore:"%s"`, listName, cutoff)
	tasks, err := h.collectTasks(filter)
	if err != nil {
		return err
	}

	if len(tasks) == 0 {
		log.Printf("[someday:%s] no tasks to archive", listName)
		return nil
	}

	log.Printf("[someday:%s] found %d task(s) to archive", listName, len(tasks))
	for _, t := range tasks {
		log.Printf("[someday:%s]   - %s", listName, t.Name)
	}

	if h.dryRun {
		return nil
	}

	archiveListID, err := h.ensureArchiveList(timeline)
	if err != nil {
		return err
	}

	for _, t := range tasks {
		if err := h.client.MoveTo(timeline, t, archiveListID); err != nil {
			return fmt.Errorf("move task %q: %w", t.Name, err)
		}
		log.Printf("[someday:%s] archived: %s", listName, t.Name)
	}

	return nil
}

// archiveTagged moves tasks with the "archive" tag to the archive list.
func (h *Housekeeper) archiveTagged(timeline string) error {
	filter := `tag:archive AND status:incomplete`
	tasks, err := h.collectTasks(filter)
	if err != nil {
		return err
	}

	if len(tasks) == 0 {
		log.Println("[archive-tag] no tasks to archive")
		return nil
	}

	log.Printf("[archive-tag] found %d task(s) to archive", len(tasks))
	for _, t := range tasks {
		log.Printf("[archive-tag]   - %s", t.Name)
	}

	if h.dryRun {
		return nil
	}

	archiveListID, err := h.ensureArchiveList(timeline)
	if err != nil {
		return err
	}

	for _, t := range tasks {
		if err := h.client.MoveTo(timeline, t, archiveListID); err != nil {
			return fmt.Errorf("move task %q: %w", t.Name, err)
		}
		log.Printf("[archive-tag] archived: %s", t.Name)
	}

	return nil
}

// deleteTrash deletes tasks with the "trash" tag.
func (h *Housekeeper) deleteTrash(timeline string) error {
	filter := `tag:trash AND status:incomplete`
	tasks, err := h.collectTasks(filter)
	if err != nil {
		return err
	}

	if len(tasks) == 0 {
		log.Println("[trash] no tasks to delete")
		return nil
	}

	log.Printf("[trash] found %d task(s) to delete", len(tasks))
	for _, t := range tasks {
		log.Printf("[trash]   - %s", t.Name)
	}

	if h.dryRun {
		return nil
	}

	for _, t := range tasks {
		if err := h.client.DeleteTask(timeline, t); err != nil {
			return fmt.Errorf("delete task %q: %w", t.Name, err)
		}
		log.Printf("[trash] deleted: %s", t.Name)
	}

	return nil
}

// demoteInbox moves stale tasks in the given inbox list to the someday list.
func (h *Housekeeper) demoteInbox(timeline, listName string) error {
	cutoff := time.Now().AddDate(0, 0, -h.inboxStaleDays).Format("2006-01-02")
	filter := fmt.Sprintf(`list:"%s" AND status:incomplete AND updatedBefore:"%s"`, listName, cutoff)
	tasks, err := h.collectTasks(filter)
	if err != nil {
		return err
	}

	if len(tasks) == 0 {
		log.Printf("[inbox:%s] no tasks to demote", listName)
		return nil
	}

	log.Printf("[inbox:%s] found %d task(s) to demote to someday", listName, len(tasks))
	for _, t := range tasks {
		log.Printf("[inbox:%s]   - %s", listName, t.Name)
	}

	if h.dryRun {
		return nil
	}

	somedayListID, err := h.ensureList(timeline, h.somedayLists[0])
	if err != nil {
		return err
	}

	for _, t := range tasks {
		if err := h.client.MoveTo(timeline, t, somedayListID); err != nil {
			return fmt.Errorf("move task %q: %w", t.Name, err)
		}
		log.Printf("[inbox:%s] demoted to someday: %s", listName, t.Name)
	}

	return nil
}

// capNext enforces the task limit on the _next smart list.
func (h *Housekeeper) capNext(timeline string) error {
	filter := `list:"_next" AND status:incomplete`
	taskLists, err := h.client.GetTaskList(filter)
	if err != nil {
		return err
	}

	type nextTask struct {
		rtm.TaskInfo
		Added string
		Tags  []string
	}

	var tasks []nextTask
	for _, tl := range taskLists {
		for _, ts := range tl.TaskSeries {
			for _, t := range ts.Task {
				tasks = append(tasks, nextTask{
					TaskInfo: rtm.TaskInfo{
						ListID:       tl.ID,
						TaskSeriesID: ts.ID,
						TaskID:       t.ID,
						Name:         ts.Name,
					},
					Added: t.Added,
					Tags:  ts.Tags.Tag,
				})
			}
		}
	}

	if len(tasks) <= h.nextLimit {
		log.Printf("[next] %d task(s) in _next (limit %d), nothing to do", len(tasks), h.nextLimit)
		return nil
	}

	// Sort by added date ascending (oldest first).
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].Added < tasks[j].Added
	})

	excess := tasks[:len(tasks)-h.nextLimit]

	nextListID, err := h.findListID(nextListName)
	if err != nil {
		return err
	}

	log.Printf("[next] %d task(s) in _next, exceeds limit of %d, %d to push out", len(tasks), h.nextLimit, len(excess))
	for _, t := range excess {
		log.Printf("[next]   - %s (added: %s)", t.Name, t.Added)
	}

	if h.dryRun {
		return nil
	}

	somedayListID, err := h.ensureList(timeline, h.somedayLists[0])
	if err != nil {
		return err
	}

	for _, t := range excess {
		info := t.TaskInfo

		if info.ListID == nextListID {
			if err := h.client.MoveTo(timeline, info, somedayListID); err != nil {
				return fmt.Errorf("move task %q to someday: %w", t.Name, err)
			}
			log.Printf("[next] moved to someday: %s", t.Name)
			info.ListID = somedayListID
		}

		if hasTag(t.Tags, "na") {
			if err := h.client.RemoveTag(timeline, info, "na"); err != nil {
				return fmt.Errorf("remove na tag from %q: %w", t.Name, err)
			}
			log.Printf("[next] removed na tag: %s", t.Name)
		}
	}

	return nil
}

// hasTag reports whether the given tag exists in the slice.
func hasTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}

// collectTasks fetches tasks matching the filter and flattens them into TaskInfo slice.
func (h *Housekeeper) collectTasks(filter string, extra ...map[string]string) ([]rtm.TaskInfo, error) {
	taskLists, err := h.client.GetTaskList(filter, extra...)
	if err != nil {
		return nil, err
	}

	var results []rtm.TaskInfo
	for _, tl := range taskLists {
		for _, ts := range tl.TaskSeries {
			for _, t := range ts.Task {
				results = append(results, rtm.TaskInfo{
					ListID:       tl.ID,
					TaskSeriesID: ts.ID,
					TaskID:       t.ID,
					Name:         ts.Name,
				})
			}
		}
	}
	return results, nil
}

// ensureArchiveList finds or creates the archive list.
func (h *Housekeeper) ensureArchiveList(timeline string) (string, error) {
	lists, err := h.client.GetLists()
	if err != nil {
		return "", err
	}

	for _, l := range lists {
		if l.Name == archiveListName {
			return l.ID, nil
		}
	}

	log.Printf("[someday] creating %q list", archiveListName)
	resp, err := h.client.AddList(timeline, archiveListName)
	if err != nil {
		return "", fmt.Errorf("create archive list: %w", err)
	}
	return resp.ID, nil
}

// ensureList finds or creates a list with the given name.
func (h *Housekeeper) ensureList(timeline, name string) (string, error) {
	lists, err := h.client.GetLists()
	if err != nil {
		return "", err
	}

	for _, l := range lists {
		if l.Name == name {
			return l.ID, nil
		}
	}

	log.Printf("creating %q list", name)
	resp, err := h.client.AddList(timeline, name)
	if err != nil {
		return "", fmt.Errorf("create %q list: %w", name, err)
	}
	return resp.ID, nil
}

// findListID returns the ID of the list with the given name.
func (h *Housekeeper) findListID(name string) (string, error) {
	lists, err := h.client.GetLists()
	if err != nil {
		return "", err
	}

	for _, l := range lists {
		if l.Name == name {
			return l.ID, nil
		}
	}
	return "", fmt.Errorf("list %q not found", name)
}
