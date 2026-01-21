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

func NewRepository(title string) (*Repository, error) {
	if title == "" {
		return nil, ErrRepositoryTitle
	}

	return &Repository{
		title: title,
		Data:  make(map[uint32]*Post),
	}, nil
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

func (r *Repository) Get(id uint32) (*Post, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	post, ok := r.Data[id]
	if !ok {
		return nil, ErrPostNotFound
	}
	return post, nil
}

func (r *Repository) Title() string {
	return r.title
}
