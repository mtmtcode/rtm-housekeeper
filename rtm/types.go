package rtm

import "encoding/json"

// Response is the top-level RTM API response.
type Response struct {
	Rsp Rsp `json:"rsp"`
}

type Rsp struct {
	Stat      string     `json:"stat"`
	Err       *RspErr    `json:"err,omitempty"`
	Frob      string     `json:"frob,omitempty"`
	Auth      *Auth      `json:"auth,omitempty"`
	Timeline  string     `json:"timeline,omitempty"`
	Tasks     *Tasks     `json:"tasks,omitempty"`
	Lists     *Lists     `json:"lists,omitempty"`
	List      *ListResp  `json:"list,omitempty"`
}

type RspErr struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
}

type Auth struct {
	Token string `json:"token"`
	Perms string `json:"perms"`
	User  struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Fullname string `json:"fullname"`
	} `json:"user"`
}

type Tasks struct {
	List []TaskList `json:"list"`
}

type TaskList struct {
	ID         string       `json:"id"`
	TaskSeries []TaskSeries `json:"taskseries"`
}

type TaskSeries struct {
	ID       string `json:"id"`
	Created  string `json:"created"`
	Modified string `json:"modified"`
	Name     string `json:"name"`
	Tags     Tags   `json:"tags"`
	Task     []Task `json:"task"`
}

type Tags struct {
	Tag []string `json:"tag"`
}

// UnmarshalJSON handles both [] (empty array) and {"tag": [...]} (object) forms.
func (t *Tags) UnmarshalJSON(data []byte) error {
	// Try as object first.
	type tagsObj struct {
		Tag []string `json:"tag"`
	}
	var obj tagsObj
	if err := json.Unmarshal(data, &obj); err == nil {
		t.Tag = obj.Tag
		return nil
	}
	// Fall back to empty array.
	var arr []any
	if err := json.Unmarshal(data, &arr); err == nil {
		t.Tag = nil
		return nil
	}
	return json.Unmarshal(data, t)
}

type Task struct {
	ID        string `json:"id"`
	Due       string `json:"due"`
	Added     string `json:"added"`
	Completed string `json:"completed"`
	Deleted   string `json:"deleted"`
}

type Lists struct {
	List []List `json:"list"`
}

type List struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Deleted  string `json:"deleted"`
	Locked   string `json:"locked"`
	Archived string `json:"archived"`
	Smart    string `json:"smart"`
}

type ListResp struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// TaskInfo holds the IDs needed to operate on a specific task.
type TaskInfo struct {
	ListID       string
	TaskSeriesID string
	TaskID       string
	Name         string
}
