//go:build integration

package sqlite

import (
	"blogengine/internal/storage"
	"context"
	"errors"
	"testing"
)

func TestDeleteCommentCRUD(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)
	defer store.Close()

	ctx := context.Background()

	alice, err := store.CreateUser(ctx, "Alice", gen60CharString())
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	bob, err := store.CreateUser(ctx, "Bob", gen60CharString())
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	var fakePostID int64 = 100
	fakeComment := "Alice's comment here"
	aliceComment, err := store.CreateComment(ctx, fakePostID, alice.ID, fakeComment)
	if err != nil {
		t.Fatalf("failed to create comment for Alice: %v", err)
	}

	if aliceComment.Content != fakeComment {
		t.Fatalf("inconsistent comment: got %v, Want %v", aliceComment.Content, fakeComment)
	}

	// we bring the create_at back one second to facilitate comparison with updated_at
	_, err = store.db.Exec("UPDATE comments SET created_at = datetime(created_at, '-1 minute') WHERE id = ?", aliceComment.ID)
	if err != nil {
		t.Fatalf("failed to manipulate time for test: %v", err)
	}

	fakeCommentV2 := "Alice updated her comment"
	aliceCommentV2, err := store.UpdateComment(ctx, aliceComment.ID, alice.ID, fakeCommentV2)
	if err != nil {
		t.Fatalf("failed to update comment for Alice: %v", err)
	}

	if aliceCommentV2.Content != fakeCommentV2 {
		t.Fatalf("inconsistent comment: got %v, Want %v", aliceComment.Content, fakeComment)
	}

	if aliceCommentV2.UpdatedAt == nil {
		t.Fatalf("expected updated_at to be populated by the trigger")
	}

	if !aliceCommentV2.UpdatedAt.After(aliceCommentV2.CreatedAt) {
		t.Fatalf("expected updated_at to be after created_at: updated %v, created %v", aliceCommentV2.UpdatedAt, aliceCommentV2.CreatedAt)
	}

	// bob tries to update alice's comment - fail
	bobWantsUpdate := "this is Bob and I updated Alice's comment"
	bobsPoison, err := store.UpdateComment(ctx, aliceComment.ID, bob.ID, bobWantsUpdate)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrNotFound):
			// expected: db should not find a comment posted by Bob with this ID
		default:
			// other db error
			t.Fatalf("failed to update comment: %v", err)
		}
	}

	commentInDB, err := store.GetCommentByID(ctx, aliceCommentV2.ID)
	if err != nil {
		t.Fatalf("could not get comment: %v", err)
	}
	if commentInDB.Content != fakeCommentV2 {
		t.Fatalf("unauthorised comment update: want %v, got %v", aliceComment.Content, bobsPoison)
	}

	// bob tries to delete alice's comment - fail
	if err := store.DeleteComment(ctx, aliceCommentV2.ID, bob.ID); err != nil {
		switch {
		case errors.Is(err, storage.ErrNotFound):
			// expected: db should not find a comment posted by Bob with this ID
		default:
			// other db error
			t.Fatalf("failed to delete comment: %v", err)
		}
	}

	// alice's comment should remain intact
	commentInDB, err = store.GetCommentByID(ctx, aliceCommentV2.ID)
	if err != nil {
		t.Fatalf("could not get comment: %v", err)
	}
	if commentInDB.Content != fakeCommentV2 {
		t.Fatalf("unauthorised comment deletion: want %v, got %v", aliceComment.Content, bobsPoison)
	}

	// alice deletes own comment - pass
	if err := store.DeleteComment(ctx, aliceCommentV2.ID, alice.ID); err != nil {
		t.Fatalf("couldn not delete comment: %v", err)
	}

	// should not be able to get Alice's comment again - soft deleted
	_, err = store.GetCommentByID(ctx, aliceCommentV2.ID)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrNotFound):
			// this is expected, should not exist
		default:
			t.Fatalf("could not get comment: %v", err)
		}
	}
}
