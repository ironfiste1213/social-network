# Backend Endpoints and Handlers

This file documents:
- all HTTP endpoints currently wired in `backend/server.go`
- the handler method that serves each endpoint
- the relation between handler packages

## Router Entry Point

All routes are registered in:
- `backend/server.go`

The server creates one `http.ServeMux` and registers package handlers in this order:
1. `auth.Handler`
2. `users.Handler`
3. `followers.Handler`
4. `posts.Handler`
5. `groups.Handler`

It also wires cross-package delegation:
- `usersHandler.SetFollowersHandler(followersHandler)`
- `usersHandler.SetPostsHandler(postsHandler)`
- `postsHandler.SetCommentsHandler(commentsHandler)`

## Endpoint -> Handler Mapping

### Auth package (`backend/pkg/auth/handler.go`)

- `POST /auth/register` -> `(*auth.Handler).handleRegister`
- `POST /auth/login` -> `(*auth.Handler).handleLogin`
- `POST /auth/logout` -> `(*auth.Handler).handleLogout`
- `GET /auth/me` -> `(*auth.Handler).handleMe`

### Users package (`backend/pkg/users/handler.go`)

- `GET /users/me` -> `(*users.Handler).handleMe`
- `PATCH /users/me` -> `(*users.Handler).handleMe` -> `handleUpdateMe`
- `POST /users/me/avatar` -> `(*users.Handler).handleAvatar`
- `GET /users/search` -> `(*users.Handler).handleSearchUsers`
- `GET /users/{id}` -> `(*users.Handler).handleUserByID`

Delegated subroutes inside `handleUserByID`:
- `GET /users/{id}/followers` -> delegated to `followers.Handler.HandleUserFollowRoutes`
- `GET /users/{id}/following` -> delegated to `followers.Handler.HandleUserFollowRoutes`
- `GET /users/{id}/follow-status` -> delegated to `followers.Handler.HandleUserFollowRoutes`
- `GET /users/{id}/posts` -> delegated to `posts.Handler.HandleUserPostRoutes`

### Followers package (`backend/pkg/followers/handler.go`)

Directly registered routes:
- `POST /follow/{targetID}` -> `(*followers.Handler).routeFollow` -> `handleFollow` (initiates follow or sends follow request)
- `DELETE /follow/{targetID}` -> `(*followers.Handler).routeFollow` -> `handleFollow` (unfollows OR cancels pending outgoing follow request)
- `GET /follow/requests` -> `(*followers.Handler).handleListRequests` (lists incoming follow requests for receiver)
- `POST /follow/requests/{requestID}/accept` -> `(*followers.Handler).handleRespondRequest` (receiver accepts incoming request)
- `POST /follow/requests/{requestID}/decline` -> `(*followers.Handler).handleRespondRequest` (receiver declines incoming request)
- `POST /follow/requests/{requestID}/cancel` -> `(*followers.Handler).handleRespondRequest` (sender cancels pending outgoing request)

Also used by `users` as delegated handler for `/users/{id}/followers|following|follow-status`.

### Posts package (`backend/pkg/posts/handler.go`)

Registered routes:
- `POST /posts` -> `(*posts.Handler).handlePosts` -> `createPost`
- `GET /posts` -> `(*posts.Handler).handlePosts` -> `getFeed`
- `GET /posts/{id}` -> `(*posts.Handler).handlePostByID` -> `getPost`
- `DELETE /posts/{id}` -> `(*posts.Handler).handlePostByID` -> `deletePost`
- `POST /posts/{id}/image` -> `(*posts.Handler).handlePostByID` -> `uploadImage`
- `GET /posts/my-followers` -> `(*posts.Handler).GetMyFollowers` (registered in `server.go`)

Delegated route from users package:
- `GET /users/{id}/posts` -> `(*posts.Handler).HandleUserPostRoutes` -> `getUserPosts`

Delegated comments subroutes:
- `/posts/{postID}/comments...` handled through `comments.Handler.HandlePostSubroute`

### Comments package (`backend/pkg/comments/handler.go`)

Comments handler is **not directly registered** in `server.go`.
It is attached to posts handler through `postsHandler.SetCommentsHandler(commentsHandler)`.

Effective endpoints (served via posts delegation):
- `GET /posts/{postID}/comments` -> `(*comments.Handler).HandlePostSubroute` -> `listComments`
- `POST /posts/{postID}/comments` -> `(*comments.Handler).HandlePostSubroute` -> `createComment`
- `DELETE /posts/{postID}/comments/{commentID}` -> `(*comments.Handler).HandlePostSubroute` -> `deleteComment`
- `POST /posts/{postID}/comments/{commentID}/image` -> `(*comments.Handler).HandlePostSubroute` -> `uploadImage`

### Groups package (`backend/pkg/groups/handler.go`)

- `GET /groups` -> `(*groups.Handler).handleGroups` -> `listGroups`
- `POST /groups` -> `(*groups.Handler).handleGroups` -> `createGroup`
- `GET /groups/{groupID}` -> `(*groups.Handler).handleGroupRoutes` -> `getGroup`
- `POST /groups/{groupID}/join` -> `(*groups.Handler).handleGroupRoutes` -> `requestJoin`
- `GET /groups/{groupID}/requests` -> `(*groups.Handler).handleGroupRoutes` -> `listJoinRequests`
- `POST /groups/requests/{requestID}/accept` -> `(*groups.Handler).handleGroupRoutes` -> `respondJoinRequest`
- `POST /groups/requests/{requestID}/decline` -> `(*groups.Handler).handleGroupRoutes` -> `respondJoinRequest`
- `POST /groups/{groupID}/invite` -> `(*groups.Handler).handleGroupRoutes` -> `inviteToGroup`
- `GET /groups/invitations` -> `(*groups.Handler).handleGroupRoutes` -> `listInvitations`
- `POST /groups/invitations/{invitationID}/accept` -> `(*groups.Handler).handleGroupRoutes` -> `respondInvitation`
- `POST /groups/invitations/{invitationID}/decline` -> `(*groups.Handler).handleGroupRoutes` -> `respondInvitation`
- `GET /groups/{groupID}/members` -> `(*groups.Handler).handleGroupRoutes` -> `listMembers`

Note:
- `server.go` comments mention group posts/events, but those routes are not present in current `groups` handler code.

### Other server-level routes

- `GET /health` -> inline handler in `server.go`
- `GET /uploads/*` -> static file handler from `users.ServeUploads(uploadDir)`

## Relations Between Package Handlers

High-level relations:

- `main/server.go` is the composition root; it creates and wires all handlers.
- `users.Handler` delegates some `/users/{id}/...` subpaths to other packages:
  - followers package for follow-related user subroutes
  - posts package for user posts subroute
- `posts.Handler` delegates post comments subpaths to `comments.Handler`.
- `groups.Handler` and `auth.Handler` are self-contained route handlers (no cross-handler delegation).

Call/delegation flow diagram:

`server.go`
-> `users.Handler.handleUserByID`
-> `followers.Handler.HandleUserFollowRoutes` (for `followers|following|follow-status`)

`server.go`
-> `users.Handler.handleUserByID`
-> `posts.Handler.HandleUserPostRoutes` (for `posts`)

`server.go`
-> `posts.Handler.handlePostByID`
-> `comments.Handler.HandlePostSubroute` (for `comments...`)

## Quick Notes

- Most handlers use shared `session_id` cookie auth helpers from `backend/pkg/sessionauth`.
- Route matching is based on `http.ServeMux` and manual path parsing in handlers.
- JSON and error responses are centralized in `backend/pkg/response` (`response.JSON` / `response.Error`).
