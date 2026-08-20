package store

import (
	"sync"
	"time"

	"go02-neighbor-help/internal/model"
)

type IDGenerator func(prefix string) string

// MemoryStore 基于 map + RWMutex 的内存实现。写成功后在释放锁后调用 persistHook。
type MemoryStore struct {
	mu sync.RWMutex

	users     map[string]model.User
	username  map[string]string
	posts     map[string]model.HelpPost
	apps      map[string]model.Application
	appIdx    map[string]string // postID+"|"+applicantID -> appID
	tasks     map[string]model.Task
	taskByPost map[string]string
	reviews   map[string]model.Review
	reviewIdx map[string]string // taskID+"|"+fromUserID -> reviewID
	messages  map[string]model.Message
	favorites map[string]model.Favorite
	favIdx    map[model.FavoriteKey]string
	notifs    map[string]model.Notification
	reports   map[string]model.Report
	credits   map[string]model.CreditLog

	now         func() time.Time
	genID       IDGenerator
	persistHook func()
}

func NewMemoryStore(now func() time.Time, genID IDGenerator) *MemoryStore {
	if now == nil {
		now = time.Now
	}
	if genID == nil {
		genID = defaultIDGenerator
	}
	return &MemoryStore{
		users:      make(map[string]model.User),
		username:   make(map[string]string),
		posts:      make(map[string]model.HelpPost),
		apps:       make(map[string]model.Application),
		appIdx:     make(map[string]string),
		tasks:      make(map[string]model.Task),
		taskByPost: make(map[string]string),
		reviews:    make(map[string]model.Review),
		reviewIdx:  make(map[string]string),
		messages:   make(map[string]model.Message),
		favorites:  make(map[string]model.Favorite),
		favIdx:     make(map[model.FavoriteKey]string),
		notifs:     make(map[string]model.Notification),
		reports:    make(map[string]model.Report),
		credits:    make(map[string]model.CreditLog),
		now:        now,
		genID:      genID,
	}
}

func (s *MemoryStore) SetPersistHook(hook func()) {
	s.mu.Lock()
	s.persistHook = hook
	s.mu.Unlock()
}

func (s *MemoryStore) afterWrite() {
	if s.persistHook != nil {
		s.persistHook()
	}
}

func appKey(postID, applicantID string) string { return postID + "|" + applicantID }

func reviewKey(taskID, fromUserID string) string { return taskID + "|" + fromUserID }

// Snapshot 供 FileStore 落盘。
type Snapshot struct {
	Version       int                  `json:"version"`
	Users         []model.User         `json:"users"`
	Posts         []model.HelpPost     `json:"posts"`
	Applications  []model.Application  `json:"applications"`
	Tasks         []model.Task         `json:"tasks"`
	Reviews       []model.Review       `json:"reviews"`
	Messages      []model.Message      `json:"messages"`
	Favorites     []model.Favorite     `json:"favorites"`
	Notifications []model.Notification `json:"notifications"`
	Reports       []model.Report       `json:"reports"`
	CreditLogs    []model.CreditLog    `json:"credit_logs"`
}

func mapValues[K comparable, V any](m map[K]V) []V {
	out := make([]V, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

func (s *MemoryStore) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return Snapshot{
		Version:       snapshotVersion,
		Users:         mapValues(s.users),
		Posts:         mapValues(s.posts),
		Applications:  mapValues(s.apps),
		Tasks:         mapValues(s.tasks),
		Reviews:       mapValues(s.reviews),
		Messages:      mapValues(s.messages),
		Favorites:     mapValues(s.favorites),
		Notifications: mapValues(s.notifs),
		Reports:       mapValues(s.reports),
		CreditLogs:    mapValues(s.credits),
	}
}

const snapshotVersion = 1

func (s *MemoryStore) ReplaceAll(snap Snapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users = make(map[string]model.User, len(snap.Users))
	s.username = make(map[string]string, len(snap.Users))
	for _, u := range snap.Users {
		s.users[u.ID] = u
		s.username[u.Username] = u.ID
	}
	s.posts = make(map[string]model.HelpPost, len(snap.Posts))
	for _, p := range snap.Posts {
		s.posts[p.ID] = p
	}
	s.apps = make(map[string]model.Application, len(snap.Applications))
	s.appIdx = make(map[string]string, len(snap.Applications))
	for _, a := range snap.Applications {
		s.apps[a.ID] = a
		s.appIdx[appKey(a.PostID, a.ApplicantID)] = a.ID
	}
	s.tasks = make(map[string]model.Task, len(snap.Tasks))
	s.taskByPost = make(map[string]string, len(snap.Tasks))
	for _, t := range snap.Tasks {
		s.tasks[t.ID] = t
		s.taskByPost[t.PostID] = t.ID
	}
	s.reviews = make(map[string]model.Review, len(snap.Reviews))
	s.reviewIdx = make(map[string]string, len(snap.Reviews))
	for _, r := range snap.Reviews {
		s.reviews[r.ID] = r
		s.reviewIdx[reviewKey(r.TaskID, r.FromUserID)] = r.ID
	}
	s.messages = make(map[string]model.Message, len(snap.Messages))
	for _, m := range snap.Messages {
		s.messages[m.ID] = m
	}
	s.favorites = make(map[string]model.Favorite, len(snap.Favorites))
	s.favIdx = make(map[model.FavoriteKey]string, len(snap.Favorites))
	for _, f := range snap.Favorites {
		s.favorites[f.ID] = f
		s.favIdx[model.FavoriteKey{UserID: f.UserID, PostID: f.PostID}] = f.ID
	}
	s.notifs = make(map[string]model.Notification, len(snap.Notifications))
	for _, n := range snap.Notifications {
		s.notifs[n.ID] = n
	}
	s.reports = make(map[string]model.Report, len(snap.Reports))
	for _, r := range snap.Reports {
		s.reports[r.ID] = r
	}
	s.credits = make(map[string]model.CreditLog, len(snap.CreditLogs))
	for _, c := range snap.CreditLogs {
		s.credits[c.ID] = c
	}
}
