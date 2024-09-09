-- name: CreatePost :one
INSERT INTO posts (id, created_at, updated_at, title, url, description, published_at, feed_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetPostsForUser :many
SELECT count(*) OVER(), posts.id, posts.created_at, posts.updated_at, posts.title, posts.url, posts.description, posts.published_at, posts.feed_id
FROM posts
JOIN feed_follows ON feed_follows.feed_id = posts.feed_id
WHERE feed_follows.user_id = @user_id::uuid
  AND (to_tsvector('simple', posts.title) @@ plainto_tsquery('simple', @title::text) OR @title::text = '')
  AND (posts.feed_id = @feed_id::uuid OR @feed_id::uuid = '00000000-0000-0000-0000-000000000000'::uuid)
ORDER BY 
  CASE 
    WHEN @sort = 'id' THEN posts.id END ASC,
    CASE 
    WHEN @sort = 'title' THEN posts.title END ASC,
    CASE 
    WHEN @sort = 'published_at' THEN posts.published_at END ASC,
    CASE 
    WHEN @sort = '-id' THEN posts.id END DESC,
    CASE 
    WHEN @sort = '-title' THEN posts.title END DESC,
    CASE 
    WHEN @sort = '-published_at' THEN posts.published_at END DESC,
  posts.id ASC
  LIMIT @lim::integer OFFSET @off::integer;

    
