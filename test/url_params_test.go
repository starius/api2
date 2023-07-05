package api2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/starius/api2"
	"github.com/starius/api2/errors"
	"github.com/stretchr/testify/require"
)

func TestUrlParams(t *testing.T) {
	// "Database".

	type Post struct {
		Text     string
		Comments []string
	}
	type Comment struct {
		Text      string
		Responses []string
	}
	type User struct {
		Name     string
		Posts    map[string]*Post
		Comments map[string]*Comment
	}
	db := map[string]*User{}

	// Schema.

	type UsersRequest struct {
	}
	type UsersResponse struct {
		Users []string `json:"users"`
	}

	handleUsers := func(ctx context.Context, req *UsersRequest) (res *UsersResponse, err error) {
		users := []string{}
		for userId := range db {
			users = append(users, userId)
		}
		sort.Strings(users)
		return &UsersResponse{
			Users: users,
		}, nil
	}

	type GetUserRequest struct {
		UserId string `url:"user"`
	}
	type GetUserResponse struct {
		Name string `json:"name"`
	}

	handleGetUser := func(ctx context.Context, req *GetUserRequest) (res *GetUserResponse, err error) {
		user, has := db[req.UserId]
		if !has {
			return nil, errors.NotFound("no such user: %s", req.UserId)
		}
		return &GetUserResponse{
			Name: user.Name,
		}, nil
	}

	type CreateUserRequest struct {
		UserId string `url:"user"`
		Name   []byte `use_as_body:"true" is_raw:"true"`
	}
	type CreateUserResponse struct {
	}

	handleCreateUser := func(ctx context.Context, req *CreateUserRequest) (res *CreateUserResponse, err error) {
		_, has := db[req.UserId]
		if has {
			return nil, errors.AlreadyExists("user already exists: %s", req.UserId)
		}
		db[req.UserId] = &User{
			Name:     string(req.Name),
			Posts:    make(map[string]*Post),
			Comments: make(map[string]*Comment),
		}
		return &CreateUserResponse{}, nil
	}

	type PostsOfUserRequest struct {
		UserId string `url:"user"`
	}
	type PostsOfUserResponse struct {
		PostIds []string `json:"posts"`
	}

	handlePostsOfUser := func(ctx context.Context, req *PostsOfUserRequest) (res *PostsOfUserResponse, err error) {
		user, has := db[req.UserId]
		if !has {
			return nil, errors.NotFound("no such user: %s", req.UserId)
		}
		posts := []string{}
		for postId := range user.Posts {
			posts = append(posts, postId)
		}
		sort.Strings(posts)
		return &PostsOfUserResponse{
			PostIds: posts,
		}, nil
	}

	type GetPostRequest struct {
		UserId string `url:"user"`
		PostId string `url:"post"`
	}
	type GetPostResponse struct {
		PostText string `json:"post_text"`
	}

	handleGetPost := func(ctx context.Context, req *GetPostRequest) (res *GetPostResponse, err error) {
		user, has := db[req.UserId]
		if !has {
			return nil, errors.NotFound("no such user: %s", req.UserId)
		}
		post, has := user.Posts[req.PostId]
		if !has {
			return nil, errors.NotFound("no such post: %s", req.PostId)
		}
		return &GetPostResponse{
			PostText: post.Text,
		}, nil
	}

	type CreatePostRequest struct {
		UserId   string `url:"user"`
		PostId   string `url:"post"`
		PostText string `json:"post_text"`
	}
	type CreatePostResponse struct {
	}

	handleCreatePost := func(ctx context.Context, req *CreatePostRequest) (res *CreatePostResponse, err error) {
		user, has := db[req.UserId]
		if !has {
			return nil, errors.NotFound("no such user: %s", req.UserId)
		}
		_, has = user.Posts[req.PostId]
		if has {
			return nil, errors.AlreadyExists("post already exists: %s", req.PostId)
		}
		user.Posts[req.PostId] = &Post{
			Text: req.PostText,
		}
		return &CreatePostResponse{}, nil
	}

	type GetCommentsRequest struct {
		UserId string `url:"user"`
		PostId string `url:"post"`
	}
	type GetCommentsResponse struct {
		Comments []string `json:"comments"`
	}

	handleGetComments := func(ctx context.Context, req *GetCommentsRequest) (res *GetCommentsResponse, err error) {
		user, has := db[req.UserId]
		if !has {
			return nil, errors.NotFound("no such user: %s", req.UserId)
		}
		post, has := user.Posts[req.PostId]
		if !has {
			return nil, errors.NotFound("no such post: %s", req.PostId)
		}
		return &GetCommentsResponse{
			Comments: post.Comments,
		}, nil
	}

	type CreateCommentRequest struct {
		UserId      string `url:"user"`
		PostId      string `url:"post"`
		CommentId   string `json:"comment_id"`
		CommentText string `json:"comment_text"`
	}
	type CreateCommentResponse struct {
	}

	handleCreateComment := func(ctx context.Context, req *CreateCommentRequest) (res *CreateCommentResponse, err error) {
		user, has := db[req.UserId]
		if !has {
			return nil, errors.NotFound("no such user: %s", req.UserId)
		}
		post, has := user.Posts[req.PostId]
		if !has {
			return nil, errors.NotFound("no such post: %s", req.PostId)
		}
		_, has = user.Comments[req.CommentId]
		if has {
			return nil, errors.AlreadyExists("comment already exists: %s", req.CommentId)
		}
		post.Comments = append(post.Comments, req.CommentId)
		user.Comments[req.CommentId] = &Comment{
			Text: req.CommentText,
		}
		return &CreateCommentResponse{}, nil
	}

	type CommentsOfUserRequest struct {
		UserId string `url:"user"`
	}
	type CommentsOfUserResponse struct {
		CommentIds []string `json:"comments"`
	}

	handleCommentsOfUser := func(ctx context.Context, req *CommentsOfUserRequest) (res *CommentsOfUserResponse, err error) {
		user, has := db[req.UserId]
		if !has {
			return nil, errors.NotFound("no such user: %s", req.UserId)
		}
		commentIds := []string{}
		for commentId := range user.Comments {
			commentIds = append(commentIds, commentId)
		}
		return &CommentsOfUserResponse{
			CommentIds: commentIds,
		}, nil
	}

	type GetCommentRequest struct {
		UserId    string `url:"user"`
		CommentId string `url:"comment"`
	}
	type GetCommentResponse struct {
		CommentText []byte `use_as_body:"true" is_raw:"true"`
	}

	handleGetComment := func(ctx context.Context, req *GetCommentRequest) (res *GetCommentResponse, err error) {
		user, has := db[req.UserId]
		if !has {
			return nil, errors.NotFound("no such user: %s", req.UserId)
		}
		comment, has := user.Comments[req.CommentId]
		if !has {
			return nil, errors.NotFound("no such comment: %s", req.CommentId)
		}
		return &GetCommentResponse{
			CommentText: []byte(comment.Text),
		}, nil
	}

	type CreateResponseRequest struct {
		UserId       string `url:"user"`
		CommentId    string `url:"comment"`
		ResponseText string `json:"response_text"`
	}
	type CreateResponseResponse struct {
	}

	handleCreateResponse := func(ctx context.Context, req *CreateResponseRequest) (res *CreateResponseResponse, err error) {
		user, has := db[req.UserId]
		if !has {
			return nil, errors.NotFound("no such user: %s", req.UserId)
		}
		comment, has := user.Comments[req.CommentId]
		if !has {
			return nil, errors.NotFound("no such comment: %s", req.CommentId)
		}
		comment.Responses = append(comment.Responses, req.ResponseText)
		return &CreateResponseResponse{}, nil
	}

	type GetCommentResponsesRequest struct {
		UserId    string `url:"user"`
		CommentId string `url:"comment"`
	}
	type GetCommentResponsesResponse struct {
		CommentResponses []string `json:"comment_responses"`
	}

	handleGetCommentResponses := func(ctx context.Context, req *GetCommentResponsesRequest) (res *GetCommentResponsesResponse, err error) {
		user, has := db[req.UserId]
		if !has {
			return nil, errors.NotFound("no such user: %s", req.UserId)
		}
		comment, has := user.Comments[req.CommentId]
		if !has {
			return nil, errors.NotFound("no such comment: %s", req.CommentId)
		}
		return &GetCommentResponsesResponse{
			CommentResponses: comment.Responses,
		}, nil
	}

	route := func(method, path string, handler interface{}) api2.Route {
		return api2.Route{
			Method:  method,
			Path:    path,
			Handler: handler,
		}
	}

	routes := []api2.Route{
		route(http.MethodGet, "/users", handleUsers),
		route(http.MethodGet, "/users/:user", handleGetUser),
		route(http.MethodPost, "/users/:user", handleCreateUser),
		route(http.MethodGet, "/users/:user/posts", handlePostsOfUser),
		route(http.MethodGet, "/users/:user/posts/:post", handleGetPost),
		route(http.MethodPost, "/users/:user/posts/:post", handleCreatePost),
		route(http.MethodGet, "/users/:user/posts/:post/comments", handleGetComments),
		route(http.MethodPost, "/users/:user/posts/:post/comments", handleCreateComment),
		route(http.MethodGet, "/users/:user/comments", handleCommentsOfUser),
		route(http.MethodGet, "/users/:user/comments/:comment", handleGetComment),
		route(http.MethodPost, "/users/:user/comments/:comment/responses", handleCreateResponse),
		route(http.MethodGet, "/users/:user/comments/:comment/responses", handleGetCommentResponses),
	}

	mux := http.NewServeMux()
	api2.BindRoutes(mux, routes)
	server := httptest.NewServer(mux)
	defer server.Close()

	client := api2.NewClient(routes, server.URL)
	ctx := context.Background()

	// Testing.

	t.Run("create users", func(t *testing.T) {
		var res CreateUserResponse
		require.NoError(t, client.Call(ctx, &res, &CreateUserRequest{
			UserId: "user1",
			Name:   []byte("Boris"),
		}))
		require.NoError(t, client.Call(ctx, &res, &CreateUserRequest{
			UserId: "user2",
			Name:   []byte("Petr"),
		}))
	})

	t.Run("get users", func(t *testing.T) {
		var res UsersResponse
		require.NoError(t, client.Call(ctx, &res, &UsersRequest{}))
		require.Equal(t, []string{"user1", "user2"}, res.Users)
	})

	t.Run("get user info", func(t *testing.T) {
		var res GetUserResponse
		require.NoError(t, client.Call(ctx, &res, &GetUserRequest{
			UserId: "user1",
		}))
		require.Equal(t, "Boris", res.Name)
		require.NoError(t, client.Call(ctx, &res, &GetUserRequest{
			UserId: "user2",
		}))
		require.Equal(t, "Petr", res.Name)
	})

	t.Run("create post", func(t *testing.T) {
		var res CreatePostResponse
		require.NoError(t, client.Call(ctx, &res, &CreatePostRequest{
			UserId:   "user1",
			PostId:   "post1",
			PostText: "Hello!",
		}))
	})

	t.Run("lists posts of user", func(t *testing.T) {
		var res PostsOfUserResponse
		require.NoError(t, client.Call(ctx, &res, &PostsOfUserRequest{
			UserId: "user1",
		}))
		require.Equal(t, []string{"post1"}, res.PostIds)
	})

	t.Run("get post", func(t *testing.T) {
		var res GetPostResponse
		require.NoError(t, client.Call(ctx, &res, &GetPostRequest{
			UserId: "user1",
			PostId: "post1",
		}))
		require.Equal(t, "Hello!", res.PostText)
	})

	t.Run("create comment", func(t *testing.T) {
		var res CreateCommentResponse
		require.NoError(t, client.Call(ctx, &res, &CreateCommentRequest{
			UserId:      "user1",
			PostId:      "post1",
			CommentId:   "comment1",
			CommentText: "Thank you",
		}))
	})

	t.Run("list comments of post of user", func(t *testing.T) {
		var res GetCommentsResponse
		require.NoError(t, client.Call(ctx, &res, &GetCommentsRequest{
			UserId: "user1",
			PostId: "post1",
		}))
		require.Equal(t, []string{"comment1"}, res.Comments)
	})

	t.Run("list comments of user", func(t *testing.T) {
		var res CommentsOfUserResponse
		require.NoError(t, client.Call(ctx, &res, &CommentsOfUserRequest{
			UserId: "user1",
		}))
		require.Equal(t, []string{"comment1"}, res.CommentIds)
	})

	t.Run("get comment text", func(t *testing.T) {
		var res GetCommentResponse
		require.NoError(t, client.Call(ctx, &res, &GetCommentRequest{
			UserId:    "user1",
			CommentId: "comment1",
		}))
		require.Equal(t, []byte("Thank you"), res.CommentText)
	})

	t.Run("create response for comment", func(t *testing.T) {
		var res CreateResponseResponse
		require.NoError(t, client.Call(ctx, &res, &CreateResponseRequest{
			UserId:       "user1",
			CommentId:    "comment1",
			ResponseText: "Welcome",
		}))
	})

	t.Run("list responses for comment", func(t *testing.T) {
		var res GetCommentResponsesResponse
		require.NoError(t, client.Call(ctx, &res, &GetCommentResponsesRequest{
			UserId:    "user1",
			CommentId: "comment1",
		}))
		require.Equal(t, []string{"Welcome"}, res.CommentResponses)
	})
}
