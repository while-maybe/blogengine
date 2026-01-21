package content

import (
	"sort"
	"sync"
	"time"
)

type Post struct {
	ID         uint32
	Author     int
	Title      string
	CreatedAt  time.Time
	ModifiedAt time.Time
	Path       string
	IsSafeHTML bool
	Content    []byte
	mu         sync.RWMutex
}

type Repository struct {
	title string
	Data  map[uint32]*Post
	mu    sync.RWMutex
}

func NewPosts(title string) *Repository {
	return &Repository{
		title: title,
		Data:  make(map[uint32]*Post),
	}
}

func (r *Repository) GetAll() []*Post {
	r.mu.RLock()

	postsList := make([]*Post, 0, len(r.Data))

	for _, post := range r.Data {
		postsList = append(postsList, post)
	}

	r.mu.RUnlock()

	// sort chronologically
	sort.Slice(postsList, func(i, j int) bool {
		return postsList[i].ModifiedAt.After(postsList[j].ModifiedAt)
	})

	return postsList
}

func (r *Repository) Get(id uint32) (*Post, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	post, ok := r.Data[id]
	return post, ok
}

func (r *Repository) Title() string {
	return r.title
}
