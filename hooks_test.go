package s3mock

import (
	"testing"
)

func TestOnPutHook(t *testing.T) {
	m := New()
	m.CreateBucket(ctx, "b1")

	var events []string
	m.OnPut(func(bucket, key string) {
		events = append(events, bucket+"/"+key)
	})

	m.PutObject(ctx, "b1", "a.txt", []byte("a"), "text/plain", nil)
	m.PutObject(ctx, "b1", "b.txt", []byte("b"), "text/plain", nil)

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0] != "b1/a.txt" || events[1] != "b1/b.txt" {
		t.Fatalf("unexpected events: %v", events)
	}
}

func TestOnDeleteHook(t *testing.T) {
	m := New()
	m.CreateBucket(ctx, "b1")
	m.PutObject(ctx, "b1", "f.txt", []byte("data"), "text/plain", nil)

	var deleted []string
	m.OnDelete(func(bucket, key string) {
		deleted = append(deleted, bucket+"/"+key)
	})

	m.DeleteObject(ctx, "b1", "f.txt")

	if len(deleted) != 1 || deleted[0] != "b1/f.txt" {
		t.Fatalf("unexpected deleted events: %v", deleted)
	}
}

func TestMultipleHooks(t *testing.T) {
	m := New()
	m.CreateBucket(ctx, "b1")

	count1 := 0
	count2 := 0
	m.OnPut(func(_, _ string) { count1++ })
	m.OnPut(func(_, _ string) { count2++ })

	m.PutObject(ctx, "b1", "f.txt", []byte("data"), "text/plain", nil)

	if count1 != 1 || count2 != 1 {
		t.Fatalf("expected both hooks to fire: count1=%d count2=%d", count1, count2)
	}
}

func TestTagging(t *testing.T) {
	m := New()
	m.CreateBucket(ctx, "b1")
	m.PutObject(ctx, "b1", "f.txt", []byte("data"), "text/plain", nil)

	// Set tags
	tags := map[string]string{"env": "prod", "team": "backend"}
	if err := m.PutObjectTagging(ctx, "b1", "f.txt", tags); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Get tags
	got, err := m.GetObjectTagging(ctx, "b1", "f.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["env"] != "prod" || got["team"] != "backend" {
		t.Fatalf("tag mismatch: %v", got)
	}

	// Delete tags
	if err := m.DeleteObjectTagging(ctx, "b1", "f.txt"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, _ = m.GetObjectTagging(ctx, "b1", "f.txt")
	if len(got) != 0 {
		t.Fatalf("expected 0 tags after delete, got %d", len(got))
	}
}

func TestTaggingNotFound(t *testing.T) {
	m := New()
	m.CreateBucket(ctx, "b1")

	err := m.PutObjectTagging(ctx, "b1", "nope", map[string]string{"k": "v"})
	if !IsNotFound(err) {
		t.Fatalf("expected NotFound, got: %v", err)
	}

	_, err = m.GetObjectTagging(ctx, "nope", "f.txt")
	if !IsNotFound(err) {
		t.Fatalf("expected NotFound, got: %v", err)
	}
}

func TestHelpers(t *testing.T) {
	m := New()
	m.CreateBucket(ctx, "b1")
	m.PutObject(ctx, "b1", "f.txt", []byte("hello"), "text/plain", nil)

	m.AssertBucketExists(t, "b1")
	m.AssertObjectExists(t, "b1", "f.txt")
	m.AssertObjectNotExists(t, "b1", "nope.txt")
	m.AssertObjectContent(t, "b1", "f.txt", []byte("hello"))

	if m.ObjectCount("b1") != 1 {
		t.Fatalf("expected 1 object, got %d", m.ObjectCount("b1"))
	}

	m.CreateBucket(ctx, "empty")
	m.AssertBucketEmpty(t, "empty")
}

func TestReset(t *testing.T) {
	m := New()
	m.CreateBucket(ctx, "b1")
	m.PutObject(ctx, "b1", "f.txt", []byte("data"), "text/plain", nil)

	m.Reset()

	buckets := m.ListBuckets(ctx)
	if len(buckets) != 0 {
		t.Fatalf("expected 0 buckets after reset, got %d", len(buckets))
	}
}
