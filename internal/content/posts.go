package content

import (
	"sort"
	"sync"
	"time"
)

type Post struct {
	ID          uint32
	Title       string
	Author      string
	Description string // for SEO
	CreatedAt   time.Time
	ModifiedAt  time.Time
	Draft       bool
	Path        string
	IsSafeHTML  bool
	NoIndex     bool
	Content     []byte
	mu          sync.RWMutex
}

// type Repository interface {
// 	GetAll() []*Post
// 	Get(id uint32) (*Post, error)
// }

type LocalRepository struct {
	title string
	Data  map[uint32]*Post
	mu    sync.RWMutex
}

func NewLocalRepository(title string) (*LocalRepository, error) {
	if title == "" {
		return nil, ErrRepositoryTitle
	}

	return &LocalRepository{
		title: title,
		Data:  make(map[uint32]*Post),
	}, nil
}

func (r *LocalRepository) GetAll() []*Post {
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

func (r *LocalRepository) Get(id uint32) (*Post, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	post, ok := r.Data[id]
	if !ok {
		return nil, ErrPostNotFound
	}
	return post, nil
}

func (r *LocalRepository) Title() string {
	return r.title
}
